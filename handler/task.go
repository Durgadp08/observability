package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Durgadp08/config"
	"github.com/Durgadp08/models"
	"github.com/Durgadp08/repository"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type ImportHandler struct {
	repo   *repository.Repository
	logger *slog.Logger
}

func NewImportHandler(repo *repository.Repository, logger *slog.Logger) *ImportHandler {
	return &ImportHandler{repo: repo, logger: logger}
}

func (h *ImportHandler) CreateImport(w http.ResponseWriter, r *http.Request) {
	ctx, span := config.Tracer.Start(r.Context(), "Handler.CreateImport",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	defer span.End()

	log := withSpan(h.logger, span)
	log.InfoContext(ctx, "create import request received")

	var imp models.Import
	if err := json.NewDecoder(r.Body).Decode(&imp); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request body")
		log.ErrorContext(ctx, "failed to decode request body", "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	span.SetAttributes(
		attribute.String("import.name", imp.Name),
		attribute.String("import.file_type", string(imp.FileType)),
		attribute.Int64("import.file_size", int64(imp.FileSize)),
	)

	log.InfoContext(ctx, "creating import", "name", imp.Name, "file_type", imp.FileType, "file_size", imp.FileSize)

	if err := h.repo.CreateImport(ctx, &imp); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "repository error")
		log.ErrorContext(ctx, "failed to create import", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	span.AddEvent("import.created", trace.WithAttributes(attribute.Int64("import.id", int64(imp.Id))))
	span.SetStatus(codes.Ok, "")
	log.InfoContext(ctx, "import created successfully", "id", imp.Id)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(imp)
}

func (h *ImportHandler) GetImport(w http.ResponseWriter, r *http.Request) {
	ctx, span := config.Tracer.Start(r.Context(), "Handler.GetImport",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	defer span.End()

	log := withSpan(h.logger, span)

	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid id param")
		log.ErrorContext(ctx, "invalid import id", "id", idStr, "error", err)
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	span.SetAttributes(attribute.Int64("import.id", int64(id)))
	log.InfoContext(ctx, "fetching import", "id", id)

	imp, err := h.repo.GetImport(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "repository error")
		log.ErrorContext(ctx, "import not found", "id", id, "error", err)
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	span.SetStatus(codes.Ok, "")
	log.InfoContext(ctx, "import fetched successfully", "id", imp.Id, "name", imp.Name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imp)
}

// withSpan enriches the logger with trace_id and span_id from the active span.
func withSpan(logger *slog.Logger, span trace.Span) *slog.Logger {
	sc := span.SpanContext()
	return logger.With(
		"trace_id", sc.TraceID().String(),
		"span_id", sc.SpanID().String(),
	)
}
