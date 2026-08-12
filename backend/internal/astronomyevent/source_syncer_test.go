package astronomyevent

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type sourceSnapshotStoreStub struct {
	started   []string
	finished  int
	snapshots []SourceSnapshot
}

func (s *sourceSnapshotStoreStub) StartSourceSync(_ context.Context, source string) (int64, error) {
	s.started = append(s.started, source)
	return int64(len(s.started)), nil
}

func (s *sourceSnapshotStoreStub) FinishSourceSync(_ context.Context, _ int64, _ int, _ error) error {
	s.finished++
	return nil
}

func (s *sourceSnapshotStoreStub) SaveSourceSnapshot(_ context.Context, snapshot SourceSnapshot) (bool, error) {
	s.snapshots = append(s.snapshots, snapshot)
	return true, nil
}

func (s *sourceSnapshotStoreStub) ReplaceExternalForecast(_ context.Context, events []Event) (int, error) {
	return len(events), nil
}

type roundTripper func(*http.Request) (*http.Response, error)

func (fn roundTripper) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }

func TestSourceSyncerRecordsOfficialSourceSnapshots(t *testing.T) {
	store := &sourceSnapshotStoreStub{}
	syncer := NewSourceSyncer(store)
	syncer.now = func() time.Time { return time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC) }
	syncer.client = &http.Client{Transport: roundTripper(func(request *http.Request) (*http.Response, error) {
		body := `{"endpoint":"` + request.URL.Path + `"}`
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}

	if err := syncer.SyncOfficialSources(context.Background()); err != nil {
		t.Fatalf("SyncOfficialSources() error = %v", err)
	}
	if got, want := len(store.snapshots), 6; got != want {
		t.Fatalf("snapshots = %d, want %d", got, want)
	}
	if got := string(store.snapshots[0].RawPayload); !strings.Contains(got, "endpoint") {
		t.Errorf("first snapshot = %s, want original JSON", got)
	}
	if got, want := store.finished, 6; got != want {
		t.Errorf("finished = %d, want %d", got, want)
	}
}

func TestSourceSnapshotPayloadHashesBinaryContentOnly(t *testing.T) {
	payload := sourceSnapshotPayload([]byte("%PDF-1.7"), "application/pdf")
	if got, want := string(payload), `{"bytes":8,"contentType":"application/pdf","stored":"hash_only"}`; got != want {
		t.Errorf("payload = %s, want %s", got, want)
	}
}
