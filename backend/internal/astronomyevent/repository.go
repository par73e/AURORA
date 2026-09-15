package astronomyevent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) List(ctx context.Context, query ListQuery) ([]Event, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT e.id, e.kind, e.title_zh, e.title_en, e.starts_at, e.ends_at,
		       e.date_label, e.summary, e.origin, s.name, e.source_url,
		       e.verified_at::text, e.geometry, e.presentation
		FROM astronomy_events e
		JOIN data_sources s ON s.code = e.source_code
		WHERE e.starts_at >= $1 AND e.starts_at < $2
		ORDER BY e.starts_at, e.id`, query.From, query.To)
	if err != nil {
		return nil, fmt.Errorf("query astronomy events: %w", err)
	}
	defer rows.Close()

	events := []Event{}
	for rows.Next() {
		var event Event
		if err := rows.Scan(
			&event.ID, &event.Kind, &event.Title, &event.TitleEN, &event.StartsAt, &event.EndsAt,
			&event.DateLabel, &event.Summary, &event.Origin, &event.SourceName, &event.SourceURL,
			&event.VerifiedAt, &event.Geometry, &event.Presentation,
		); err != nil {
			return nil, fmt.Errorf("scan astronomy event: %w", err)
		}
		event.StartsAt = event.StartsAt.UTC()
		if event.EndsAt != nil {
			value := event.EndsAt.UTC()
			event.EndsAt = &value
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate astronomy events: %w", err)
	}
	return events, nil
}

// ReplaceComputed 原子替换窗口内由 AURORA 模型生成的事件，绝不触碰人工校订或外部预测事件。
func (r *Repository) ReplaceComputed(ctx context.Context, query ListQuery, events []Event) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin replace computed astronomy events: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(ctx, `DELETE FROM astronomy_events WHERE origin = 'computed' AND starts_at >= $1 AND starts_at < $2`, query.From, query.To); err != nil {
		return fmt.Errorf("clear computed astronomy events: %w", err)
	}
	for _, event := range events {
		geometry, presentation, err := normalizedEventDocuments(event)
		if err != nil {
			return fmt.Errorf("normalize computed astronomy event %s: %w", event.ID, err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO astronomy_events
				(id, kind, title_zh, title_en, starts_at, ends_at, date_label, summary, origin, source_code, source_url, verified_at, geometry, presentation)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'computed', COALESCE(NULLIF($9, ''), 'aurora_astronomy_model'), $10, $11, $12, $13)
			ON CONFLICT (id) DO UPDATE SET
				kind = EXCLUDED.kind, title_zh = EXCLUDED.title_zh, title_en = EXCLUDED.title_en,
				starts_at = EXCLUDED.starts_at, ends_at = EXCLUDED.ends_at, date_label = EXCLUDED.date_label,
				summary = EXCLUDED.summary, origin = EXCLUDED.origin, source_code = EXCLUDED.source_code,
				source_url = EXCLUDED.source_url, verified_at = EXCLUDED.verified_at,
				geometry = EXCLUDED.geometry, presentation = EXCLUDED.presentation, updated_at = now()`,
			event.ID, event.Kind, event.Title, event.TitleEN, event.StartsAt, event.EndsAt, event.DateLabel,
			event.Summary, event.SourceCode, event.SourceURL, event.VerifiedAt, geometry, presentation,
		); err != nil {
			return fmt.Errorf("upsert computed astronomy event %s: %w", event.ID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit computed astronomy events: %w", err)
	}
	return nil
}

func (r *Repository) ReplaceEphemerisSamples(ctx context.Context, body string, from, to time.Time, samples []EphemerisSample) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin replace %s ephemeris: %w", body, err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(ctx, `DELETE FROM astronomy_ephemeris_samples WHERE body = $1 AND epoch >= $2 AND epoch < $3`, body, from, to); err != nil {
		return fmt.Errorf("clear %s ephemeris: %w", body, err)
	}
	for _, sample := range samples {
		if _, err := tx.Exec(ctx, `
			INSERT INTO astronomy_ephemeris_samples(body, epoch, x_au, y_au, z_au, source_url)
			VALUES($1, $2, $3, $4, $5, $6)
			ON CONFLICT(body, epoch) DO UPDATE SET x_au=EXCLUDED.x_au, y_au=EXCLUDED.y_au, z_au=EXCLUDED.z_au, source_url=EXCLUDED.source_url, synced_at=now()`,
			sample.Body, sample.Epoch, sample.XAU, sample.YAU, sample.ZAU, sample.SourceURL,
		); err != nil {
			return fmt.Errorf("upsert %s ephemeris: %w", body, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit %s ephemeris: %w", body, err)
	}
	return nil
}

func (r *Repository) StartSourceSync(ctx context.Context, sourceCode string) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `INSERT INTO sync_runs(source_code) VALUES($1) RETURNING id`, sourceCode).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("start astronomy source sync %s: %w", sourceCode, err)
	}
	return id, nil
}

