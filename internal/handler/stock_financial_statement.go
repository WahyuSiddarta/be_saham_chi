package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
	"github.com/WahyuSiddarta/be_saham_chi/internal/response"
	"github.com/WahyuSiddarta/be_saham_chi/internal/service"

	"github.com/go-chi/chi/v5"
)

type StockFinancialStatementResponse struct {
	Ticker        string                         `json:"ticker"`
	StatementType service.FinancialStatementType `json:"statement_type"`
	ReportType    string                         `json:"report_type"`
	Source        repository.Source              `json:"source"`
	ScrapedAt     time.Time                      `json:"scraped_at"`
	Periods       []StockFinancialPeriodResponse `json:"periods"`
	Rows          []StockFinancialRowResponse    `json:"rows"`
	Pagination    PaginationResponse             `json:"pagination"`
}

type StockFinancialPeriodResponse struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Year    int    `json:"year"`
	Quarter *int   `json:"quarter"`
	Basis   string `json:"basis"`
}

type StockFinancialRowResponse struct {
	RowID      string                        `json:"row_id"`
	RowRole    string                        `json:"row_role"`
	MetricKey  string                        `json:"metric_key"`
	MetricName string                        `json:"metric_name"`
	Label      string                        `json:"label"`
	LabelEN    *string                       `json:"label_en"`
	LabelID    *string                       `json:"label_id"`
	Values     []StockFinancialValueResponse `json:"values"`
}

type StockFinancialValueResponse struct {
	PeriodKey    string  `json:"period_key"`
	Currency     *string `json:"currency"`
	Value        *string `json:"value"`
	IsMissing    bool    `json:"is_missing"`
	DisplayValue string  `json:"display_value"`
	Percentage   *string `json:"percentage"`
}

type PaginationResponse struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalRows  int `json:"total_rows"`
	TotalPages int `json:"total_pages"`
}

func (h Handler) GetIncomeStatement(w http.ResponseWriter, req *http.Request) error {
	return h.getFinancialStatement(w, req, service.FinancialStatementIncome)
}

func (h Handler) GetBalanceSheet(w http.ResponseWriter, req *http.Request) error {
	return h.getFinancialStatement(w, req, service.FinancialStatementBalance)
}

func (h Handler) GetCashFlow(w http.ResponseWriter, req *http.Request) error {
	return h.getFinancialStatement(w, req, service.FinancialStatementCashFlow)
}

func (h Handler) getFinancialStatement(w http.ResponseWriter, req *http.Request, statementType service.FinancialStatementType) error {
	filter, err := stockFinancialStatementFilter(req)
	if err != nil {
		return response.Fail(w, http.StatusBadRequest, err.Error())
	}
	statement, err := h.stockService.GetFinancialStatement(req.Context(), chi.URLParam(req, "ticker"), statementType, filter)
	if err != nil {
		if errors.Is(err, service.ErrStockNotFound) || errors.Is(err, service.ErrInactiveStock) {
			h.logRequestError(req, http.StatusNotFound, "stock financial statement not found", err)
			return response.Fail(w, http.StatusNotFound, "stock financial statement not found")
		}
		h.logRequestError(req, http.StatusInternalServerError, "failed to read stock financial statement", err)
		return response.Fail(w, http.StatusInternalServerError, "failed to read stock financial statement")
	}
	return response.Success(w, http.StatusOK, newStockFinancialStatementResponse(statement))
}

func stockFinancialStatementFilter(req *http.Request) (repository.StockFinancialStatementFilter, error) {
	filter := repository.StockFinancialStatementFilter{ReportType: "quarterly", Page: 1, PerPage: 20}
	query := req.URL.Query()
	if raw := query.Get("report_type"); raw != "" {
		filter.ReportType = raw
	}
	if filter.ReportType != "quarterly" && filter.ReportType != "annual" {
		return filter, errors.New("report_type must be quarterly or annual")
	}

	var err error
	if filter.FromYear, err = optionalPositiveInt(query.Get("from_year"), "from_year"); err != nil {
		return filter, err
	}
	if filter.ToYear, err = optionalPositiveInt(query.Get("to_year"), "to_year"); err != nil {
		return filter, err
	}
	if filter.FromYear != nil && filter.ToYear != nil && *filter.FromYear > *filter.ToYear {
		return filter, errors.New("from_year must not be after to_year")
	}
	if filter.Quarter, err = optionalPositiveInt(query.Get("quarter"), "quarter"); err != nil {
		return filter, err
	}
	if filter.Quarter != nil {
		if *filter.Quarter > 4 {
			return filter, errors.New("quarter must be between 1 and 4")
		}
		if filter.ReportType != "quarterly" {
			return filter, errors.New("quarter is only valid for quarterly reports")
		}
	}
	if filter.Page, err = positiveIntWithDefault(query.Get("page"), "page", 1); err != nil {
		return filter, err
	}
	if filter.PerPage, err = positiveIntWithDefault(query.Get("per_page"), "per_page", 20); err != nil {
		return filter, err
	}
	if filter.PerPage > 100 {
		return filter, errors.New("per_page must not exceed 100")
	}
	return filter, nil
}

func optionalPositiveInt(raw, name string) (*int, error) {
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return nil, errors.New(name + " must be a positive integer")
	}
	return &value, nil
}

func positiveIntWithDefault(raw, name string, fallback int) (int, error) {
	value, err := optionalPositiveInt(raw, name)
	if err != nil {
		return 0, err
	}
	if value == nil {
		return fallback, nil
	}
	return *value, nil
}

func newStockFinancialStatementResponse(statement service.StockFinancialStatement) StockFinancialStatementResponse {
	periods := make([]StockFinancialPeriodResponse, 0, len(statement.Periods))
	for _, period := range statement.Periods {
		periods = append(periods, StockFinancialPeriodResponse{Key: period.Key, Label: period.Label, Year: period.Year, Quarter: period.Quarter, Basis: period.Basis})
	}
	rows := make([]StockFinancialRowResponse, 0, len(statement.Rows))
	for _, row := range statement.Rows {
		values := make([]StockFinancialValueResponse, 0, len(row.Values))
		for _, value := range row.Values {
			values = append(values, StockFinancialValueResponse{
				PeriodKey: value.PeriodKey, Currency: value.Currency, Value: value.Value,
				IsMissing: value.IsMissing, DisplayValue: value.DisplayValue, Percentage: value.Percentage,
			})
		}
		rows = append(rows, StockFinancialRowResponse{
			RowID: row.RowID, RowRole: row.RowRole, MetricKey: row.MetricKey, MetricName: row.MetricName,
			Label: row.Label, LabelEN: row.LabelEN, LabelID: row.LabelID, Values: values,
		})
	}
	return StockFinancialStatementResponse{
		Ticker: statement.Ticker, StatementType: statement.StatementType, ReportType: statement.ReportType,
		Source: repository.SourceStockbit, ScrapedAt: statement.ScrapedAt, Periods: periods, Rows: rows,
		Pagination: PaginationResponse{Page: statement.Page, PerPage: statement.PerPage, TotalRows: statement.TotalRows, TotalPages: statement.TotalPages},
	}
}
