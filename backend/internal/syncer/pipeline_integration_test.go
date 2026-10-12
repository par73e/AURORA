package syncer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"aurora/backend/internal/database"
	"aurora/backend/internal/httpapi"
	"aurora/backend/internal/mars"
	"aurora/backend/internal/moon"
	"aurora/backend/internal/orbit"
	"aurora/backend/internal/voyage"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Tests require an explicitly selected disposable local database. Never fall back
// to the developer's application database, and isolate every case in its own schema.
func integrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("AURORA_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set AURORA_TEST_DATABASE_URL to a disposable local aurora_test database")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "aurora_test" || (cfg.ConnConfig.Host != "127.0.0.1" && cfg.ConnConfig.Host != "localhost") {
		t.Fatal("integration tests require local aurora_test")
	}
	ctx := context.Background()
	admin, err := pgxpool.NewWithConfig(ctx, cfg.Copy())
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
		admin.Close()
	})
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return pool
}

type redirectedTransport struct {
	base   http.RoundTripper
	target *url.URL
}

func (r redirectedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	destination := *req.URL
	destination.Scheme, destination.Host = r.target.Scheme, r.target.Host
	clone.URL = &destination
	return r.base.RoundTrip(clone)
}
func fixtureClient(server *httptest.Server) *http.Client {
	target, _ := url.Parse(server.URL)
	return &http.Client{Transport: redirectedTransport{base: server.Client().Transport, target: target}, Timeout: time.Second}
}

func assertRun(t *testing.T, pool *pgxpool.Pool, source string, success bool, written int) {
	t.Helper()
	var got bool
	var records int
	var finished *time.Time
	err := pool.QueryRow(context.Background(), "SELECT success, records_written, finished_at FROM sync_runs WHERE source_code=$1 ORDER BY id DESC LIMIT 1", source).Scan(&got, &records, &finished)
	if err != nil {
		t.Fatal(err)
	}
	if got != success || records != written || finished == nil {
		t.Fatalf("run got success=%v records=%d finished=%v, want %v/%d", got, records, finished, success, written)
	}
}

func TestLaunchPipelineIntegration(t *testing.T) {
	pool := integrationPool(t)
	failSecond := false
	now := time.Now().UTC()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		second := r.URL.Query().Get("offset") != ""
		if second && failSecond {
			http.Error(w, "unavailable", 503)
			return
		}
		start, end := 0, 20
		if second {
			start, end = 20, 25
		}
		results := []map[string]any{}
		for i := start; i < end; i++ {
			results = append(results, map[string]any{"id": fmt.Sprintf("launch-%02d", i), "name": "Test launch", "net": now.Add(time.Duration(i+1) * time.Hour).Format(time.RFC3339)})
		}
		var next any
		if !second {
			next = "/2.3.0/launches/upcoming/?offset=20"
		} else {
			results = append(results, map[string]any{"id": "launch-00", "name": "Duplicate", "net": now.Add(time.Hour).Format(time.RFC3339)}, map[string]any{"id": "outside-window", "name": "Outside", "net": now.Add(31 * 24 * time.Hour).Format(time.RFC3339)})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"results": results, "next": next})
	}))
	defer server.Close()
	repo := orbit.NewRepository(pool)
	s := New(repo)
	s.client = fixtureClient(server)
	if err := s.SyncLaunches(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertRun(t, pool, "launch_library_2", true, 25)
	router := httpapi.Router(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	record := httptest.NewRecorder()
	router.ServeHTTP(record, httptest.NewRequest(http.MethodGet, "/api/v1/orbit/overview", nil))
	if record.Code != 200 {
		t.Fatalf("overview HTTP %d: %s", record.Code, record.Body.String())
	}
	var overview orbit.Overview
	if err := json.Unmarshal(record.Body.Bytes(), &overview); err != nil {
		t.Fatal(err)
	}
	if len(overview.Events) != 25 {
		t.Fatalf("API returned %d events, want 25", len(overview.Events))
	}
	seen := map[string]bool{}
	for _, event := range overview.Events {
		if seen[event.ExternalID] || event.ExternalID == "outside-window" {
			t.Fatalf("invalid returned event %s", event.ExternalID)
		}
		seen[event.ExternalID] = true
	}
	failSecond = true
	if err := s.SyncLaunches(context.Background()); err == nil {
		t.Fatal("incomplete pagination reported success")
	}
	assertRun(t, pool, "launch_library_2", false, 0)
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM launch_events").Scan(&count); err != nil || count != 25 {
		t.Fatalf("old launch data not preserved: %d/%v", count, err)
	}
}

