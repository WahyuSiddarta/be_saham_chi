package repository

import (
	"errors"
	"math"
	"testing"
)

func TestCalculateGoldMovingAverage(t *testing.T) {
	result, err := calculateGoldMovingAverage([]goldReplayTransaction{
		{Kind: GoldTransactionBuy, Quantity: 10, Gross: 15_000_000, Fee: 10_000, Tax: 5_000},
		{Kind: GoldTransactionBuy, Quantity: 10, Gross: 17_000_000, Fee: 10_000, Tax: 5_000},
		{Kind: GoldTransactionSell, Quantity: 4, Gross: 7_000_000, Fee: 10_000, Tax: 5_000},
	})
	if err != nil {
		t.Fatal(err)
	}
	wantAverage := 1_601_500.0
	assertGoldFloat(t, "quantity", result.Quantity, 16)
	assertGoldFloat(t, "average cost", result.AverageCost, wantAverage)
	assertGoldFloat(t, "total cost", result.TotalCost, 16*wantAverage)
	assertGoldFloat(t, "removed cost", result.Transactions[2].CostAmount, 4*wantAverage)
	assertGoldFloat(t, "net proceeds", result.Transactions[2].NetAmount, 6_985_000)
	assertGoldFloat(t, "realized pnl", result.RealizedPnL, 6_985_000-(4*wantAverage))
}

func TestCalculateGoldMovingAverageRejectsHistoricalOversell(t *testing.T) {
	_, err := calculateGoldMovingAverage([]goldReplayTransaction{{Kind: GoldTransactionBuy, Quantity: 2, Gross: 3_000_000}, {Kind: GoldTransactionSell, Quantity: 3, Gross: 5_000_000}, {Kind: GoldTransactionBuy, Quantity: 5, Gross: 8_000_000}})
	if !errors.Is(err, ErrGoldHoldingQuantity) {
		t.Fatalf("expected ErrGoldHoldingQuantity, got %v", err)
	}
}

func assertGoldFloat(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.001 {
		t.Fatalf("%s: got %f want %f", name, got, want)
	}
}
