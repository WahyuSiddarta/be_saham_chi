package repository

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
)

func TestStockWritesRejectForeignPortfolioBeforeMutation(t *testing.T) {
	for _, operation := range []string{"create", "update", "delete"} {
		t.Run(operation, func(t *testing.T) {
			repo, conn := testRepository(t, queryResult{contains: "FOR UPDATE", columns: []string{"portfolio_id"}})
			var err error
			if operation == "delete" {
				err = repo.DeleteStockTransaction(context.Background(), "other-user", "p1", "t1")
			} else {
				id := ""
				if operation == "update" {
					id = "t1"
				}
				_, err = repo.SaveStockTransaction(context.Background(), "other-user", "p1", id, StockTransactionCommand{})
			}
			if !errors.Is(err, ErrPortfolioNotFound) || !conn.rolledBack || conn.committed || len(conn.queries) != 1 {
				t.Fatalf("ownership boundary: err=%v rollback=%v queries=%v", err, conn.rolledBack, conn.queries)
			}
			if conn.arguments[0][1].Value != "other-user" {
				t.Fatal("owner must be scoped in lock query")
			}
		})
	}
}
func TestStockMutationRejectsNonStockTransaction(t *testing.T) {
	repo, conn := testRepository(t,
		queryResult{contains: "FOR UPDATE", columns: []string{"portfolio_id"}, values: []driver.Value{"p1"}},
		queryResult{contains: "ac.code='stock'", columns: []string{"transaction_id"}},
	)
	err := repo.DeleteStockTransaction(context.Background(), "u1", "p1", "gold-transaction")
	if !errors.Is(err, ErrPortfolioStockTransactionNotFound) || !conn.rolledBack || len(conn.queries) != 2 {
		t.Fatalf("asset boundary: %v, queries %v", err, conn.queries)
	}
}

func TestStockLedgerWeightedCostAndHistoricalOversell(t *testing.T) {
	// The stock repository deliberately reuses the existing buy/sell replay engine.
	got, err := calculateGoldMovingAverage([]goldReplayTransaction{
		{Kind: "buy", Quantity: 100, Gross: 100000, Fee: 1000, Tax: 500},
		{Kind: "buy", Quantity: 100, Gross: 120000, Fee: 1000, Tax: 500},
		{Kind: "sell", Quantity: 50, Gross: 65000, Fee: 500, Tax: 250},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertGoldFloat(t, "remaining shares", got.Quantity, 150)
	assertGoldFloat(t, "average cost", got.AverageCost, 1115)
	assertGoldFloat(t, "remaining cost", got.TotalCost, 167250)
	assertGoldFloat(t, "realized pnl", got.RealizedPnL, 8500)
	_, err = calculateGoldMovingAverage([]goldReplayTransaction{{Kind: "sell", Quantity: 100, Gross: 100000}, {Kind: "buy", Quantity: 100, Gross: 90000}})
	if !errors.Is(err, ErrGoldHoldingQuantity) {
		t.Fatalf("backdated sale accepted: %v", err)
	}
}
