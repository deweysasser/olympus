package middleware

import (
	"github.com/rs/zerolog/log"
	"net/http"
	"time"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lrw := &loggingResponseWriter{w, 200}
		start := time.Now()
		next.ServeHTTP(lrw, r)
		log.Info().Str("uri", r.RequestURI).Dur("duration", time.Since(start)).Int("status", lrw.statusCode).Msg("handled")
	})
}