func TestSpacecraftPipelineFailureMatrix(t *testing.T) {
	for _, source := range []string{"celestrak", "jpl_horizons_moon", "jpl_horizons_mars", "jpl_horizons_voyage"} {
		for _, mode := range []string{"success", "all_failed", "partial", "unsupported", "timeout", "save_failed", "status_failed"} {
			t.Run(source+"/"+mode, func(t *testing.T) {
				pool := integrationPool(t)
				ctx := context.Background()
				repo := orbit.NewRepository(pool)
				s := NewWithMoonVoyageMars(repo, moon.NewRepository(pool), mars.NewRepository(pool), voyage.NewRepository(pool))
				ids := []string{}
				table, snapshotTable, key := "", "", ""
				switch source {
				case "celestrak":
					ids = []string{"iss", "tianhe"}
					table = "spacecraft"
					snapshotTable = "orbit_snapshots"
					key = "spacecraft_id"
				case "jpl_horizons_moon":
					ids = []string{"lro", "chandrayaan-2"}
					table = "moon_spacecraft"
					snapshotTable = "moon_orbit_snapshots"
					key = "spacecraft_id"
				case "jpl_horizons_mars":
					ids = []string{"mro", "maven"}
					table = "mars_spacecraft"
					snapshotTable = "mars_orbit_snapshots"
					key = "spacecraft_id"
				case "jpl_horizons_voyage":
					ids = []string{"voyager-1", "voyager-2"}
					table = "deep_space_probes"
					snapshotTable = "probe_position_samples"
					key = "probe_id"
				}
				if _, err := pool.Exec(ctx, "DELETE FROM "+table+" WHERE id <> ALL($1)", ids); err != nil {
					t.Fatal(err)
				}
				currentMode := "success"
				epoch := time.Now().UTC().Truncate(time.Second)
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					command := r.URL.Query().Get("COMMAND")
					first := command == "-85" || command == "-74" || command == "-31" || r.URL.Query().Get("CATNR") == "25544"
					if currentMode == "timeout" {
						<-r.Context().Done()
						return
					}
					if currentMode == "all_failed" || (currentMode == "partial" && !first) {
						http.Error(w, "fixture unavailable", 503)
						return
					}
					if source == "celestrak" {
						_ = json.NewEncoder(w).Encode([]map[string]any{{"EPOCH": epoch.Format("2006-01-02T15:04:05.000000"), "MEAN_MOTION": 15}})
						return
					}
					if source == "jpl_horizons_voyage" {
						fmt.Fprintf(w, "$$SOE\n2450000, A.D. %s, 1.234567e08, -5.678901e07, 1.000e06\n2450001, A.D. %s, 1.240000e08, -5.700000e07, 1.010e06\n$$EOE\n", epoch.Add(-24*time.Hour).Format("2006-Jan-02 15:04:05.0000"), epoch.Add(24*time.Hour).Format("2006-Jan-02 15:04:05.0000"))
						return
					}
					x, vy := 2000., 1.4
					if source == "jpl_horizons_mars" {
						x, vy = 4000, 3.1
					}
					fmt.Fprintf(w, "$$SOE\n2450000, A.D. %s, %.6f, 0, 0, 0, %.6f, 0\n$$EOE\n", epoch.Format("2006-Jan-02 15:04:05.0000"), x, vy)
				}))
				defer server.Close()
				s.client = fixtureClient(server)
				run := func(ctx context.Context) error {
					switch source {
					case "celestrak":
						return s.SyncCelesTrak(ctx)
					case "jpl_horizons_moon":
						return s.SyncMoonSpacecraft(ctx)
					case "jpl_horizons_mars":
						return s.SyncMarsSpacecraft(ctx)
					default:
						return s.SyncDeepSpaceProbes(ctx)
					}
				}
				if err := run(ctx); err != nil {
					t.Fatal("seed successful sync:", err)
				}
				old := map[string]time.Time{}
				for _, id := range ids {
					var stamp time.Time
					if err := pool.QueryRow(ctx, "SELECT last_success_at FROM spacecraft_sync_status WHERE source_code=$1 AND target_id=$2", source, id).Scan(&stamp); err != nil {
						t.Fatal(err)
					}
					old[id] = stamp
				}
				currentMode = mode
				if mode == "unsupported" {
					switch source {
					case "celestrak":
						if _, err := pool.Exec(ctx, "DELETE FROM spacecraft"); err != nil {
							t.Fatal(err)
						}
					case "jpl_horizons_moon", "jpl_horizons_mars":
						if _, err := pool.Exec(ctx, "UPDATE "+table+" SET kind='stationary'"); err != nil {
							t.Fatal(err)
						}
					default:
						if _, err := pool.Exec(ctx, "UPDATE deep_space_probes SET naif_id=''"); err != nil {
							t.Fatal(err)
						}
					}
				}
				if mode == "save_failed" {
					// A rejecting trigger verifies database write failures and transaction rollback.
					if _, err := pool.Exec(ctx, "CREATE FUNCTION reject_fixture_write() RETURNS trigger LANGUAGE plpgsql AS 'BEGIN RAISE EXCEPTION ''fixture rejection''; END'; CREATE TRIGGER fixture_reject BEFORE INSERT ON "+snapshotTable+" FOR EACH ROW EXECUTE FUNCTION reject_fixture_write()"); err != nil {
						t.Fatal(err)
					}
				}
				if mode == "status_failed" {
					if _, err := pool.Exec(ctx, "CREATE FUNCTION reject_status_write() RETURNS trigger LANGUAGE plpgsql AS 'BEGIN RAISE EXCEPTION ''status rejection''; END'; CREATE TRIGGER status_reject BEFORE INSERT ON spacecraft_sync_status FOR EACH ROW WHEN (NEW.status = 'success') EXECUTE FUNCTION reject_status_write()"); err != nil {
						t.Fatal(err)
					}
				}
				attemptCtx := ctx
				if mode == "timeout" {
					var cancel context.CancelFunc
					attemptCtx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
					defer cancel()
				}
				err := run(attemptCtx)
				if (err == nil) != (mode == "success") {
					t.Fatalf("mode %s returned %v", mode, err)
				}
				if mode == "timeout" && !strings.Contains(err.Error(), "deadline") {
					t.Fatalf("timeout not preserved: %v", err)
				}
				expectedWrites := 0
				if mode == "success" {
					expectedWrites = 2
				} else if mode == "partial" {
					expectedWrites = 1
				}
				if source == "jpl_horizons_voyage" {
					expectedWrites *= 2
				}
				assertRun(t, pool, source, mode == "success", expectedWrites)
				if source == "celestrak" && mode == "unsupported" {
					return
				}
				for index, id := range ids {
					var status string
					var stamp time.Time
					var attempt time.Time
					if err := pool.QueryRow(ctx, "SELECT status,last_success_at,last_attempt_at FROM spacecraft_sync_status WHERE source_code=$1 AND target_id=$2", source, id).Scan(&status, &stamp, &attempt); err != nil {
						t.Fatal(err)
					}
					want := "failed"
					if mode == "success" || (mode == "partial" && index == 0) {
						want = "success"
					}
					if mode == "unsupported" {
						want = "unsupported"
					}
					if mode == "timeout" {
						want = "canceled"
					}
					if status != want {
						t.Fatalf("%s got status %s, want %s", id, status, want)
					}
					if want != "success" && !stamp.Equal(old[id]) {
						t.Fatalf("%s failed attempt advanced success time", id)
					}
					if want == "success" && !stamp.After(old[id]) {
						t.Fatalf("%s success time did not advance", id)
					}
					var count int
					if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+snapshotTable+" WHERE "+key+"=$1", id).Scan(&count); err != nil || count < 1 {
						t.Fatalf("%s lost previous data: %d/%v", id, count, err)
					}
				}
				// Read through the real repositories to verify API-ready per-target status.
				switch source {
				case "celestrak":
					data, err := repo.Overview(ctx)
					if err != nil || len(data.Spacecraft) != 2 || data.Spacecraft[0].SyncStatus == nil {
						t.Fatalf("orbit status read failed: %v", err)
					}
				case "jpl_horizons_moon":
					data, err := s.moonRepo.ListSpacecraft(ctx)
					if err != nil || len(data) != 2 || data[0].SyncStatus == nil || data[0].Snapshot.SyncedAt == "" {
						t.Fatalf("moon status read failed: %v", err)
					}
				case "jpl_horizons_mars":
					data, err := s.marsRepo.ListSpacecraft(ctx)
					if err != nil || len(data) != 2 || data[0].SyncStatus == nil || data[0].Snapshot.SyncedAt == "" {
						t.Fatalf("mars status read failed: %v", err)
					}
				default:
					data, err := s.voyageRepo.ListProbes(ctx)
					if err != nil || len(data) != 2 || data[0].SyncStatus == nil {
						t.Fatalf("voyage status read failed: %v", err)
					}
				}
			})
		}
	}
}

