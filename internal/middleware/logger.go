package middleware

import (
	"net/http"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

// RequestLogger logs HTTP requests through the supplied Zerolog logger.
func RequestLogger(log *zerolog.Logger) func(http.Handler) http.Handler {
	return chimiddleware.RequestLogger(chiLogFormatter{log: log})
}

type chiLogFormatter struct {
	log *zerolog.Logger
}

func (formatter chiLogFormatter) NewLogEntry(r *http.Request) chimiddleware.LogEntry {
	return chiLogEntry{request: r, log: formatter.log}
}

type chiLogEntry struct {
	log     *zerolog.Logger
	request *http.Request
}

func (entry chiLogEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	if status == 0 {
		status = http.StatusOK
	}

	event := entry.log.Info()
	if status >= http.StatusInternalServerError {
		event = entry.log.Error()
	}

	event.
		Str("request_id", chimiddleware.GetReqID(entry.request.Context())).
		Str("method", entry.request.Method).
		Str("path", entry.request.URL.Path).
		Str("remote_ip", entry.request.RemoteAddr).
		Int("status", status).
		Int("bytes", bytes).
		Dur("duration", elapsed).
		Msg("HTTP request")
}

func (entry chiLogEntry) Panic(v interface{}, stack []byte) {
	entry.log.Error().
		Interface("panic", v).
		Bytes("stack", stack).
		Str("request_id", chimiddleware.GetReqID(entry.request.Context())).
		Msg("HTTP request panic")
}
