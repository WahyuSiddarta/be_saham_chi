package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/auth"
	"github.com/WahyuSiddarta/be_saham_chi/internal/handler"
	"github.com/WahyuSiddarta/be_saham_chi/internal/logger"
	"github.com/WahyuSiddarta/be_saham_chi/internal/provider/yahoo"
	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
	"github.com/WahyuSiddarta/be_saham_chi/internal/response"
	"github.com/WahyuSiddarta/be_saham_chi/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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
	r.Use(middleware.RequestLogger(logger.ChiLogFormatter()))
	r.Use(recoverJSON)
	r.Use(corsMiddleware(app.config.corsOrigins))

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
	r.Get("/health", handlers.Health)
	r.Post("/auth/login", handlers.Login)
	r.With(auth.Middleware(app.config.jwt)).Get("/protected", handlers.ProtectedExample)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/public", func(r chi.Router) {
			r.Route("/auth", func(r chi.Router) {
				r.Post("/register", handlers.Handle(handlers.Register))
				r.Post("/login", handlers.Handle(handlers.LoginV2))
			})
		})
		r.Route("/private", func(r chi.Router) {
			r.Use(auth.Middleware(app.config.jwt))
			r.Route("/admin", func(r chi.Router) {
				r.Route("/master-data", func(r chi.Router) {
					r.With(auth.RequireRule("master_data.read")).Get("/", handlers.Handle(handlers.ListMasterData))
					r.With(auth.RequireRule("master_data.update")).Put("/{key}", handlers.Handle(handlers.UpdateMasterData))
				})
				r.Route("/stocks", func(r chi.Router) {
					r.Use(auth.RequireRule("stock.manage"))
					r.Get("/", handlers.Handle(handlers.ListStock))
					r.Post("/", handlers.Handle(handlers.CreateStock))
					r.Route("/{ticker}", func(r chi.Router) {
						r.Get("/", handlers.Handle(handlers.GetStock))
						r.Put("/", handlers.Handle(handlers.UpdateStock))
						r.Put("/status", handlers.Handle(handlers.UpdateStockStatus))
					})
				})
			})
			r.Route("/commodities", func(r chi.Router) {
				r.Route("/{commodity}", func(r chi.Router) {
					r.Use(auth.CommodityRule)
					r.Get("/quote", handlers.Handle(handlers.GetCommodityQuote))
					r.Get("/kline", handlers.Handle(handlers.GetCommodityKlines))
				})
			})
			r.Route("/stocks", func(r chi.Router) {
				r.Use(auth.RequireRule("market.stock.read"))
				r.Get("/tickers", handlers.Handle(handlers.SearchTickers))
				r.Route("/{ticker}", func(r chi.Router) {
					r.Get("/quote", handlers.Handle(handlers.GetStockQuote))
					r.Get("/kline", handlers.Handle(handlers.GetStockKlines))
					r.Get("/fundamentals", handlers.Handle(handlers.GetFundamentals))
				})
			})
			r.Route("/portfolios", func(r chi.Router) {
				r.With(auth.RequireRule("portfolio.read")).Get("/", handlers.Handle(handlers.ListPortfolio))
				r.With(auth.RequireRule("portfolio.create")).Post("/", handlers.Handle(handlers.CreatePortfolio))
				r.With(auth.RequireRule("portfolio.read")).Get("/{portfolio_id}", handlers.Handle(handlers.GetPortfolio))
				r.With(auth.RequireRule("portfolio.update")).Put("/{portfolio_id}", handlers.Handle(handlers.UpdatePortfolio))
				r.With(auth.RequireRule("portfolio.delete")).Delete("/{portfolio_id}", handlers.Handle(handlers.DeletePortfolio))
			})
			r.Route("/portfolio", func(r chi.Router) {
				r.Route("/{portfolio_id}", func(r chi.Router) {
					r.Route("/cash", func(r chi.Router) {
						r.With(auth.RequireRule("portfolio.cash.read")).Get("/", handlers.Handle(handlers.GetCash))
						r.With(auth.RequireRule("portfolio.cash.create")).Post("/", handlers.Handle(handlers.AddCash))
						r.With(auth.RequireRule("portfolio.cash.read")).Get("/snapshots", handlers.Handle(handlers.ListCashSnapshots))
						r.Route("/transactions", func(r chi.Router) {
							r.With(auth.RequireRule("portfolio.cash.read")).Get("/", handlers.Handle(handlers.ListCashTransactions))
							r.With(auth.RequireRule("portfolio.cash.create")).Post("/", handlers.Handle(handlers.CreateCashTransaction))
							r.With(auth.RequireRule("portfolio.cash.read")).Get("/{transaction_id}", handlers.Handle(handlers.GetCashTransaction))
							r.With(auth.RequireRule("portfolio.cash.update")).Put("/{transaction_id}", handlers.Handle(handlers.UpdateCashTransaction))
							r.With(auth.RequireRule("portfolio.cash.delete")).Delete("/{transaction_id}", handlers.Handle(handlers.DeleteCashTransaction))
						})
					})
					r.Route("/bonds", func(r chi.Router) {
						r.With(auth.RequireRule("portfolio.bond.read")).Get("/", handlers.Handle(handlers.ListBonds))
						r.With(auth.RequireRule("portfolio.bond.create")).Post("/", handlers.Handle(handlers.CreateBond))
						r.With(auth.RequireRule("portfolio.bond.read")).Get("/snapshots", handlers.Handle(handlers.ListBondSnapshots))
						r.Route("/transactions", func(r chi.Router) {
							r.With(auth.RequireRule("portfolio.bond.read")).Get("/", handlers.Handle(handlers.ListBondTransactions))
							r.With(auth.RequireRule("portfolio.bond.create")).Post("/", handlers.Handle(handlers.CreateBondTransaction))
							r.With(auth.RequireRule("portfolio.bond.read")).Get("/{transaction_id}", handlers.Handle(handlers.GetBondTransaction))
							r.With(auth.RequireRule("portfolio.bond.update")).Put("/{transaction_id}", handlers.Handle(handlers.UpdateBondTransaction))
							r.With(auth.RequireRule("portfolio.bond.delete")).Delete("/{transaction_id}", handlers.Handle(handlers.DeleteBondTransaction))
						})
						r.Route("/{asset_id}", func(r chi.Router) {
							r.With(auth.RequireRule("portfolio.bond.read")).Get("/", handlers.Handle(handlers.GetBond))
							r.With(auth.RequireRule("portfolio.bond.update")).Put("/", handlers.Handle(handlers.UpdateBond))
							r.With(auth.RequireRule("portfolio.bond.update")).Post("/valuation", handlers.Handle(handlers.AdjustBondValuation))
						})
					})
					r.Route("/commodities", func(r chi.Router) {
						r.Route("/gold", func(r chi.Router) {
							r.With(auth.RequireRule("portfolio.commodity.read")).Get("/", handlers.Handle(handlers.GetGold))
							r.With(auth.RequireRule("portfolio.commodity.create")).Post("/", handlers.Handle(handlers.CreateGold))
							r.Route("/transactions", func(r chi.Router) {
								r.With(auth.RequireRule("portfolio.commodity.read")).Get("/", handlers.Handle(handlers.ListGoldTransactions))
								r.With(auth.RequireRule("portfolio.commodity.create")).Post("/", handlers.Handle(handlers.CreateGoldTransaction))
								r.With(auth.RequireRule("portfolio.commodity.read")).Get("/{transaction_id}", handlers.Handle(handlers.GetGoldTransaction))
								r.With(auth.RequireRule("portfolio.commodity.update")).Put("/{transaction_id}", handlers.Handle(handlers.UpdateGoldTransaction))
								r.With(auth.RequireRule("portfolio.commodity.delete")).Delete("/{transaction_id}", handlers.Handle(handlers.DeleteGoldTransaction))
							})
						})
					})
				})
			})
		})
	})
	r.Get("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "docs/openapi.yaml") })
	r.Get("/docs", docsHandler)
	r.Get("/docs/", docsHandler)
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