func TestTargetStatusPreservesSuccessAndRejectsOldAttempts(t *testing.T) {
	pool := integrationPool(t)
	repo := orbit.NewRepository(pool)
	ctx := context.Background()
	newer := time.Now().UTC()
	if err := repo.RecordTargetSync(ctx, "celestrak", "iss", newer, "success", nil); err != nil {
		t.Fatal(err)
	}
	var successAt time.Time
	if err := pool.QueryRow(ctx, "SELECT last_success_at FROM spacecraft_sync_status WHERE source_code='celestrak' AND target_id='iss'").Scan(&successAt); err != nil {
		t.Fatal(err)
	}
	if err := repo.RecordTargetSync(ctx, "celestrak", "iss", newer.Add(-time.Hour), "failed", fmt.Errorf("old request")); err != nil {
		t.Fatal(err)
	}
	var status string
	var after time.Time
	if err := pool.QueryRow(ctx, "SELECT status,last_success_at FROM spacecraft_sync_status WHERE source_code='celestrak' AND target_id='iss'").Scan(&status, &after); err != nil {
		t.Fatal(err)
	}
	if status != "success" || !after.Equal(successAt) {
		t.Fatal("old attempt overwrote newer success")
	}
	if err := repo.RecordTargetSync(ctx, "celestrak", "iss", newer.Add(time.Hour), "failed", fmt.Errorf("new failed request")); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT status,last_success_at FROM spacecraft_sync_status WHERE source_code='celestrak' AND target_id='iss'").Scan(&status, &after); err != nil {
		t.Fatal(err)
	}
	if status != "failed" || !after.Equal(successAt) {
		t.Fatal("failed attempt advanced successful data time")
	}
}

