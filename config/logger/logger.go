package logger

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelslog"
)

const instrumentationName = "service"

type fanoutHandler struct {
	stdout slog.Handler
	otel   slog.Handler
}

func newFanoutHandler(stdout, otel slog.Handler) slog.Handler {
	return &fanoutHandler{
		stdout: stdout,
		otel:   otel,
	}
}

func (h *fanoutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.stdout.Enabled(ctx, level) || h.otel.Enabled(ctx, level)
}

func (h *fanoutHandler) Handle(ctx context.Context, record slog.Record) error {
	var err error

	if h.stdout.Enabled(ctx, record.Level) {
		if e := h.stdout.Handle(ctx, record); e != nil {
			err = e
		}
	}

	if h.otel.Enabled(ctx, record.Level) {
		if e := h.otel.Handle(ctx, record); e != nil && err == nil {
			err = e
		}
	}

	return err
}

func (h *fanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return newFanoutHandler(
		h.stdout.WithAttrs(attrs),
		h.otel.WithAttrs(attrs),
	)
}

func (h *fanoutHandler) WithGroup(name string) slog.Handler {
	return newFanoutHandler(
		h.stdout.WithGroup(name),
		h.otel.WithGroup(name),
	)
}

func GetLogger(ctx context.Context) *slog.Logger {
	stdoutHandler := slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level: slog.LevelInfo,
		},
	)

	otelHandler := otelslog.NewHandler(
		instrumentationName,
	)

	handler := newFanoutHandler(
		stdoutHandler,
		otelHandler,
	)

	log := slog.New(handler)

	slog.SetDefault(log)

	return log
}
