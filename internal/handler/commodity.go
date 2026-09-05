package handler

import (
	"errors"
	"net/http"
	"time"

	applogger "github.com/WahyuSiddarta/be_saham_chi/internal/logger"
	"github.com/WahyuSiddarta/be_saham_chi/internal/middleware"
	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
	"github.com/WahyuSiddarta/be_saham_chi/internal/response"
	"github.com/WahyuSiddarta/be_saham_chi/internal/service"

	"github.com/go-chi/chi/v5"
)

func (h Handler) GetCommodityQuote(w http.ResponseWriter, req *http.Request) error {
	commodity := chi.URLParam(req, "commodity")
	userID, ok := middleware.UserIDFromContext(req.Context())
	if !ok {
		return response.Fail(w, http.StatusUnauthorized, "missing user_id")
	}

	applogger.Info().
		Str("user_id", userID).
		Str("commodity", commodity).
		Msg("get commodity quote")

	quote, err := h.commodityService.GetQuote(req.Context(), commodity)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCommodity) {
			h.logRequestError(req, http.StatusNotFound, "commodity not found", err)
			return response.Fail(w, http.StatusNotFound, "commodity not found")
		}
		h.logRequestError(req, http.StatusBadGateway, "failed to fetch commodity quote", err)
		return response.Fail(w, http.StatusBadGateway, "failed to fetch commodity quote")
	}

	return response.Success(w, http.StatusOK, NewQuoteResponse(quote))
}

func (h Handler) GetCommodityKlines(w http.ResponseWriter, req *http.Request) error {
	dataRange := req.URL.Query().Get("range")
	if dataRange == "" {
		dataRange = "1mo"
	}

	klines, err := h.commodityService.GetKlines(req.Context(), chi.URLParam(req, "commodity"), dataRange)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCommodity) {
			h.logRequestError(req, http.StatusNotFound, "commodity not found", err)
			return response.Fail(w, http.StatusNotFound, "commodity not found")
		}
		if errors.Is(err, service.ErrInvalidRange) {
			h.logRequestError(req, http.StatusBadRequest, "invalid range", err)
			return response.Fail(w, http.StatusBadRequest, "invalid range")
		}
		h.logRequestError(req, http.StatusBadGateway, "failed to fetch commodity kline", err)
		return response.Fail(w, http.StatusBadGateway, "failed to fetch commodity kline")
	}

	return response.Success(w, http.StatusOK, NewKlineListResponse(klines))
}

type QuoteResponse struct {
	Symbol    string            `json:"symbol"`
	Open      float64           `json:"open"`
	High      float64           `json:"high"`
	Low       float64           `json:"low"`
	Close     float64           `json:"close"`
	Volume    int64             `json:"volume"`
	Source    repository.Source `json:"source"`
	FetchedAt time.Time         `json:"fetched_at"`
}

type KlineListResponse = []KlineResponse

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
