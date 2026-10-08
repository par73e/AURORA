package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aurora/backend/internal/observatory"
)

type stubLightProvider struct {
	value observatory.LightPollution
	err   error
}

func (stub stubLightProvider) Light(context.Context, float64, float64) (observatory.LightPollution, error) {
	return stub.value, stub.err
}

func TestLightPollutionHandlerReturnsProvenance(t *testing.T) {
	provider := stubLightProvider{value: observatory.LightPollution{
		Bortle: 7, SQM: 17.3, Radiance: 96, RadianceUnit: "nW/cm²/sr", DataYear: 2025,
		ResolutionMeters: 500, Estimated: true, Source: "NASA Black Marble test",
	}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/astronomy/light-pollution?latitude=31.23&longitude=121.47", nil)
	response := httptest.NewRecorder()
	lightPollutionHandler(provider).ServeHTTP(response, request)

	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"dataYear":2025`) || !strings.Contains(response.Body.String(), `"estimated":true`) {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestLightPollutionHandlerReportsNoCoverage(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/astronomy/light-pollution?latitude=0&longitude=0", nil)
	response := httptest.NewRecorder()
	lightPollutionHandler(stubLightProvider{err: observatory.ErrNoCoverage}).ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestLightPollutionHandlerReportsMissingConfiguration(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/astronomy/light-pollution?latitude=31&longitude=121", nil)
	response := httptest.NewRecorder()
	lightPollutionHandler(stubLightProvider{err: observatory.ErrNotConfigured}).ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}
