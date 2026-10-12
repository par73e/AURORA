package syncstate

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Status separates the latest attempt from the last successful data acquisition.
type Status struct {
	Status        string     `json:"status"`
	LastAttemptAt *time.Time `json:"lastAttemptAt,omitempty"`
	LastSuccessAt *time.Time `json:"lastSuccessAt,omitempty"`
}

type executor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func Record(ctx context.Context, db executor, source, target string, attemptedAt time.Time, status string, syncErr error) error {
	var message any
	if syncErr != nil {
		message = syncErr.Error()
	}
	_, err := db.Exec(ctx, `
 INSERT INTO spacecraft_sync_status (source_code, target_id, status, last_attempt_at, last_success_at, error_message)
 VALUES ($1, $2, $3, $4, CASE WHEN $3 = 'success' THEN now() ELSE NULL END, $5)
 ON CONFLICT (source_code, target_id) DO UPDATE SET
 status = EXCLUDED.status, last_attempt_at = EXCLUDED.last_attempt_at,
 last_success_at = COALESCE(EXCLUDED.last_success_at, spacecraft_sync_status.last_success_at),
 error_message = EXCLUDED.error_message
 WHERE spacecraft_sync_status.last_attempt_at <= EXCLUDED.last_attempt_at`, source, target, status, attemptedAt, message)
	return err
}

// New data and its successful acquisition time commit together or both roll back.
func Save(ctx context.Context, pool *pgxpool.Pool, source, target string, write func(pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if err := write(tx); err != nil {
		return err
	}
	if err := Record(ctx, tx, source, target, time.Now().UTC(), "success", nil); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// JSONColumn is shared by all four spacecraft repositories (joined alias st).
const JSONColumn = `CASE WHEN st.target_id IS NULL THEN NULL ELSE jsonb_build_object(
 'status', st.status, 'lastAttemptAt', st.last_attempt_at, 'lastSuccessAt', st.last_success_at) END`

func Decode(raw []byte) (*Status, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var status Status
	if err := json.Unmarshal(raw, &status); err != nil {
		return nil, err
	}
	return &status, nil
}
