# Solution Summary

## Approach
I implemented a lightweight prediction markets feature that fits into the existing trading platform without requiring a database migration. The Node backend now exposes prediction-market REST endpoints and WebSocket updates, while a small Go microservice provides an AMM-style pricing model for market probability updates.

## Design decisions
- Kept the feature in-memory for fast local testing and demonstration.
- Reused the existing Express server and static frontend structure so the feature is immediately accessible.
- Added a simple Go service to model prediction-market pricing logic and real-time updates.
- Used the same browser UI flow that the platform already uses, with a dedicated prediction create/trade/settle experience.

## Trade-offs
- The implementation uses in-memory storage rather than PostgreSQL for simplicity and portability.
- The AMM logic is intentionally lightweight and deterministic rather than a full production-grade market maker.
- Authentication is handled through the existing demo auth flow rather than a separate user-management system.

## How to test
1. Install dependencies:
   ```bash
   npm run install:all
   ```
2. Start the full stack:
   ```bash
   npm run dev
   ```
3. Open the frontend and log in with the demo account:
   - Email: test@example.com
   - Password: password123
4. Create a prediction market, trade on it, and settle it from the UI.

## What I would improve with more time
- Replace in-memory storage with a real database and persistence layer.
- Add proper user balances, positions, and payout handling.
- Expand the Go service with more robust AMM logic and persistent WebSocket channels.
- Add automated integration tests across the API and UI.
