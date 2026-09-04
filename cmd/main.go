package main

import (
	"net/http"
	"os"

	"github.com/WahyuSiddarta/be_saham_chi/internal/logger"
)

func main() {
	log := logger.Configure(os.Stdout)
	app := Application{config: Config{addr: ":8080"}}

	log.Info().Str("addr", app.config.addr).Msg("API server starting")
	if err := http.ListenAndServe(app.config.addr, app.routes()); err != nil {
		log.Fatal().Err(err).Msg("API server stopped")
	}
}
