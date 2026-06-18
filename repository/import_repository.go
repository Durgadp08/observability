package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Durgadp08/config"
	"github.com/Durgadp08/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func (r *Repository) CreateImport(ctx context.Context, imp *models.Import) error {
	ctx, span := config.Tracer.Start(ctx, "Repository.CreateImport",
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	now := time.Now().UTC().Format(time.RFC3339)
	imp.CreatedAt = now
	imp.UpdatedAt = now

	time.Sleep(10 * time.Second)

	stmt, err := r.db.PrepareContext(ctx, `
		INSERT INTO imports (name, file_path, file_type, file_size, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to prepare statement")
		r.logger.ErrorContext(ctx, "prepare statement failed", "error", err)
		return err
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, imp.Name, imp.FilePath, string(imp.FileType), imp.FileSize, imp.CreatedAt, imp.UpdatedAt)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to execute insert")
		r.logger.ErrorContext(ctx, "insert failed", "error", err)
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		span.RecordError(err)
		r.logger.ErrorContext(ctx, "failed to get the last inserted id", "err", err)
		return err
	}
	imp.Id = uint64(id)

	span.SetAttributes(attribute.Int64("db.import.id", id))
	span.SetStatus(codes.Ok, "")
	r.logger.InfoContext(ctx, "import row inserted", "id", id)
	return nil
}

func (r *Repository) GetImport(ctx context.Context, id uint64) (*models.Import, error) {
	ctx, span := config.Tracer.Start(ctx, "Repository.GetImport",
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	span.SetAttributes(attribute.Int64("db.import.id", int64(id)))

	var imp models.Import
	var errorPath sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, file_path, file_type, file_size, error_path, created_at, updated_at
		FROM imports WHERE id = ?`, id,
	).Scan(&imp.Id, &imp.Name, &imp.FilePath, &imp.FileType, &imp.FileSize, &errorPath, &imp.CreatedAt, &imp.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		span.SetStatus(codes.Error, "import not found")
		r.logger.WarnContext(ctx, "import not found", "id", id)
		return nil, err
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "query failed")
		r.logger.ErrorContext(ctx, "query failed", "id", id, "error", err)
		return nil, err
	}

	if errorPath.Valid {
		imp.ErrorPath = errorPath.String
	}

	span.SetStatus(codes.Ok, "")
	r.logger.InfoContext(ctx, "import row fetched", "id", id)
	return &imp, nil
}
