package main

type PredictionMarket struct {
	ID            string  `json:"id"`
	Question      string  `json:"question"`
	Description   string  `json:"description"`
	YesShares     float64 `json:"yesShares"`
	NoShares      float64 `json:"noShares"`
	LiquidityPool float64 `json:"liquidityPool"`
	ExpiryDate    string  `json:"expiryDate"`
	Status        string  `json:"status"`
}

type TradeRequest struct {
	Outcome string `json:"outcome"`
	Shares  int    `json:"shares"`
}
