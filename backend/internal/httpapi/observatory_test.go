package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"aurora/backend/internal/observatory"
)

type conditionsProviderStub struct{ calls int }

func (stub *conditionsProviderStub) Conditions(_ context.Context, _, _ float64) (observatory.Conditions, error) {
	stub.calls++
	return observatory.Conditions{Timezone: "UTC"}, nil
}

func TestObservingConditionsRejectsInvalidInputBeforeWeatherLookup(t *testing.T) {
	for _, query := range []string{
		"latitude=91&longitude=0",
		"latitude=0&longitude=181",
		"latitude=NaN&longitude=0",
		"latitude=Inf&longitude=0",
		"latitude=0&longitude=0&time=invalid",
		"latitude=0&longitude=0&time=-1",
	} {
		t.Run(query, func(t *testing.T) {
			provider := &conditionsProviderStub{}
			request := httptest.NewRequest(http.MethodGet, "/api/v1/astronomy/conditions?"+query, nil)
			response := httptest.NewRecorder()
			observingConditionsHandler(provider, nil)(response, request)
			if response.Code != http.StatusBadRequest || provider.calls != 0 {
				t.Fatalf("status=%d, provider calls=%d; want 400 and no weather lookup", response.Code, provider.calls)
			}
		})
	}
}

type degradedProviderStub struct{}

func (degradedProviderStub) Conditions(_ context.Context, _, _ float64) (observatory.Conditions, error) {
	return observatory.Conditions{Timezone: "UTC", Degraded: &observatory.DegradedNotice{Reason: "upstream_unavailable", Since: "2026-11-14T10:00:00Z"}}, nil
}

func TestDegradedConditionsCarryNoStore(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/astronomy/conditions?latitude=31.23&longitude=121.47", nil)
	response := httptest.NewRecorder()
	observingConditionsHandler(degradedProviderStub{}, nil)(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200 for degraded forecast", response.Code)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control=%q, want no-store so degraded snapshots never propagate", got)
	}
}

func TestFreshConditionsAreNotMarkedNoStore(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/astronomy/conditions?latitude=31.23&longitude=121.47", nil)
	response := httptest.NewRecorder()
	observingConditionsHandler(&conditionsProviderStub{}, nil)(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d", response.Code)
	}
	if got := response.Header().Get("Cache-Control"); got == "no-store" {
		t.Fatal("fresh conditions were marked no-store")
	}
}
