package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type pelicanCheckRepository struct {
	db *sql.DB
}

func NewPelicanCheckRepository(db *sql.DB) service.PelicanCheckRepository {
	return &pelicanCheckRepository{db: db}
}

func (r *pelicanCheckRepository) Create(ctx context.Context, check *service.PelicanCheck) error {
	signals, err := json.Marshal(check.Signals)
	if err != nil {
		return err
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO pelican_checks (account_id, model, status, verdict, confidence, reason, signals, output_excerpt, output_html, error_message, judge_model, latency_ms, started_at, finished_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW())
		RETURNING id, created_at
	`, check.AccountID, check.Model, check.Status, check.Verdict, check.Confidence, check.Reason,
		string(signals), check.OutputExcerpt, check.OutputHTML, check.ErrorMessage, check.JudgeModel, check.LatencyMs,
		check.StartedAt, check.FinishedAt)
	return row.Scan(&check.ID, &check.CreatedAt)
}

func (r *pelicanCheckRepository) Finish(ctx context.Context, check *service.PelicanCheck) error {
	signals, err := json.Marshal(check.Signals)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE pelican_checks
		SET status = $2, verdict = $3, confidence = $4, reason = $5, signals = $6,
		    output_excerpt = $7, output_html = $8, error_message = $9, judge_model = $10,
		    latency_ms = $11, finished_at = $12
		WHERE id = $1
	`, check.ID, check.Status, check.Verdict, check.Confidence, check.Reason, string(signals),
		check.OutputExcerpt, check.OutputHTML, check.ErrorMessage, check.JudgeModel, check.LatencyMs, check.FinishedAt)
	return err
}

func (r *pelicanCheckRepository) ListRecentByAccount(ctx context.Context, accountID int64, limit int) ([]*service.PelicanCheck, error) {
	return r.listRecent(ctx, accountID, limit, false)
}

func (r *pelicanCheckRepository) ListRecentByAccountWithHTML(ctx context.Context, accountID int64, limit int) ([]*service.PelicanCheck, error) {
	return r.listRecent(ctx, accountID, limit, true)
}

func (r *pelicanCheckRepository) listRecent(ctx context.Context, accountID int64, limit int, includeHTML bool) ([]*service.PelicanCheck, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	columns := `id, account_id, model, status, verdict, confidence, reason, signals,
		       output_excerpt, error_message, judge_model, latency_ms, started_at, finished_at, created_at`
	if includeHTML {
		columns = `id, account_id, model, status, verdict, confidence, reason, signals,
		       output_excerpt, output_html, error_message, judge_model, latency_ms, started_at, finished_at, created_at`
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+columns+`
		FROM pelican_checks
		WHERE account_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2
	`, accountID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	checks := make([]*service.PelicanCheck, 0, limit)
	for rows.Next() {
		check := &service.PelicanCheck{}
		var signals string
		var finishedAt sql.NullTime
		dest := []any{
			&check.ID, &check.AccountID, &check.Model, &check.Status, &check.Verdict,
			&check.Confidence, &check.Reason, &signals, &check.OutputExcerpt,
		}
		if includeHTML {
			dest = append(dest, &check.OutputHTML)
		}
		dest = append(dest, &check.ErrorMessage, &check.JudgeModel, &check.LatencyMs, &check.StartedAt,
			&finishedAt, &check.CreatedAt)
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		if finishedAt.Valid {
			value := finishedAt.Time
			check.FinishedAt = &value
		}
		if signals != "" {
			_ = json.Unmarshal([]byte(signals), &check.Signals)
		}
		checks = append(checks, check)
	}
	return checks, rows.Err()
}
