const { v4: uuidv4 } = require('uuid');

function createPrediction({ question, description, createdBy, expiryDate }) {
  const market = {
    id: uuidv4(),
    question,
    description,
    outcomes: ['Yes', 'No'],
    currentPrices: { Yes: 0.5, No: 0.5 },
    totalLiquidity: 0,
    volume24h: 0,
    expiryDate: expiryDate || new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
    settledAt: null,
    winningOutcome: null,
    createdBy,
    createdAt: new Date().toISOString(),
    status: 'active',
    trades: [],
  };

  return market;
}

function tradePrediction(market, userId, outcome, shares) {
  const normalizedShares = Number(shares) || 0;
  if (!market || normalizedShares <= 0 || !['Yes', 'No'].includes(outcome)) {
    throw new Error('Invalid trade request');
  }

  const pricePerShare = market.currentPrices[outcome];
  const totalCost = normalizedShares * pricePerShare;

  market.totalLiquidity += totalCost;
  market.volume24h += totalCost;
  market.trades.push({
    id: uuidv4(),
    predictionId: market.id,
    userId,
    outcome,
    shares: normalizedShares,
    pricePerShare,
    totalCost,
    timestamp: new Date().toISOString(),
    status: 'open',
  });

  const impact = Math.min(0.2, normalizedShares / 100);
  if (outcome === 'Yes') {
    market.currentPrices.Yes = Math.min(0.99, market.currentPrices.Yes + impact);
  } else {
    market.currentPrices.Yes = Math.max(0.01, market.currentPrices.Yes - impact);
  }
  market.currentPrices.No = 1 - market.currentPrices.Yes;

  return market;
}

function settlePrediction(market, winningOutcome) {
  if (!market || !['Yes', 'No'].includes(winningOutcome)) {
    throw new Error('Invalid settlement');
  }

  market.status = 'settled';
  market.settledAt = new Date().toISOString();
  market.winningOutcome = winningOutcome;
  market.currentPrices.Yes = winningOutcome === 'Yes' ? 1 : 0;
  market.currentPrices.No = winningOutcome === 'No' ? 1 : 0;

  return market;
}

module.exports = {
  createPrediction,
  tradePrediction,
  settlePrediction,
};
