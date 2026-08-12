package dailyimage

import (
	"context"
	"errors"
	"time"
)

// ImageWindow is one independently loadable image in AURORA's image wall.
// It contains attribution and source links alongside the thumbnail so the UI
// never needs to infer ownership from an institution name or logo.
type ImageWindow struct {
	ID            string `json:"id"`
	SourceID      string `json:"sourceId"`
	SourceName    string `json:"sourceName"`
	Title         string `json:"title"`
	PublishedAt   string `json:"publishedAt,omitempty"`
	ImageURL      string `json:"imageUrl,omitempty"`
	ThumbnailURL  string `json:"thumbnailUrl,omitempty"`
	MediaType     string `json:"mediaType"`
	Credit        string `json:"credit"`
	LicenseNote   string `json:"licenseNote,omitempty"`
	SourceURL     string `json:"sourceUrl"`
	HDURL         string `json:"hdUrl,omitempty"`
	SelectionMode string `json:"selectionMode"`
	Summary       string `json:"summary,omitempty"`
	Status        string `json:"status"`
	Error         string `json:"error,omitempty"`
	IsFallback    bool   `json:"isFallback,omitempty"`
}

type ImageWall struct {
	Windows     []ImageWindow `json:"windows"`
	GeneratedAt string        `json:"generatedAt"`
}

type WallProvider interface {
	Wall(context.Context, time.Time) (ImageWall, error)
}

type LibraryProvider interface {
	Pick(context.Context, time.Time) (ImageWindow, error)
}

type WallService struct {
	apod    Provider
	library LibraryProvider
}

func NewWallService(apod Provider, library LibraryProvider) *WallService {
	return &WallService{apod: apod, library: library}
}

func (service *WallService) Wall(ctx context.Context, at time.Time) (ImageWall, error) {
	if service == nil {
		return ImageWall{}, errors.New("image wall service is not configured")
	}
	// The two remote sources are deliberately concurrent. A slow APOD response
	// must not serially delay the public NASA archive search (or vice versa).
	apodResult := make(chan ImageWindow, 1)
	libraryResult := make(chan ImageWindow, 1)
	go func() { apodResult <- service.apodWindow(ctx, at) }()
	go func() { libraryResult <- service.libraryWindow(ctx, at) }()
	windows := make([]ImageWindow, 0, 5)
	windows = append(windows, <-apodResult)
	windows = append(windows, <-libraryResult)
	windows = append(windows, CuratedWindows(at)...)
	return ImageWall{Windows: windows, GeneratedAt: at.UTC().Format(time.RFC3339)}, nil
}

func (service *WallService) apodWindow(ctx context.Context, at time.Time) ImageWindow {
	if service.apod == nil {
		return unavailableWindow("apod", "apod", "NASA Astronomy Picture of the Day", "https://apod.nasa.gov/apod/astropix.html", "每日图像服务尚未配置")
	}
	image, err := service.apod.Daily(ctx, at)
	if err != nil {
		return unavailableWindow("apod", "apod", "NASA Astronomy Picture of the Day", "https://apod.nasa.gov/apod/astropix.html", "今日图像暂不可用，请稍后重试")
	}
	credit := "NASA"
	if image.Copyright != "" {
		credit = image.Copyright
	}
	thumbnail := image.URL
	if image.MediaType == "video" && image.ThumbnailURL != "" {
		thumbnail = image.ThumbnailURL
	}
	return ImageWindow{
		ID: "apod", SourceID: "apod", SourceName: image.SourceName, Title: image.Title,
		PublishedAt: image.Date, ImageURL: image.URL, ThumbnailURL: thumbnail, MediaType: image.MediaType,
		Credit: credit, LicenseNote: "版权以 NASA APOD 当期字段为准", SourceURL: image.SourceURL,
		HDURL: image.HDURL, SelectionMode: "daily", Summary: image.Explanation, Status: "ready",
	}
}

func (service *WallService) libraryWindow(ctx context.Context, at time.Time) ImageWindow {
	if service.library == nil {
		return unavailableWindow("nasa-library", "nasa-library", "NASA Image and Video Library", "https://images.nasa.gov/", "NASA 图库服务尚未配置")
	}
	window, err := service.library.Pick(ctx, at)
	if err != nil {
		return unavailableWindow("nasa-library", "nasa-library", "NASA Image and Video Library", "https://images.nasa.gov/", "本次档案图读取失败，请稍后重试")
	}
	return window
}

func unavailableWindow(id, sourceID, sourceName, sourceURL, message string) ImageWindow {
	return ImageWindow{ID: id, SourceID: sourceID, SourceName: sourceName, Title: "此窗口暂不可用", MediaType: "image", Credit: sourceName, SourceURL: sourceURL, SelectionMode: "rotating", Status: "error", Error: message}
}

func sourceWindow(id string, sourceID string, sourceName string, title string, publishedAt string, imageURL string, credit string, licenseNote string, sourceURL string, summary string) ImageWindow {
	return ImageWindow{ID: id, SourceID: sourceID, SourceName: sourceName, Title: title, PublishedAt: publishedAt, ImageURL: imageURL, ThumbnailURL: imageURL, MediaType: "image", Credit: credit, LicenseNote: licenseNote, SourceURL: sourceURL, SelectionMode: "curated", Summary: summary, Status: "ready"}
}
