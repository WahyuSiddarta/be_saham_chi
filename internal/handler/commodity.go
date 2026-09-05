package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/auth"
	applogger "github.com/WahyuSiddarta/be_saham_chi/internal/logger"
	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
	"github.com/WahyuSiddarta/be_saham_chi/internal/response"
	"github.com/WahyuSiddarta/be_saham_chi/internal/service"

	"github.com/go-chi/chi/v5"
)

func (h Handler) GetCommodityQuote(w http.ResponseWriter, req *http.Request) error {
	commodity := chi.URLParam(req, "commodity")
	userID, ok := auth.UserIDFromContext(req.Context())
	if !ok {
		return newHTTPError(http.StatusUnauthorized, "missing user_id")
	}

	applogger.Info().
		Str("user_id", userID).
		Str("commodity", commodity).
		Msg("get commodity quote")

	quote, err := h.commodityService.GetQuote(req.Context(), commodity)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCommodity) {
			return newHTTPError(http.StatusNotFound, "commodity not found").SetInternal(err)
		}
		return newHTTPError(http.StatusBadGateway, "failed to fetch commodity quote").SetInternal(err)
	}

	return response.JSON(w, http.StatusOK, NewQuoteResponse(quote))
}

func (h Handler) GetCommodityKlines(w http.ResponseWriter, req *http.Request) error {
	dataRange := req.URL.Query().Get("range")
	if dataRange == "" {
		dataRange = "1mo"
	}

	klines, err := h.commodityService.GetKlines(req.Context(), chi.URLParam(req, "commodity"), dataRange)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCommodity) {
			return newHTTPError(http.StatusNotFound, "commodity not found").SetInternal(err)
		}
		if errors.Is(err, service.ErrInvalidRange) {
			return newHTTPError(http.StatusBadRequest, "invalid range").SetInternal(err)
		}
		return newHTTPError(http.StatusBadGateway, "failed to fetch commodity kline").SetInternal(err)
	}

	return response.JSON(w, http.StatusOK, NewKlineListResponse(klines))
}

type QuoteResponse struct {
	Status    string            `json:"status"`
	Symbol    string            `json:"symbol"`
	Open      float64           `json:"open"`
	High      float64           `json:"high"`
	Low       float64           `json:"low"`
	Close     float64           `json:"close"`
	Volume    int64             `json:"volume"`
	Source    repository.Source `json:"source"`
	FetchedAt time.Time         `json:"fetched_at"`
}

func NewQuoteResponse(quote repository.MarketPrice) QuoteResponse {
	return QuoteResponse{
		Status:    "ok",
		Symbol:    quote.Symbol,
		Open:      quote.Open,
		High:      quote.High,
		Low:       quote.Low,
		Close:     quote.Close,
		Volume:    quote.Volume,
		Source:    quote.Source,
		FetchedAt: quote.FetchedAt,
	}
}

type KlineListResponse struct {
	Status string          `json:"status"`
	Data   []KlineResponse `json:"data"`
}

type KlineResponse struct {
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

func NewKlineListResponse(klines []repository.MarketKline) KlineListResponse {
	response := make([]KlineResponse, 0, len(klines))
	for _, kline := range klines {
		response = append(response, KlineResponse{
			Symbol:    kline.Symbol,
			Interval:  kline.Interval,
			OpenTime:  kline.OpenTime,
			Open:      kline.Open,
			High:      kline.High,
			Low:       kline.Low,
			Close:     kline.Close,
			Volume:    kline.Volume,
			Source:    kline.Source,
			FetchedAt: kline.FetchedAt,
		})
	}

	return KlineListResponse{
		Status: "ok",
		Data:   response,
	}
}
