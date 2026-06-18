package logger

import (
	"context"
	"log/slog"
	"os"
)

func GetLogger(ctx context.Context) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}
