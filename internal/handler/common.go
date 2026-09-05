package handler

import (
	"errors"
	"net/http"

	"github.com/WahyuSiddarta/be_saham_chi/internal/response"
	"github.com/WahyuSiddarta/be_saham_chi/internal/service"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

type Handler struct {
	status            string
	log               *zerolog.Logger
	authService       *service.AuthService
	commodityService  *service.CommodityService
	stockService      *service.StockService
	portfolioService  *service.PortfolioService
	cashService       *service.CashService
	bondService       *service.BondService
	goldService       *service.GoldService
	masterDataService *service.MasterDataService
}

func New(status string, log *zerolog.Logger, authService *service.AuthService, domains Domains) Handler {
	return Handler{
		status:            status,
		log:               log,
		authService:       authService,
		commodityService:  domains.Commodity,
		stockService:      domains.Stock,
		portfolioService:  domains.Portfolio,
		cashService:       domains.Cash,
		bondService:       domains.Bond,
		goldService:       domains.Gold,
		masterDataService: domains.MasterData,
	}
}

func (h Handler) fail(w http.ResponseWriter, status int, message string) {
	if err := response.Fail(w, status, message); err != nil {
		h.log.Error().Err(err).Msg("write failure response")
	}
}

type Domains struct {
	Commodity  *service.CommodityService
	Stock      *service.StockService
	Portfolio  *service.PortfolioService
	Cash       *service.CashService
	Bond       *service.BondService
	Gold       *service.GoldService
	MasterData *service.MasterDataService
}

// Handle maps domain errors to the shared response envelope.
func (h Handler) Handle(next func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := next(w, r); err != nil {
			code, message := http.StatusInternalServerError, "Internal Server Error"
			var failure *httpError
			if errors.As(err, &failure) {
				code, message = failure.Code, failure.Message
			}
			event := h.log.Warn()
			if code >= http.StatusInternalServerError {
				event = h.log.Error()
			}
			event.Err(err).Int("status_code", code).Str("request_id", middleware.GetReqID(r.Context())).Str("method", r.Method).Str("path", r.URL.Path).Msg("request failed")
			_ = response.Fail(w, code, message)
		}
	}
}

type httpError struct {
	Code     int
	Message  string
	Internal error
}

func newHTTPError(code int, message string) *httpError {
	return &httpError{Code: code, Message: message}
}
func (e *httpError) Error() string {
	if e.Internal != nil {
		return e.Message + ": " + e.Internal.Error()
	}
	return e.Message
}
func (e *httpError) Unwrap() error                    { return e.Internal }
func (e *httpError) SetInternal(err error) *httpError { e.Internal = err; return e }
