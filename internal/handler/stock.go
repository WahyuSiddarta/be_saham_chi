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

func (h Handler) CreateStock(w http.ResponseWriter, req *http.Request) error {
	var request CreateStockRequest
	if err := binding.BindJSON(req.Body, &request); err != nil {
		return response.Fail(w, http.StatusBadRequest, "invalid request body")
	}
	stock, err := h.stockService.CreateStock(req.Context(), request.Ticker, request.Name)
	if err != nil {
		if errors.Is(err, service.ErrInvalidStock) {
			h.logRequestError(req, http.StatusBadRequest, err.Error(), err)
			return response.Fail(w, http.StatusBadRequest, err.Error())
		}
		h.logRequestError(req, http.StatusInternalServerError, "failed to create stock", err)
		return response.Fail(w, http.StatusInternalServerError, "failed to create stock")
	}
	return response.Success(w, http.StatusCreated, StockResponse{StockItemResponse: newStockItemResponse(stock)})
}

func (h Handler) ListStock(w http.ResponseWriter, req *http.Request) error {
	stocks, err := h.stockService.ListStocks(req.Context())
	if err != nil {
		h.logRequestError(req, http.StatusInternalServerError, "failed to list stocks", err)
		return response.Fail(w, http.StatusInternalServerError, "failed to list stocks")
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
			return response.Fail(w, http.StatusBadRequest, "limit must be a positive integer")
		}
		limit = parsed
	}
	stocks, err := h.stockService.SearchTickers(req.Context(), req.URL.Query().Get("q"), limit)
	if err != nil {
		h.logRequestError(req, http.StatusInternalServerError, "failed to search stock tickers", err)
		return response.Fail(w, http.StatusInternalServerError, "failed to search stock tickers")
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
		h.logRequestError(req, http.StatusNotFound, "stock not found", err)
		return response.Fail(w, http.StatusNotFound, "stock not found")
	}
	if err != nil {
		h.logRequestError(req, http.StatusInternalServerError, "failed to get stock", err)
		return response.Fail(w, http.StatusInternalServerError, "failed to get stock")
	}
	return response.Success(w, http.StatusOK, StockResponse{StockItemResponse: newStockItemResponse(stock)})
}
func (h Handler) UpdateStock(w http.ResponseWriter, req *http.Request) error {
	var request UpdateStockRequest
	if err := binding.BindJSON(req.Body, &request); err != nil {
		return response.Fail(w, http.StatusBadRequest, "invalid request body")
	}
	stock, err := h.stockService.UpdateStockName(req.Context(), chi.URLParam(req, "ticker"), request.Name)
	if errors.Is(err, service.ErrInvalidStockName) {
		h.logRequestError(req, http.StatusBadRequest, err.Error(), err)
		return response.Fail(w, http.StatusBadRequest, err.Error())
	}
	if errors.Is(err, service.ErrStockNotFound) {
		h.logRequestError(req, http.StatusNotFound, "stock not found", err)
		return response.Fail(w, http.StatusNotFound, "stock not found")
	}
	if err != nil {
		h.logRequestError(req, http.StatusInternalServerError, "failed to update stock", err)
		return response.Fail(w, http.StatusInternalServerError, "failed to update stock")
	}
	return response.Success(w, http.StatusOK, StockResponse{StockItemResponse: newStockItemResponse(stock)})
}

func (h Handler) UpdateStockStatus(w http.ResponseWriter, req *http.Request) error {
	var request UpdateStockStatusRequest
	if err := binding.BindJSON(req.Body, &request); err != nil || request.Active == nil {
		return response.Fail(w, http.StatusBadRequest, "active is required")
	}
	stock, err := h.stockService.UpdateStockStatus(req.Context(), chi.URLParam(req, "ticker"), *request.Active)
	if errors.Is(err, service.ErrStockNotFound) {
		h.logRequestError(req, http.StatusNotFound, "stock not found", err)
		return response.Fail(w, http.StatusNotFound, "stock not found")
	}
	if err != nil {
		h.logRequestError(req, http.StatusInternalServerError, "failed to update stock status", err)
		return response.Fail(w, http.StatusInternalServerError, "failed to update stock status")
	}
	return response.Success(w, http.StatusOK, StockResponse{StockItemResponse: newStockItemResponse(stock)})
}

func (h Handler) GetStockKlines(w http.ResponseWriter, req *http.Request) error {
	from, to, err := stockKlineDateRange(req)
	if err != nil {
		return response.Fail(w, http.StatusBadRequest, err.Error())
	}
	items, err := h.stockService.GetKlines(req.Context(), chi.URLParam(req, "ticker"), from, to)
	if err != nil {
		if errors.Is(err, service.ErrStockNotFound) {
			h.logRequestError(req, http.StatusNotFound, "stock not found", err)
			return response.Fail(w, http.StatusNotFound, "stock not found")
		}
		if errors.Is(err, service.ErrInactiveStock) {
			h.logRequestError(req, http.StatusNotFound, "stock not found", err)
			return response.Fail(w, http.StatusNotFound, "stock not found")
		}
		h.logRequestError(req, http.StatusBadGateway, "failed to get stock kline", err)
		return response.Fail(w, http.StatusBadGateway, "failed to get stock kline")
	}
	return response.Success(w, http.StatusOK, stockKlinesToResponse(items))
}

func (h Handler) GetStockQuote(w http.ResponseWriter, req *http.Request) error {
	quote, err := h.stockService.GetQuote(req.Context(), chi.URLParam(req, "ticker"))
	if err != nil {
		if errors.Is(err, service.ErrStockNotFound) || errors.Is(err, service.ErrInactiveStock) {
			h.logRequestError(req, http.StatusNotFound, "stock not found", err)
			return response.Fail(w, http.StatusNotFound, "stock not found")
		}
		h.logRequestError(req, http.StatusBadGateway, "failed to get stock quote", err)
		return response.Fail(w, http.StatusBadGateway, "failed to get stock quote")
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
			h.logRequestError(req, http.StatusNotFound, "stock fundamentals not found", err)
			return response.Fail(w, http.StatusNotFound, "stock fundamentals not found")
		}
		h.logRequestError(req, http.StatusInternalServerError, "failed to read stock fundamentals", err)
		return response.Fail(w, http.StatusInternalServerError, "failed to read stock fundamentals")
	}
	return response.Success(w, http.StatusOK, StockFundamentalsResponse{
		Fundamentals: StockFundamentalsSnapshot{
			Ticker: fundamentals.Ticker, Source: repository.SourceStockbit,
			Payload: fundamentals.Payload, ScrapedAt: fundamentals.ScrapedAt,
			UpdatedAt: fundamentals.UpdatedAt,
		},
	})
}
