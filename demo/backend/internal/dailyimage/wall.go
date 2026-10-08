package dailyimage

import (
	"context"
	"errors"
	"net/url"
	"path"
	"strings"
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
	Recent      []ImageWindow `json:"recent"`
	Collection  []ImageWindow `json:"collection"`
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
	// Five APOD issues form a real recent timeline. The gallery entries are a
	// separate continuing collection: their original release dates remain
	// visible and are never misrepresented as a daily feed.
	recent := service.apodWindows(ctx, at, 5)
	collection := service.libraryWindows(ctx, at, 5)
	collection = append(collection, CuratedWindows()...)
	return ImageWall{Recent: recent, Collection: collection, GeneratedAt: at.UTC().Format(time.RFC3339)}, nil
}

func (service *WallService) apodWindows(ctx context.Context, at time.Time, count int) []ImageWindow {
	if provider, ok := service.apod.(RecentProvider); ok {
		images, err := provider.Recent(ctx, at, count)
		if err == nil {
			windows := make([]ImageWindow, 0, len(images))
			for _, image := range images {
				if image.MediaType == "image" {
					windows = append(windows, imageWindow(image))
				}
			}
			if len(windows) > 0 {
				return windows
			}
		}
	}
	// APOD can lag the local calendar by a day or two. Request a small lookback
	// buffer, then keep the newest actual publications instead of rendering
	// empty future slots as if they were daily content.
	candidates := service.parallelWindows(ctx, at, count+5, service.apodWindow)
	windows := make([]ImageWindow, 0, count)
	for _, candidate := range candidates {
		if candidate.Status == "ready" && candidate.MediaType == "image" {
			windows = append(windows, candidate)
			if len(windows) == count {
				return windows
			}
		}
	}
	if len(windows) > 0 {
		return windows
	}
	return candidates[:1]
}

func (service *WallService) libraryWindows(ctx context.Context, at time.Time, count int) []ImageWindow {
	return service.parallelWindows(ctx, at, count, service.libraryWindow)
}

func (service *WallService) parallelWindows(ctx context.Context, at time.Time, count int, load func(context.Context, time.Time) ImageWindow) []ImageWindow {
	windows := make([]ImageWindow, count)
	type result struct {
		index  int
		window ImageWindow
	}
	results := make(chan result, count)
	for index := range windows {
		index := index
		date := at.UTC().AddDate(0, 0, -index)
		go func() { results <- result{index: index, window: load(ctx, date)} }()
	}
	for range windows {
		result := <-results
		result.window.ID = result.window.SourceID + "-" + result.window.PublishedAt
		if result.window.PublishedAt == "" {
			result.window.ID = result.window.SourceID + "-" + at.UTC().AddDate(0, 0, -result.index).Format(time.DateOnly)
		}
		windows[result.index] = result.window
	}
	return windows
}

func (service *WallService) apodWindow(ctx context.Context, at time.Time) ImageWindow {
	if service.apod == nil {
		return unavailableWindow("apod", "apod", "NASA Astronomy Picture of the Day", "https://apod.nasa.gov/apod/astropix.html", "每日图像服务尚未配置")
	}
	image, err := service.apod.Daily(ctx, at)
	if err != nil {
		return unavailableWindow("apod", "apod", "NASA Astronomy Picture of the Day", "https://apod.nasa.gov/apod/astropix.html", "今日图像暂不可用，请稍后重试")
	}
	return imageWindow(image)
}

func imageWindow(image Image) ImageWindow {
	credit := "NASA"
	if image.Copyright != "" {
		credit = image.Copyright
	}
	thumbnail := image.URL
	if image.MediaType == "video" {
		thumbnail = ""
		if imageURLLooksLikeImage(image.ThumbnailURL) {
			thumbnail = image.ThumbnailURL
		}
	}
	return ImageWindow{
		ID: "apod-" + image.Date, SourceID: "apod", SourceName: image.SourceName, Title: image.Title,
		PublishedAt: image.Date, ImageURL: image.URL, ThumbnailURL: thumbnail, MediaType: image.MediaType,
		Credit: credit, LicenseNote: "版权信息见原始链接", SourceURL: image.SourceURL,
		HDURL: firstNonEmpty(image.HDURL, image.URL), SelectionMode: "daily", Summary: image.Explanation, Status: "ready",
	}
}

func imageURLLooksLikeImage(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	switch strings.ToLower(path.Ext(parsed.Path)) {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".avif":
		return true
	default:
		return false
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
	return ImageWindow{ID: id, SourceID: sourceID, SourceName: sourceName, Title: title, PublishedAt: publishedAt, ImageURL: imageURL, ThumbnailURL: imageURL, MediaType: "image", Credit: credit, LicenseNote: licenseNote, SourceURL: sourceURL, HDURL: strings.Replace(imageURL, "/screen/", "/large/", 1), SelectionMode: "curated", Summary: summary, Status: "ready"}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
