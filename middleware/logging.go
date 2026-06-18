package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/Durgadp08/config"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type contextKey string

const TraceIDKey contextKey = "trace_id"

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := config.Tracer.Start(r.Context(), r.Method+" "+r.URL.Path,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("http.method", r.Method),
					attribute.String("http.path", r.URL.Path),
					attribute.String("http.user_agent", r.UserAgent()),
				),
			)
			defer span.End()

			sc := span.SpanContext()
			ctx = context.WithValue(ctx, TraceIDKey, sc.TraceID().String())

			log := logger.With(
				"trace_id", sc.TraceID().String(),
				"span_id", sc.SpanID().String(),
				"method", r.Method,
				"path", r.URL.Path,
			)

			log.InfoContext(ctx, "request received")

			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()

			next.ServeHTTP(rw, r.WithContext(ctx))

			duration := time.Since(start)

			span.SetAttributes(
				attribute.Int("http.status_code", rw.status),
				attribute.Int64("http.duration_ms", duration.Milliseconds()),
			)

			if rw.status >= 500 {
				span.SetStatus(codes.Error, http.StatusText(rw.status))
			} else {
				span.SetStatus(codes.Ok, "")
			}

			log.InfoContext(ctx, "request completed",
				"status", rw.status,
				"duration_ms", duration.Milliseconds(),
			)
		})
	}
}
