package httpapi

import (
	"aurora/backend/internal/dailyimage"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type dailyImageStub struct {
	image dailyimage.Image
	err   error
	date  time.Time
}

func (stub *dailyImageStub) Daily(_ context.Context, date time.Time) (dailyimage.Image, error) {
	stub.date = date
	return stub.image, stub.err
}

func TestDailyImageHandlerReturnsProviderImage(t *testing.T) {
	stub := &dailyImageStub{image: dailyimage.Image{Date: "2026-08-12", Title: "Perseids"}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/astronomy/daily-image?date=2026-08-12", nil)
	dailyImageHandler(stub, func() time.Time { return time.Date(2026, 8, 12, 8, 0, 0, 0, time.UTC) }).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || stub.date.Format(time.DateOnly) != "2026-08-12" {
		t.Fatalf("status=%d date=%s", recorder.Code, stub.date)
	}
}

func TestDailyImageHandlerReportsMissingKey(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/astronomy/daily-image", nil)
	dailyImageHandler(&dailyImageStub{err: dailyimage.ErrNotConfigured}, time.Now).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", recorder.Code)
	}
}
