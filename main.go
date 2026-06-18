package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Durgadp08/config"
	"github.com/Durgadp08/config/logger"
	"github.com/Durgadp08/handler"
	"github.com/Durgadp08/middleware"
	"github.com/Durgadp08/repository"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	ctx := context.Background()
	log := logger.GetLogger(ctx)

	if os.Getenv("OTEL_ENABLED") == "true" {
		shutdown, err := config.InitTracer(ctx)
		if err != nil {
			log.Error("failed to init tracer", "error", err)
			os.Exit(1)
		}
		defer shutdown(ctx)
		log.Info("otel tracing enabled", "endpoint", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	}

	db, err := connectDB(log)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	repo := repository.NewRepository(db, log)
	importHandler := handler.NewImportHandler(repo, log)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /imports", importHandler.CreateImport)
	mux.HandleFunc("GET /imports/{id}", importHandler.GetImport)

	addr := getenv("SERVER_ADDR", ":8080")
	srv := &http.Server{
		Addr:    addr,
		Handler: middleware.Logging(log)(mux),
	}

	go func() {
		log.Info("server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info("shutdown signal received")

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
	}

	log.Info("server stopped")
}

func connectDB(log *slog.Logger) (*sql.DB, error) {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
			getenv("DB_USER", "root"),
			getenv("DB_PASSWORD", ""),
			getenv("DB_HOST", "localhost"),
			getenv("DB_PORT", "3306"),
			getenv("DB_NAME", "observe"),
		)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}

	log.Info("database connected", "host", getenv("DB_HOST", "localhost"), "db", getenv("DB_NAME", "observe"))
	return db, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
