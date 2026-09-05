package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/auth"
	"github.com/WahyuSiddarta/be_saham_chi/internal/handler"
	"github.com/WahyuSiddarta/be_saham_chi/internal/middleware"
	"github.com/WahyuSiddarta/be_saham_chi/internal/provider/yahoo"
	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
	"github.com/WahyuSiddarta/be_saham_chi/internal/response"
	"github.com/WahyuSiddarta/be_saham_chi/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

type Application struct {
	config   Config
	database *sqlx.DB
}

type Config struct {
	addr        string
	databaseURL string
	logFile     string
	status      string
	jwt         auth.Config
	goldSymbol  string
	wtiSymbol   string
	brentSymbol string
	corsOrigins []string
}

func (app Application) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RequestLogger(Log))
	r.Use(middleware.Recover(Log))
	r.Use(middleware.CORS(app.config.corsOrigins))

	repositories := repository.New(app.database)
	authService := service.NewAuthService(repositories, app.config.jwt)
	commodityService := service.NewCommodityService(map[string]string{
		"gold": app.config.goldSymbol, "oil-wti": app.config.wtiSymbol, "oil-brent": app.config.brentSymbol,
	}, yahoo.NewCommodityProvider(app.config.goldSymbol), repositories, repositories)

	handlers := handler.New(app.config.status, Log, authService, handler.Domains{
		Commodity:  commodityService,
		Stock:      service.NewStockService(yahoo.NewStockProvider(), repositories),
		Portfolio:  service.NewPortfolioService(repositories),
		Cash:       service.NewCashService(repositories),
		Bond:       service.NewBondService(repositories),
		Gold:       service.NewGoldService(repositories),
		MasterData: service.NewMasterDataService(repositories),
	})

	// public routes
	r.Route("/api/v1/public", func(publicRoute chi.Router) {
		publicRoute.Get("/health", handlers.Health)
		publicRoute.With(middleware.Authenticate(app.config.jwt)).Get("/protected", handlers.ProtectedExample)
		publicRoute.Get("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "docs/openapi.yaml") })
		publicRoute.Get("/docs", handlers.Docs)
		publicRoute.Get("/docs/", handlers.Docs)
		registerRoutes(publicRoute, handlers, "/auth", "", []apiRoute{
			{http.MethodPost, "/register", "", handlers.Register},
			{http.MethodPost, "/login", "", handlers.Login},
		})
	})

	// private routes
	privateRoute := r.Route("/api/v1/private", func(privateRoute chi.Router) {
		privateRoute.Use(middleware.Authenticate(app.config.jwt))
	})

	registerRoutes(privateRoute, handlers, "/portfolios", "portfolio", []apiRoute{
		{http.MethodGet, "/", "read", handlers.ListPortfolio},
		{http.MethodPost, "/", "create", handlers.CreatePortfolio},
		{http.MethodGet, "/{portfolio_id}", "read", handlers.GetPortfolio},
		{http.MethodPut, "/{portfolio_id}", "update", handlers.UpdatePortfolio},
		{http.MethodDelete, "/{portfolio_id}", "delete", handlers.DeletePortfolio},
	})

	registerRoutes(privateRoute, handlers, "/portfolio/{portfolio_id}/cash", "portfolio.cash", []apiRoute{
		{http.MethodGet, "/", "read", handlers.GetCash},
		{http.MethodPost, "/", "create", handlers.AddCash},
		{http.MethodGet, "/snapshots", "read", handlers.ListCashSnapshots},
		{http.MethodGet, "/transactions", "read", handlers.ListCashTransactions},
		{http.MethodPost, "/transactions", "create", handlers.CreateCashTransaction},
		{http.MethodGet, "/transactions/{transaction_id}", "read", handlers.GetCashTransaction},
		{http.MethodPut, "/transactions/{transaction_id}", "update", handlers.UpdateCashTransaction},
		{http.MethodDelete, "/transactions/{transaction_id}", "delete", handlers.DeleteCashTransaction},
	})

	registerRoutes(privateRoute, handlers, "/portfolio/{portfolio_id}/bonds", "portfolio.bond", []apiRoute{
		{http.MethodGet, "/", "read", handlers.ListBonds},
		{http.MethodPost, "/", "create", handlers.CreateBond},
		{http.MethodGet, "/snapshots", "read", handlers.ListBondSnapshots},
		{http.MethodGet, "/transactions", "read", handlers.ListBondTransactions},
		{http.MethodPost, "/transactions", "create", handlers.CreateBondTransaction},
		{http.MethodGet, "/transactions/{transaction_id}", "read", handlers.GetBondTransaction},
		{http.MethodPut, "/transactions/{transaction_id}", "update", handlers.UpdateBondTransaction},
		{http.MethodDelete, "/transactions/{transaction_id}", "delete", handlers.DeleteBondTransaction},
		{http.MethodGet, "/{asset_id}", "read", handlers.GetBond},
		{http.MethodPut, "/{asset_id}", "update", handlers.UpdateBond},
		{http.MethodPost, "/{asset_id}/valuation", "update", handlers.AdjustBondValuation},
	})

	registerRoutes(privateRoute, handlers, "/portfolio/{portfolio_id}/commodities/gold", "portfolio.commodity", []apiRoute{
		{http.MethodGet, "/", "read", handlers.GetGold},
		{http.MethodPost, "/", "create", handlers.CreateGold},
		{http.MethodGet, "/transactions", "read", handlers.ListGoldTransactions},
		{http.MethodPost, "/transactions", "create", handlers.CreateGoldTransaction},
		{http.MethodGet, "/transactions/{transaction_id}", "read", handlers.GetGoldTransaction},
		{http.MethodPut, "/transactions/{transaction_id}", "update", handlers.UpdateGoldTransaction},
		{http.MethodDelete, "/transactions/{transaction_id}", "delete", handlers.DeleteGoldTransaction},
	})

	registerRoutes(privateRoute, handlers, "/admin/master-data", "master_data", []apiRoute{
		{http.MethodGet, "/", "read", handlers.ListMasterData},
		{http.MethodPut, "/{key}", "update", handlers.UpdateMasterData},
	})

	registerRoutes(privateRoute, handlers, "/admin/stocks", "", []apiRoute{
		{http.MethodGet, "/", "", handlers.ListStock},
		{http.MethodPost, "/", "", handlers.CreateStock},
		{http.MethodGet, "/{ticker}", "", handlers.GetStock},
		{http.MethodPut, "/{ticker}", "", handlers.UpdateStock},
		{http.MethodPut, "/{ticker}/status", "", handlers.UpdateStockStatus},
	}, middleware.RequireRule("stock.manage"))

	registerRoutes(privateRoute, handlers, "/commodities/{commodity}", "", []apiRoute{
		{http.MethodGet, "/quote", "", handlers.GetCommodityQuote},
		{http.MethodGet, "/kline", "", handlers.GetCommodityKlines},
	}, middleware.CommodityRule)

	registerRoutes(privateRoute, handlers, "/stocks", "", []apiRoute{
		{http.MethodGet, "/tickers", "", handlers.SearchTickers},
		{http.MethodGet, "/{ticker}/quote", "", handlers.GetStockQuote},
		{http.MethodGet, "/{ticker}/kline", "", handlers.GetStockKlines},
		{http.MethodGet, "/{ticker}/fundamentals", "", handlers.GetFundamentals},
	}, middleware.RequireRule("market.stock.read"))

	r.NotFound(func(w http.ResponseWriter, r *http.Request) { _ = response.Fail(w, http.StatusNotFound, "Not Found") })
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		_ = response.Fail(w, http.StatusMethodNotAllowed, "Method Not Allowed")
	})
	return r
}

func loadJWTConfig() (auth.Config, error) {
	ttl, err := time.ParseDuration(os.Getenv("JWT_TTL"))
	if err != nil || ttl <= 0 {
		return auth.Config{}, fmt.Errorf("JWT_TTL must be a positive duration")
	}

	config := auth.Config{
		Secret: os.Getenv("JWT_SECRET"),
		Issuer: os.Getenv("JWT_ISSUER"),
		TTL:    ttl,
	}
	if config.Secret == "" || config.Issuer == "" {
		return auth.Config{}, fmt.Errorf("JWT_SECRET and JWT_ISSUER must be set")
	}

	return config, nil
}
