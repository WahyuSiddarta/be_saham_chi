package database

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// EnsureTables prepares a new database or an existing current V2 database.
// Includes shared scraper-owned data schemas used by Stockbit and Yahoo Finance jobs.
func EnsureTables(ctx context.Context, db *sqlx.DB) error {
	for _, setup := range []struct {
		name string
		run  func(context.Context, *sqlx.DB) error
	}{
		{"market", EnsureMarketTables}, {"auth", EnsureAuthTables}, {"master data", EnsureMasterDataTables},
		{"portfolio", EnsurePortfolioTables}, {"stock fundamentals", EnsureStockFundamentalsTables},
		{"stock financials", EnsureStockFinancialsTables}, {"stock betas", EnsureStockBetaTables},
	} {
		if err := setup.run(ctx, db); err != nil {
			return fmt.Errorf("ensure %s tables: %w", setup.name, err)
		}
	}
	return nil
}
