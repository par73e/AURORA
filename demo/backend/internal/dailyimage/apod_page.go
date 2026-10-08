package dailyimage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// Site chrome must never become a publication, including in persisted caches.
func invalidAPODContent(title, imageURL string) bool {
	title = strings.ToLower(strings.TrimSpace(title))
	u, err := url.Parse(imageURL)
	if err != nil {
		return true
	}
	path := strings.ToLower(u.Path)
	return title == "nasa science" || strings.Contains(path, "logo") || strings.Contains(path, "/themes/") || strings.Contains(path, "/icons/")
}

func (client *APODClient) pageImages(ctx context.Context, end time.Time, count int, exact bool) ([]Image, error) {
	if count < 1 {
		return []Image{}, nil
	}
	endpoint := client.pageURL
	seen := map[string]bool{}
	images := make([]Image, 0, count)
	lastDate := ""
	// Walk actual publications, never synthesize dates from the requested day.
	for i := 0; i < count+10 && endpoint != "" && !seen[endpoint]; i++ {
		seen[endpoint] = true
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("APOD page returned %d", resp.StatusCode)
		}
		doc, err := html.Parse(io.LimitReader(resp.Body, 4<<20))
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		image, previous, err := parseAPODPage(doc, resp.Request.URL)
		if err != nil {
			return nil, err
		}
		if lastDate != "" && image.Date >= lastDate {
			return nil, errors.New("APOD history does not move to an older publication")
		}
		lastDate = image.Date
		date, _ := time.Parse(time.DateOnly, image.Date)
		target := end.UTC().Format(time.DateOnly)
		if image.Date <= target {
			if exact && image.Date != target {
				return nil, errors.New("requested APOD date is not published")
			}
			if image.MediaType == "image" {
				images = append(images, image)
			}
			if exact && image.MediaType != "image" {
				return nil, errors.New("requested APOD is not an image")
			}
			if len(images) == count {
				return images, nil
			}
		}
		if date.Before(end.UTC().AddDate(0, 0, -10)) {
			break
		}
		endpoint = previous
	}
	if len(images) > 0 && !exact {
		return images, nil
	}
	return nil, errors.New("no valid APOD publications found")
}

func parseAPODPage(doc *html.Node, base *url.URL) (Image, string, error) {
	hero := findNode(doc, func(n *html.Node) bool { return hasClass(n, "hds-media-detail-hero") })
	if hero == nil {
		return Image{}, "", errors.New("APOD body is missing")
	}
	image := Image{SourceName: "NASA Astronomy Picture of the Day", SourceURL: base.String(), MediaType: "video"}
	title := findNode(hero, func(n *html.Node) bool { return n.Type == html.ElementNode && (n.Data == "h1" || n.Data == "h2") })
	image.Title = nodeText(title)
	if strings.HasPrefix(image.Title, "APOD:") {
		if _, short, ok := strings.Cut(image.Title, " – "); ok {
			image.Title = short
		}
	}
	media := findNode(hero, func(n *html.Node) bool { return hasClass(n, "media-detail-hero__media") })
	picture := findNode(media, func(n *html.Node) bool {
		if n.Type != html.ElementNode || n.Data != "img" {
			return false
		}
		u := resolveURL(base, attr(n, "src"))
		return imageURLLooksLikeImage(u) && !invalidAPODContent(image.Title, u)
	})
	if picture != nil {
		image.URL = resolveURL(base, attr(picture, "src"))
		image.HDURL = image.URL
		image.MediaType = "image"
		// The image links to its permanent publication, rather than a moving today page.
		for parent := picture.Parent; parent != nil && parent != media; parent = parent.Parent {
			if parent.Data == "a" {
				if link := publicationLink(base, attr(parent, "href")); link != "" {
					image.SourceURL = link
				}
				break
			}
		}
	}
	if picture == nil && findNode(media, func(n *html.Node) bool { return n.Data == "iframe" || n.Data == "video" }) == nil {
		return Image{}, "", errors.New("APOD body has no valid image or video")
	}
	description := findNode(hero, func(n *html.Node) bool { return hasClass(n, "media-detail-hero__description") })
	image.Explanation = strings.TrimSpace(strings.TrimPrefix(nodeText(description), "Explanation:"))
	walk(hero, func(n *html.Node) {
		if !hasClass(n, "media-detail-hero__meta-row") {
			return
		}
		label := nodeText(findNode(n, func(x *html.Node) bool { return x.Data == "th" }))
		value := nodeText(findNode(n, func(x *html.Node) bool { return x.Data == "td" }))
		switch strings.ToLower(strings.TrimSuffix(label, ":")) {
		case "date":
			for _, layout := range []string{"January 2, 2006", "Jan 2, 2006"} {
				if d, err := time.Parse(layout, value); err == nil {
					image.Date = d.Format(time.DateOnly)
					break
				}
			}
		case "credit & copyright", "credit", "image credit & copyright", "image credit":
			image.Copyright = value
		}
	})
	previous := ""
	nav := findNode(doc, func(n *html.Node) bool { return hasClass(n, "smd-next-prev__prev") })
	if nav != nil {
		previous = publicationLink(base, attr(nav, "href"))
	}
	if image.Date == "" || image.Title == "" || image.Explanation == "" || invalidAPODContent(image.Title, image.URL) {
		return Image{}, "", fmt.Errorf("APOD body lacks valid publication metadata at %s (date=%q, title=%q)", base.Path, image.Date, image.Title)
	}
	return image, previous, nil
}

func publicationLink(base *url.URL, value string) string {
	u, err := base.Parse(value)
	if err != nil || u.Scheme != base.Scheme || u.Host != base.Host || !strings.HasPrefix(u.Path, "/image-article/apod-") {
		return ""
	}
	return u.String()
}
func resolveURL(base *url.URL, value string) string {
	if value == "" {
		return ""
	}
	u, err := base.Parse(value)
	if err != nil {
		return ""
	}
	return u.String()
}
func attr(n *html.Node, key string) string {
	if n != nil {
		for _, a := range n.Attr {
			if a.Key == key {
				return a.Val
			}
		}
	}
	return ""
}
func hasClass(n *html.Node, class string) bool {
	for _, c := range strings.Fields(attr(n, "class")) {
		if c == class {
			return true
		}
	}
	return false
}
func walk(n *html.Node, visit func(*html.Node)) {
	if n == nil {
		return
	}
	visit(n)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, visit)
	}
}
func findNode(n *html.Node, predicate func(*html.Node) bool) *html.Node {
	if n == nil {
		return nil
	}
	if predicate(n) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findNode(c, predicate); result != nil {
			return result
		}
	}
	return nil
}
func nodeText(n *html.Node) string {
	var s strings.Builder
	walk(n, func(x *html.Node) {
		if x.Type == html.TextNode {
			s.WriteString(x.Data)
			s.WriteByte(' ')
		}
	})
	return strings.Join(strings.Fields(s.String()), " ")
}
