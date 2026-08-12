package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aurora/backend/internal/dailyimage"
)

type imageWallStub struct {
	wall dailyimage.ImageWall
	err  error
}

func (stub imageWallStub) Wall(context.Context, time.Time) (dailyimage.ImageWall, error) {
	return stub.wall, stub.err
}

func TestImageWallHandlerReturnsAllWindows(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/astronomy/image-wall", nil)
	imageWallHandler(imageWallStub{wall: dailyimage.ImageWall{Recent: []dailyimage.ImageWindow{{ID: "apod"}}}}, time.Now).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d", recorder.Code)
	}
}
