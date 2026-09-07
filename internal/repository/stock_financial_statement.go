package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type StockFinancialStatementFilter struct {
	ReportType string
	FromYear   *int
	ToYear     *int
	Quarter    *int
	Page       int
	PerPage    int
}

type StockFinancialStatementSummary struct {
	ScrapedAt time.Time `db:"scraped_at"`
	TotalRows int       `db:"total_rows"`
}

type StockFinancialStatementValue struct {
	RowID         string         `db:"row_id"`
	RowRole       string         `db:"row_role"`
	MetricKey     string         `db:"metric_key"`
	MetricName    string         `db:"metric_name"`
	Label         string         `db:"label"`
	LabelEN       sql.NullString `db:"label_en"`
	LabelID       sql.NullString `db:"label_id"`
	PeriodKey     string         `db:"period_key"`
	PeriodLabel   string         `db:"period_label"`
	PeriodYear    int            `db:"period_year"`
	PeriodQuarter sql.NullInt64  `db:"period_quarter"`
	PeriodBasis   string         `db:"period_basis"`
	Currency      sql.NullString `db:"currency"`
	Value         sql.NullString `db:"value"`
	IsMissing     bool           `db:"is_missing"`
	DisplayValue  string         `db:"display_value"`
	Percentage    sql.NullString `db:"percentage"`
}

func (r *Repository) GetFinancialStatementSummary(ctx context.Context, ticker string, report int, filter StockFinancialStatementFilter) (StockFinancialStatementSummary, error) {
	var summary StockFinancialStatementSummary
	err := r.db.GetContext(ctx, &summary, `
		WITH latest_capture AS (
			SELECT id, scraped_at
			FROM stock_financials
			WHERE ticker = $1
			ORDER BY scraped_at DESC, id DESC
			LIMIT 1
		)
		SELECT c.scraped_at, COUNT(DISTINCT v.row_id) AS total_rows
		FROM latest_capture c
		LEFT JOIN stock_financial_values v
		  ON v.capture_id = c.id
		 AND v.ticker = $1
		 AND v.table_id = $2
		 AND v.period_type = $3
		 AND ($4::integer IS NULL OR v.period_year >= $4)
		 AND ($5::integer IS NULL OR v.period_year <= $5)
		 AND ($6::integer IS NULL OR v.period_quarter = $6)
		GROUP BY c.scraped_at
	`, ticker, financialStatementTableID(report, filter.ReportType), financialStatementPeriodType(filter.ReportType), filter.FromYear, filter.ToYear, filter.Quarter)
	if errors.Is(err, sql.ErrNoRows) {
		return StockFinancialStatementSummary{}, ErrStockNotFound
	}
	return summary, err
}

func (r *Repository) ListFinancialStatementValues(ctx context.Context, ticker string, report int, filter StockFinancialStatementFilter) ([]StockFinancialStatementValue, error) {
	items := make([]StockFinancialStatementValue, 0)
	offset := (filter.Page - 1) * filter.PerPage
	err := r.db.SelectContext(ctx, &items, `
		WITH latest_capture AS (
			SELECT id
			FROM stock_financials
			WHERE ticker = $1
			ORDER BY scraped_at DESC, id DESC
			LIMIT 1
		), filtered_values AS (
			SELECT v.*
			FROM stock_financial_values v
			JOIN latest_capture c ON c.id = v.capture_id
			WHERE v.ticker = $1
			  AND v.table_id = $2
			  AND v.period_type = $3
			  AND ($4::integer IS NULL OR v.period_year >= $4)
			  AND ($5::integer IS NULL OR v.period_year <= $5)
			  AND ($6::integer IS NULL OR v.period_quarter = $6)
		), available_rows AS (
			SELECT DISTINCT row_id, row_role, row_left, row_right, metric_key, metric_name, label, label_en, label_id
			FROM filtered_values
		), paged_rows AS (
			SELECT *
			FROM available_rows
			ORDER BY
			  CASE WHEN row_left ~ '^[0-9]+$' THEN row_left::integer ELSE 2147483647 END,
			  CASE WHEN row_right ~ '^[0-9]+$' THEN row_right::integer ELSE 2147483647 END,
			  row_id
			LIMIT $7 OFFSET $8
		)
		SELECT
		  v.row_id, v.row_role, v.metric_key, v.metric_name, v.label, v.label_en, v.label_id,
		  v.period_key, v.period_label, v.period_year, v.period_quarter, v.period_basis, v.currency,
		  v.value::text AS value, v.is_missing, v.display_value, v.percentage::text AS percentage
		FROM filtered_values v
		JOIN paged_rows r ON r.row_id = v.row_id
		ORDER BY
		  CASE WHEN v.row_left ~ '^[0-9]+$' THEN v.row_left::integer ELSE 2147483647 END,
		  CASE WHEN v.row_right ~ '^[0-9]+$' THEN v.row_right::integer ELSE 2147483647 END,
		  v.row_id,
		  v.period_year DESC,
		  v.period_quarter DESC NULLS LAST,
		  v.period_key DESC
	`, ticker, financialStatementTableID(report, filter.ReportType), financialStatementPeriodType(filter.ReportType), filter.FromYear, filter.ToYear, filter.Quarter, filter.PerPage, offset)
	return items, err
}

func financialStatementTableID(report int, reportType string) string {
	statement := 1
	if reportType == "annual" {
		statement = 2
	}
	return fmt.Sprintf("financialStatement:report-%d:statement-%d:data_table_1", report, statement)
}

func financialStatementPeriodType(reportType string) string {
	if reportType == "annual" {
		return "annual"
	}
	return "quarter"
}
