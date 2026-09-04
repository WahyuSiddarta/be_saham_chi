package logger

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// ChiLogFormatter routes Chi's request log through the configured Zerolog instance.
func ChiLogFormatter() middleware.LogFormatter {
	return chiLogFormatter{}
}

type chiLogFormatter struct{}

func (chiLogFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	return chiLogEntry{request: r}
}

type chiLogEntry struct {
	request *http.Request
}

func (entry chiLogEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	if status == 0 {
		status = http.StatusOK
	}

	event := instance.Info()
	if status >= http.StatusInternalServerError {
		event = instance.Error()
	}

	event.
		Str("request_id", middleware.GetReqID(entry.request.Context())).
		Str("method", entry.request.Method).
		Str("path", entry.request.URL.Path).
		Str("remote_ip", entry.request.RemoteAddr).
		Int("status", status).
		Int("bytes", bytes).
		Dur("duration", elapsed).
		Msg("HTTP request")
}

func (entry chiLogEntry) Panic(v interface{}, stack []byte) {
	instance.Error().
		Interface("panic", v).
		Bytes("stack", stack).
		Str("request_id", middleware.GetReqID(entry.request.Context())).
		Msg("HTTP request panic")
}