func (r *Repository) FinishSourceSync(ctx context.Context, id int64, records int, syncErr error) error {
	success := syncErr == nil
	var message *string
	if syncErr != nil {
		value := syncErr.Error()
		message = &value
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE sync_runs
		SET finished_at = now(), success = $2, records_written = $3, error_message = $4
		WHERE id = $1`, id, success, records, message)
	if err != nil {
		return fmt.Errorf("finish astronomy source sync: %w", err)
	}
	return nil
}

// SaveSourceSnapshot 以内容哈希去重。相同内容重复抓取不增加新快照，
// 但同步运行本身仍会记录一次成功检查。
func (r *Repository) SaveSourceSnapshot(ctx context.Context, snapshot SourceSnapshot) (bool, error) {
	if !json.Valid(snapshot.RawPayload) {
		return false, errors.New("astronomy source snapshot payload must be valid JSON")
	}
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO astronomy_event_source_snapshots
			(source_code, coverage_start, coverage_end, source_url, payload_hash, raw_payload, records_written)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (source_code, payload_hash) DO NOTHING
		RETURNING id`,
		snapshot.SourceCode, snapshot.CoverageStart, snapshot.CoverageEnd, snapshot.SourceURL,
		snapshot.PayloadHash, snapshot.RawPayload, snapshot.RecordsWritten,
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("save astronomy source snapshot: %w", err)
	}
	return true, nil
}

var _ Store = (*Repository)(nil)
var _ SourceSnapshotStore = (*Repository)(nil)
var _ ComputedEventStore = (*Repository)(nil)
var _ EphemerisStore = (*Repository)(nil)
var _ ExternalSourceStore = (*Repository)(nil)

// ReplaceExternalForecast 原子替换某个权威资料页的 external_forecast 事件。
// 同一 source_code 下可有多个独立资料页（例如 NASA 日食与月食目录），
// 因而替换范围必须包含 source_url，不能相互清空。
// 失败时事务回滚，旧事件保留。
func (r *Repository) ReplaceExternalForecast(ctx context.Context, sourceCode, sourceURL string, events []Event) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin replace external forecast: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(ctx, `DELETE FROM astronomy_events WHERE origin = 'external_forecast' AND source_code = $1 AND source_url = $2`, sourceCode, sourceURL); err != nil {
		return 0, fmt.Errorf("clear external forecast events: %w", err)
	}
	written := 0
	for _, event := range events {
		geometry, presentation, err := normalizedEventDocuments(event)
		if err != nil {
			return written, fmt.Errorf("normalize external forecast event %s: %w", event.ID, err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO astronomy_events
				(id, kind, title_zh, title_en, starts_at, ends_at, date_label, summary, origin, source_code, source_url, verified_at, geometry, presentation)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'external_forecast', $9, $10, $11, $12, $13)
			ON CONFLICT (id) DO UPDATE SET
				kind = EXCLUDED.kind, title_zh = EXCLUDED.title_zh, title_en = EXCLUDED.title_en,
				starts_at = EXCLUDED.starts_at, ends_at = EXCLUDED.ends_at, date_label = EXCLUDED.date_label,
				summary = EXCLUDED.summary, origin = EXCLUDED.origin, source_code = EXCLUDED.source_code,
				source_url = EXCLUDED.source_url, verified_at = EXCLUDED.verified_at,
				geometry = EXCLUDED.geometry, presentation = EXCLUDED.presentation, updated_at = now()`,
			event.ID, event.Kind, event.Title, event.TitleEN, event.StartsAt, event.EndsAt, event.DateLabel,
			event.Summary, sourceCode, sourceURL, event.VerifiedAt, geometry, presentation,
		); err != nil {
			return written, fmt.Errorf("upsert external forecast event %s: %w", event.ID, err)
		}
		written++
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit external forecast events: %w", err)
	}
	return written, nil
}

// normalizedEventDocuments ensures the NOT NULL JSONB columns always receive
// JSON objects. Empty optional documents become {}, while malformed or
// non-object JSON remains an explicit producer error instead of corrupting the
// event contract.
func normalizedEventDocuments(event Event) (json.RawMessage, json.RawMessage, error) {
	geometry, err := normalizedJSONObject(event.Geometry)
	if err != nil {
		return nil, nil, fmt.Errorf("geometry: %w", err)
	}
	presentation, err := normalizedJSONObject(event.Presentation)
	if err != nil {
		return nil, nil, fmt.Errorf("presentation: %w", err)
	}
	return geometry, presentation, nil
}

func normalizedJSONObject(raw json.RawMessage) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return json.RawMessage(`{}`), nil
	}
	var object map[string]any
	if err := json.Unmarshal(trimmed, &object); err != nil || object == nil {
		if err == nil {
			err = errors.New("must be a JSON object")
		}
		return nil, err
	}
	return raw, nil
}
