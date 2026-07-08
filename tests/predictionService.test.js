const test = require('node:test');
const assert = require('node:assert/strict');
const { createPrediction, tradePrediction, settlePrediction } = require('../predictionService');

test('createPrediction initializes a balanced market', () => {
  const market = createPrediction({
    question: 'Will ETH reach $5000?',
    description: 'A test prediction market',
    createdBy: 'user-1',
    expiryDate: '2026-12-31T00:00:00.000Z',
  });

  assert.equal(market.status, 'active');
  assert.equal(market.currentPrices.Yes, 0.5);
  assert.equal(market.currentPrices.No, 0.5);
  assert.equal(market.totalLiquidity, 0);
});

test('tradePrediction updates prices and history', () => {
  const market = createPrediction({
    question: 'Will BTC reach $100k?',
    description: 'A test prediction market',
    createdBy: 'user-1',
    expiryDate: '2026-12-31T00:00:00.000Z',
  });

  const afterTrade = tradePrediction(market, 'user-2', 'Yes', 10);

  assert.equal(afterTrade.trades.length, 1);
  assert.equal(afterTrade.currentPrices.Yes > 0.5, true);
  assert.equal(afterTrade.totalLiquidity > 0, true);
  assert.equal(afterTrade.volume24h > 0, true);
});

test('settlePrediction marks a market settled', () => {
  const market = createPrediction({
    question: 'Will DOGE moon?',
    description: 'A test prediction market',
    createdBy: 'user-1',
    expiryDate: '2026-12-31T00:00:00.000Z',
  });

  const settled = settlePrediction(market, 'Yes');

  assert.equal(settled.status, 'settled');
  assert.equal(settled.winningOutcome, 'Yes');
  assert.equal(settled.currentPrices.Yes, 1);
  assert.equal(settled.currentPrices.No, 0);
});
