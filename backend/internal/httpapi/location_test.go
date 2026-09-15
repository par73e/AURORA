package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	observerlocation "aurora/backend/internal/location"
)

type fakeReverseGeocoder struct {
	place     observerlocation.Place
	places    []observerlocation.Candidate
	err       error
	latitude  float64
	longitude float64
}

func (fake *fakeReverseGeocoder) Reverse(_ context.Context, latitude, longitude float64) (observerlocation.Place, error) {
	fake.latitude = latitude
	fake.longitude = longitude
	return fake.place, fake.err
}

func (fake *fakeReverseGeocoder) Search(_ context.Context, query string) ([]observerlocation.Candidate, error) {
	if query == "失败地点" {
		return nil, fake.err
	}
	return fake.places, fake.err
}

func TestReverseLocationHandler(t *testing.T) {
	geocoder := &fakeReverseGeocoder{place: observerlocation.Place{Label: "上海市 · 黄浦区", City: "上海市", District: "黄浦区"}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/location/reverse?latitude=31.2304&longitude=121.4737", nil)
	response := httptest.NewRecorder()

	reverseLocationHandler(geocoder).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if geocoder.latitude != 31.2304 || geocoder.longitude != 121.4737 {
		t.Fatalf("coordinates = %f,%f", geocoder.latitude, geocoder.longitude)
	}
	if !strings.Contains(response.Body.String(), `"label":"上海市 · 黄浦区"`) {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestReverseLocationHandlerRejectsInvalidCoordinate(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/location/reverse?latitude=91&longitude=121", nil)
	response := httptest.NewRecorder()
	reverseLocationHandler(&fakeReverseGeocoder{}).ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestReverseLocationHandlerReportsMissingConfiguration(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/location/reverse?latitude=31&longitude=121", nil)
	response := httptest.NewRecorder()
	reverseLocationHandler(&fakeReverseGeocoder{err: observerlocation.ErrNotConfigured}).ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestReverseLocationHandlerHidesUpstreamError(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/location/reverse?latitude=31&longitude=121", nil)
	response := httptest.NewRecorder()
	reverseLocationHandler(&fakeReverseGeocoder{err: errors.New("third-party detail")}).ServeHTTP(response, request)
	if response.Code != http.StatusBadGateway || strings.Contains(response.Body.String(), "third-party detail") {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestSearchLocationHandlerReturnsCandidates(t *testing.T) {
	geocoder := &fakeReverseGeocoder{places: []observerlocation.Candidate{{Label: "上海市崇明区陈家镇", Latitude: 31.50, Longitude: 121.81}}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/location/search?q=上海市崇明区陈家镇", nil)
	response := httptest.NewRecorder()
	searchLocationHandler(geocoder).ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"label":"上海市崇明区陈家镇"`) {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestSearchLocationHandlerRejectsShortQuery(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/location/search?q=沪", nil)
	response := httptest.NewRecorder()
	searchLocationHandler(&fakeReverseGeocoder{}).ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", response.Code)
	}
}
