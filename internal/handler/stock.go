package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
	binding "github.com/WahyuSiddarta/be_saham_chi/internal/request"
	"github.com/WahyuSiddarta/be_saham_chi/internal/response"
	"github.com/WahyuSiddarta/be_saham_chi/internal/service"

	"github.com/go-chi/chi/v5"
)

type CreateStockRequest struct {
	Ticker string `json:"ticker"`
	Name   string `json:"name"`
}

type UpdateStockRequest struct {
	Name string `json:"name"`
}
type UpdateStockStatusRequest struct {
	Active *bool `json:"active"`
}

type StockItemResponse struct {
	Ticker    string    `json:"ticker"`
	Name      string    `json:"name"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type StockResponse struct {
	StockItemResponse
}

type StockListResponse = []StockItemResponse

type StockTickerResponse struct {
	Ticker string `json:"ticker"`
	Name   string `json:"name"`
}

type StockTickerListResponse = []StockTickerResponse

type StockKlineResponse struct {
	Symbol    string            `json:"symbol"`
	Interval  string            `json:"interval"`
	OpenTime  time.Time         `json:"open_time"`
	Open      float64           `json:"open"`
	High      float64           `json:"high"`
	Low       float64           `json:"low"`
	Close     float64           `json:"close"`
	Volume    int64             `json:"volume"`
	Source    repository.Source `json:"source"`
	FetchedAt time.Time         `json:"fetched_at"`
}

type StockKlineListResponse = []StockKlineResponse

type StockQuoteResponse struct {
	Symbol    string            `json:"symbol"`
	Open      float64           `json:"open"`
	High      float64           `json:"high"`
	Low       float64           `json:"low"`
	Close     float64           `json:"close"`
	Volume    int64             `json:"volume"`
	Source    repository.Source `json:"source"`
	FetchedAt time.Time         `json:"fetched_at"`
}

type StockFundamentalsResponse struct {
	Fundamentals StockFundamentalsSnapshot `json:"fundamentals"`
}

type StockFundamentalsSnapshot struct {
	Ticker    string            `json:"ticker"`
	Source    repository.Source `json:"source"`
	Payload   json.RawMessage   `json:"payload"`
	ScrapedAt time.Time         `json:"scraped_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

func newStockItemResponse(stock repository.Stock) StockItemResponse {
	return StockItemResponse{
		Ticker: stock.Ticker, Name: stock.Name, Active: stock.Active,
		CreatedAt: stock.CreatedAt, UpdatedAt: stock.UpdatedAt,
	}
}

func (h Handler) CreateStock(w http.ResponseWriter, req *http.Request) error {
	var request CreateStockRequest
	if err := binding.BindJSON(req.Body, &request); err != nil {
		return newHTTPError(http.StatusBadRequest, "invalid request body")
	}
	stock, err := h.stockService.CreateStock(req.Context(), request.Ticker, request.Name)
	if err != nil {
		if errors.Is(err, service.ErrInvalidStock) {
			return newHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		return newHTTPError(http.StatusInternalServerError, "failed to create stock").SetInternal(err)
	}
	return response.Success(w, http.StatusCreated, StockResponse{StockItemResponse: newStockItemResponse(stock)})
}
func (h Handler) ListStock(w http.ResponseWriter, req *http.Request) error {
	stocks, err := h.stockService.ListStocks(req.Context())
	if err != nil {
		return newHTTPError(http.StatusInternalServerError, "failed to list stocks").SetInternal(err)
	}
	data := make([]StockItemResponse, 0, len(stocks))
	for _, stock := range stocks {
		data = append(data, newStockItemResponse(stock))
	}
	return response.Success(w, http.StatusOK, data)
}
func (h Handler) SearchTickers(w http.ResponseWriter, req *http.Request) error {
	limit := 10
	if raw := req.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			return newHTTPError(http.StatusBadRequest, "limit must be a positive integer")
		}
		limit = parsed
	}
	stocks, err := h.stockService.SearchTickers(req.Context(), req.URL.Query().Get("q"), limit)
	if err != nil {
		return newHTTPError(http.StatusInternalServerError, "failed to search stock tickers").SetInternal(err)
	}
	data := make([]StockTickerResponse, 0, len(stocks))
	for _, stock := range stocks {
		data = append(data, StockTickerResponse{Ticker: stock.Ticker, Name: stock.Name})
	}
	return response.Success(w, http.StatusOK, data)
}
func (h Handler) GetStock(w http.ResponseWriter, req *http.Request) error {
	stock, err := h.stockService.GetStock(req.Context(), chi.URLParam(req, "ticker"))
	if errors.Is(err, service.ErrStockNotFound) {
		return newHTTPError(http.StatusNotFound, "stock not found").SetInternal(err)
	}
	if err != nil {
		return newHTTPError(http.StatusInternalServerError, "failed to get stock").SetInternal(err)
	}
	return response.Success(w, http.StatusOK, StockResponse{StockItemResponse: newStockItemResponse(stock)})
}
func (h Handler) UpdateStock(w http.ResponseWriter, req *http.Request) error {
	var request UpdateStockRequest
	if err := binding.BindJSON(req.Body, &request); err != nil {
		return newHTTPError(http.StatusBadRequest, "invalid request body")
	}
	stock, err := h.stockService.UpdateStockName(req.Context(), chi.URLParam(req, "ticker"), request.Name)
	if errors.Is(err, service.ErrInvalidStockName) {
		return newHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
	}
	if errors.Is(err, service.ErrStockNotFound) {
		return newHTTPError(http.StatusNotFound, "stock not found").SetInternal(err)
	}
	if err != nil {
		return newHTTPError(http.StatusInternalServerError, "failed to update stock").SetInternal(err)
	}
	return response.Success(w, http.StatusOK, StockResponse{StockItemResponse: newStockItemResponse(stock)})
}
func (h Handler) UpdateStockStatus(w http.ResponseWriter, req *http.Request) error {
	var request UpdateStockStatusRequest
	if err := binding.BindJSON(req.Body, &request); err != nil || request.Active == nil {
		return newHTTPError(http.StatusBadRequest, "active is required")
	}
	stock, err := h.stockService.UpdateStockStatus(req.Context(), chi.URLParam(req, "ticker"), *request.Active)
	if errors.Is(err, service.ErrStockNotFound) {
		return newHTTPError(http.StatusNotFound, "stock not found").SetInternal(err)
	}
	if err != nil {
		return newHTTPError(http.StatusInternalServerError, "failed to update stock status").SetInternal(err)
	}
	return response.Success(w, http.StatusOK, StockResponse{StockItemResponse: newStockItemResponse(stock)})
}
func (h Handler) GetStockKlines(w http.ResponseWriter, req *http.Request) error {
	from, to, err := stockKlineDateRange(req)
	if err != nil {
		return err
	}
	items, err := h.stockService.GetKlines(req.Context(), chi.URLParam(req, "ticker"), from, to)
	if err != nil {
		if errors.Is(err, service.ErrStockNotFound) {
			return newHTTPError(http.StatusNotFound, "stock not found").SetInternal(err)
		}
		if errors.Is(err, service.ErrInactiveStock) {
			return newHTTPError(http.StatusNotFound, "stock not found").SetInternal(err)
		}
		return newHTTPError(http.StatusBadGateway, "failed to get stock kline").SetInternal(err)
	}
	return response.Success(w, http.StatusOK, stockKlinesToResponse(items))
}

