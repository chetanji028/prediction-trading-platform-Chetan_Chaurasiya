package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type MarketStore struct {
	mu      sync.RWMutex
	markets map[string]*PredictionMarket
	clients map[*websocket.Conn]bool
	upgrader websocket.Upgrader
}

func NewMarketStore() *MarketStore {
	return &MarketStore{
		markets: make(map[string]*PredictionMarket),
		clients: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
	}
}

func (s *MarketStore) seed() {
	market := &PredictionMarket{
		ID:            uuid.NewString(),
		Question:      "Will ETH reach $5000 by end of 2026?",
		Description:   "Seeded AMM demo market",
		YesShares:     100,
		NoShares:      100,
		LiquidityPool: 0,
		ExpiryDate:    time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339),
		Status:        "active",
	}
	s.mu.Lock()
	s.markets[market.ID] = market
	s.mu.Unlock()
}

func calculatePrice(yesShares, noShares float64) (yesPrice, noPrice float64) {
	total := yesShares + noShares
	if total <= 0 {
		return 0.5, 0.5
	}
	return yesShares / total, noShares / total
}

func calculateCost(yesShares, noShares float64, outcome string, shares int) float64 {
	if shares <= 0 {
		return 0
	}
	if outcome == "Yes" {
		return noShares * float64(shares) / (yesShares + float64(shares))
	}
	return yesShares * float64(shares) / (noShares + float64(shares))
}

func executeTrade(market *PredictionMarket, outcome string, shares int) (cost float64, err error) {
	if shares <= 0 {
		return 0, fmt.Errorf("shares must be greater than zero")
	}
	if outcome != "Yes" && outcome != "No" {
		return 0, fmt.Errorf("outcome must be Yes or No")
	}
	if market.Status != "active" {
		return 0, fmt.Errorf("market is no longer active")
	}
	if expiry, err := time.Parse(time.RFC3339, market.ExpiryDate); err == nil && expiry.Before(time.Now()) {
		market.Status = "expired"
		return 0, fmt.Errorf("market is expired")
	}
	cost = calculateCost(market.YesShares, market.NoShares, outcome, shares)
	if outcome == "Yes" {
		market.YesShares += float64(shares)
		market.NoShares -= cost
	} else {
		market.NoShares += float64(shares)
		market.YesShares -= cost
	}
	market.LiquidityPool += cost
	return cost, nil
}

func (s *MarketStore) handleList(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	markets := make([]*PredictionMarket, 0, len(s.markets))
	for _, market := range s.markets {
		markets = append(markets, market)
	}
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, markets)
}

func (s *MarketStore) handleGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	s.mu.RLock()
	market, ok := s.markets[id]
	s.mu.RUnlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "market not found"})
		return
	}
	writeJSON(w, http.StatusOK, market)
}

func (s *MarketStore) handleCreate(w http.ResponseWriter, r *http.Request) {
	var payload PredictionMarket
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	payload.ID = uuid.NewString()
	payload.Status = "active"
	payload.YesShares = 100
	payload.NoShares = 100
	if payload.ExpiryDate == "" {
		payload.ExpiryDate = time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339)
	}
	s.mu.Lock()
	s.markets[payload.ID] = &payload
	s.mu.Unlock()
	s.broadcast(map[string]interface{}{"type": "market_created", "payload": payload})
	writeJSON(w, http.StatusCreated, payload)
}

func (s *MarketStore) handleTrade(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	var req TradeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	s.mu.Lock()
	market, ok := s.markets[id]
	if !ok {
		s.mu.Unlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "market not found"})
		return
	}
	cost, err := executeTrade(market, req.Outcome, req.Shares)
	s.mu.Unlock()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	s.broadcast(map[string]interface{}{"type": "market_traded", "payload": map[string]interface{}{"marketID": id, "cost": cost}})
	writeJSON(w, http.StatusOK, map[string]interface{}{"marketID": id, "cost": cost})
}

func (s *MarketStore) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade failed", err)
		return
	}
	defer conn.Close()
	s.mu.Lock()
	s.clients[conn] = true
	s.mu.Unlock()
	s.mu.RLock()
	for _, market := range s.markets {
		yesPrice, noPrice := calculatePrice(market.YesShares, market.NoShares)
		payload := map[string]interface{}{"type": "price_update", "payload": map[string]interface{}{"marketID": market.ID, "yesPrice": yesPrice, "noPrice": noPrice}}
		msg, _ := json.Marshal(payload)
		_ = conn.WriteMessage(websocket.TextMessage, msg)
	}
	s.mu.RUnlock()
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
	s.mu.Lock()
	delete(s.clients, conn)
	s.mu.Unlock()
}

func (s *MarketStore) broadcast(payload interface{}) {
	msg, err := json.Marshal(payload)
	if err != nil {
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for client := range s.clients {
		_ = client.WriteMessage(websocket.TextMessage, msg)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func main() {
	store := NewMarketStore()
	store.seed()
	r := mux.NewRouter()
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}).Methods(http.MethodGet)
	r.HandleFunc("/api/markets", store.handleList).Methods(http.MethodGet)
	r.HandleFunc("/api/markets", store.handleCreate).Methods(http.MethodPost)
	r.HandleFunc("/api/markets/{id}", store.handleGet).Methods(http.MethodGet)
	r.HandleFunc("/api/markets/{id}/trade", store.handleTrade).Methods(http.MethodPost)
	r.HandleFunc("/ws", store.handleWS)

	log.Println("prediction service listening on :8082")
	log.Fatal(http.ListenAndServe(":8082", r))
}
