package syncer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetWithRetryRecoversFromTransientFailure(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			http.Error(w, "temporary upstream failure", http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	s := &Syncer{client: &http.Client{Timeout: time.Second}}
	body, err := s.getWithRetry(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("getWithRetry returned error: %v", err)
	}
	if got := string(body); got != "ok" {
		t.Fatalf("body = %q, want ok", got)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestGetWithRetryStopsForCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := &Syncer{client: &http.Client{Timeout: time.Second}}
	if _, err := s.getWithRetry(ctx, "https://example.invalid"); err == nil {
		t.Fatal("getWithRetry succeeded with canceled context")
	}
}
