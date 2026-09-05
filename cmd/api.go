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
		publicRoute.Route("/auth", func(authRoute chi.Router) {
			registerRoutes(authRoute, handlers, []apiRoute{
				{http.MethodPost, "/register", "", handlers.Register},
				{http.MethodPost, "/login", "", handlers.Login},
			})
		})
	})

	// private routes
	privateRoute := r.Route("/api/v1/private", func(privateRoute chi.Router) {
		privateRoute.Use(middleware.Authenticate(app.config.jwt))
	})

	// portfolio routes
	privateRoute.Route("/portfolios", func(portfoliosRoute chi.Router) {
		registerRoutes(portfoliosRoute, handlers, []apiRoute{
			{http.MethodGet, "/", "portfolio.read", handlers.ListPortfolio},
			{http.MethodPost, "/", "portfolio.create", handlers.CreatePortfolio},
			{http.MethodGet, "/{portfolio_id}", "portfolio.read", handlers.GetPortfolio},
			{http.MethodPut, "/{portfolio_id}", "portfolio.update", handlers.UpdatePortfolio},
			{http.MethodDelete, "/{portfolio_id}", "portfolio.delete", handlers.DeletePortfolio},
		})
	})

	privateRoute.Route("/portfolio/{portfolio_id}", func(portfolioDetailRoute chi.Router) {
		// portfolio cash routes
		portfolioDetailRoute.Route("/cash", func(cashRoute chi.Router) {
			registerRoutes(cashRoute, handlers, []apiRoute{
				{http.MethodGet, "/", "portfolio.cash.read", handlers.GetCash},
				{http.MethodPost, "/", "portfolio.cash.create", handlers.AddCash},
				{http.MethodGet, "/snapshots", "portfolio.cash.read", handlers.ListCashSnapshots},
			})

			// portfolio cash transaction routes
			cashRoute.Route("/transactions", func(transactionRoute chi.Router) {
				registerRoutes(transactionRoute, handlers, []apiRoute{
					{http.MethodGet, "/", "portfolio.cash.read", handlers.ListCashTransactions},
					{http.MethodPost, "/", "portfolio.cash.create", handlers.CreateCashTransaction},
					{http.MethodGet, "/{transaction_id}", "portfolio.cash.read", handlers.GetCashTransaction},
					{http.MethodPut, "/{transaction_id}", "portfolio.cash.update", handlers.UpdateCashTransaction},
					{http.MethodDelete, "/{transaction_id}", "portfolio.cash.delete", handlers.DeleteCashTransaction},
				})
			})
		})

		// portfolio bond routes
		portfolioDetailRoute.Route("/bonds", func(bondRoute chi.Router) {
			registerRoutes(bondRoute, handlers, []apiRoute{
				{http.MethodGet, "/", "portfolio.bond.read", handlers.ListBonds},
				{http.MethodPost, "/", "portfolio.bond.create", handlers.CreateBond},
				{http.MethodGet, "/snapshots", "portfolio.bond.read", handlers.ListBondSnapshots},
			})

			// portfolio bond transaction routes
			bondRoute.Route("/transactions", func(transactionRoute chi.Router) {
				registerRoutes(transactionRoute, handlers, []apiRoute{
					{http.MethodGet, "/", "portfolio.bond.read", handlers.ListBondTransactions},
					{http.MethodPost, "/", "portfolio.bond.create", handlers.CreateBondTransaction},
					{http.MethodGet, "/{transaction_id}", "portfolio.bond.read", handlers.GetBondTransaction},
					{http.MethodPut, "/{transaction_id}", "portfolio.bond.update", handlers.UpdateBondTransaction},
					{http.MethodDelete, "/{transaction_id}", "portfolio.bond.delete", handlers.DeleteBondTransaction},
				})
			})

			// portfolio bond detail routes
			bondRoute.Route("/{asset_id}", func(assetRoute chi.Router) {
				registerRoutes(assetRoute, handlers, []apiRoute{
					{http.MethodGet, "/", "portfolio.bond.read", handlers.GetBond},
					{http.MethodPut, "/", "portfolio.bond.update", handlers.UpdateBond},
					{http.MethodPost, "/valuation", "portfolio.bond.update", handlers.AdjustBondValuation},
				})
			})
		})

		// portfolio commodity routes
		portfolioDetailRoute.Route("/commodities", func(commodityRoute chi.Router) {
			// portfolio gold routes
			commodityRoute.Route("/gold", func(goldRoute chi.Router) {
				registerRoutes(goldRoute, handlers, []apiRoute{
					{http.MethodGet, "/", "portfolio.commodity.read", handlers.GetGold},
					{http.MethodPost, "/", "portfolio.commodity.create", handlers.CreateGold},
				})

				// portfolio commodity transaction routes
				goldRoute.Route("/transactions", func(transactionRoute chi.Router) {
					registerRoutes(transactionRoute, handlers, []apiRoute{
						{http.MethodGet, "/", "portfolio.commodity.read", handlers.ListGoldTransactions},
						{http.MethodPost, "/", "portfolio.commodity.create", handlers.CreateGoldTransaction},
						{http.MethodGet, "/{transaction_id}", "portfolio.commodity.read", handlers.GetGoldTransaction},
						{http.MethodPut, "/{transaction_id}", "portfolio.commodity.update", handlers.UpdateGoldTransaction},
						{http.MethodDelete, "/{transaction_id}", "portfolio.commodity.delete", handlers.DeleteGoldTransaction},
					})
				})
			})
		})

	})

	// admin routes
	privateRoute.Route("/admin", func(adminRoute chi.Router) {
		// master data routes
		adminRoute.Route("/master-data", func(masterDataRoute chi.Router) {
			registerRoutes(masterDataRoute, handlers, []apiRoute{
				{http.MethodGet, "/", "master_data.read", handlers.ListMasterData},
				{http.MethodPut, "/{key}", "master_data.update", handlers.UpdateMasterData},
			})
		})

		// stock management routes
		adminRoute.Route("/stocks", func(stockRoute chi.Router) {
			// stock management routes require "stock.manage" permission
			stockRoute.Use(middleware.RequireRule("stock.manage"))
			registerRoutes(stockRoute, handlers, []apiRoute{
				{http.MethodGet, "/", "", handlers.ListStock},
				{http.MethodPost, "/", "", handlers.CreateStock},
			})

			// stock detail routes
			stockRoute.Route("/{ticker}", func(tickerRoute chi.Router) {
				registerRoutes(tickerRoute, handlers, []apiRoute{
					{http.MethodGet, "/", "", handlers.GetStock},
					{http.MethodPut, "/", "", handlers.UpdateStock},
					{http.MethodPut, "/status", "", handlers.UpdateStockStatus},
				})
			})
		})
	})

	// commodity market routes
	privateRoute.Route("/commodities", func(commodityRoute chi.Router) {
		commodityRoute.Route("/{commodity}", func(commodityDetailRoute chi.Router) {
			commodityDetailRoute.Use(middleware.CommodityRule)
			registerRoutes(commodityDetailRoute, handlers, []apiRoute{
				{http.MethodGet, "/quote", "", handlers.GetCommodityQuote},
				{http.MethodGet, "/kline", "", handlers.GetCommodityKlines},
			})
		})
	})

	// stock market routes
	privateRoute.Route("/stocks", func(stockRoute chi.Router) {
		stockRoute.Use(middleware.RequireRule("market.stock.read"))
		registerRoutes(stockRoute, handlers, []apiRoute{
			{http.MethodGet, "/tickers", "", handlers.SearchTickers},
		})
		stockRoute.Route("/{ticker}", func(tickerRoute chi.Router) {
			registerRoutes(tickerRoute, handlers, []apiRoute{
				{http.MethodGet, "/quote", "", handlers.GetStockQuote},
				{http.MethodGet, "/kline", "", handlers.GetStockKlines},
				{http.MethodGet, "/fundamentals", "", handlers.GetFundamentals},
			})
		})
	})

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