func (h Handler) GetStockQuote(w http.ResponseWriter, req *http.Request) error {
	quote, err := h.stockService.GetQuote(req.Context(), chi.URLParam(req, "ticker"))
	if err != nil {
		if errors.Is(err, service.ErrStockNotFound) || errors.Is(err, service.ErrInactiveStock) {
			return newHTTPError(http.StatusNotFound, "stock not found").SetInternal(err)
		}
		return newHTTPError(http.StatusBadGateway, "failed to get stock quote").SetInternal(err)
	}
	return response.Success(w, http.StatusOK, StockQuoteResponse{
		Symbol: quote.Symbol, Open: quote.Open, High: quote.High,
		Low: quote.Low, Close: quote.Close, Volume: quote.Volume,
		Source: quote.Source, FetchedAt: quote.FetchedAt,
	})
}

func (h Handler) GetFundamentals(w http.ResponseWriter, req *http.Request) error {
	fundamentals, err := h.stockService.GetFundamentals(req.Context(), chi.URLParam(req, "ticker"))
	if err != nil {
		if errors.Is(err, service.ErrStockNotFound) || errors.Is(err, service.ErrInactiveStock) {
			return newHTTPError(http.StatusNotFound, "stock fundamentals not found").SetInternal(err)
		}
		return newHTTPError(http.StatusInternalServerError, "failed to read stock fundamentals").SetInternal(err)
	}
	return response.Success(w, http.StatusOK, StockFundamentalsResponse{
		Fundamentals: StockFundamentalsSnapshot{
			Ticker: fundamentals.Ticker, Source: repository.SourceStockbit,
			Payload: fundamentals.Payload, ScrapedAt: fundamentals.ScrapedAt,
			UpdatedAt: fundamentals.UpdatedAt,
		},
	})
}

func stockKlineDateRange(req *http.Request) (time.Time, time.Time, error) {
	to := time.Now().UTC()
	from := to.AddDate(0, -1, 0)
	var err error
	if raw := req.URL.Query().Get("from"); raw != "" {
		from, err = time.Parse("2006-01-02", raw)
		if err != nil {
			return time.Time{}, time.Time{}, newHTTPError(http.StatusBadRequest, "invalid from date")
		}
	}
	if raw := req.URL.Query().Get("to"); raw != "" {
		to, err = time.Parse("2006-01-02", raw)
		if err != nil {
			return time.Time{}, time.Time{}, newHTTPError(http.StatusBadRequest, "invalid to date")
		}
	}
	if from.After(to) {
		return time.Time{}, time.Time{}, newHTTPError(http.StatusBadRequest, "from must not be after to")
	}
	return from, to, nil
}
func stockKlinesToResponse(items []repository.StockKline) []StockKlineResponse {
	out := make([]StockKlineResponse, 0, len(items))
	for _, k := range items {
		out = append(out, StockKlineResponse{
			Symbol: k.Symbol, Interval: k.Interval, OpenTime: k.OpenTime,
			Open: k.Open, High: k.High, Low: k.Low, Close: k.Close,
			Volume: k.Volume, Source: k.Source, FetchedAt: k.FetchedAt,
		})
	}
	return out
}