func TestLiveLaunchPipelineIntegration(t *testing.T) {
	if os.Getenv("AURORA_TEST_LIVE_LAUNCHES") != "1" {
		t.Skip("explicit opt-in required for public Launch Library requests")
	}
	pool := integrationPool(t)
	repo := orbit.NewRepository(pool)
	s := New(repo)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if err := s.SyncLaunches(ctx); err != nil {
		t.Fatal(err)
	}
	overview, err := repo.Overview(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var records int
	if err := pool.QueryRow(ctx, "SELECT records_written FROM sync_runs WHERE source_code='launch_library_2' AND success=true ORDER BY id DESC LIMIT 1").Scan(&records); err != nil {
		t.Fatal(err)
	}
	if records == 0 || len(overview.Events) == 0 {
		t.Fatal("live launch sync returned no data")
	}
	t.Logf("real upstream: %d records persisted; %d events in current 30-day API window", records, len(overview.Events))
}

func TestSyncStatusMigrationBackfillsAcquisitionTime(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	acquired := time.Now().UTC().Add(-72 * time.Hour).Truncate(time.Second)
	_, err := pool.Exec(ctx, "INSERT INTO orbit_snapshots(spacecraft_id,epoch,raw_omm,synced_at) VALUES('iss',$1,'{}',$1)", acquired)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "DROP TABLE spacecraft_sync_status; DELETE FROM schema_migrations WHERE version='053_spacecraft_sync_status.sql'"); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	var stamp time.Time
	if err := pool.QueryRow(ctx, "SELECT last_success_at FROM spacecraft_sync_status WHERE source_code='celestrak' AND target_id='iss'").Scan(&stamp); err != nil {
		t.Fatal(err)
	}
	if !stamp.Equal(acquired) {
		t.Fatalf("migration replaced acquisition time: %v != %v", stamp, acquired)
	}
}
