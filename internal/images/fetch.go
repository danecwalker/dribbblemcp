// Package images downloads CDN images and encodes them for MCP responses.
package images

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const maxBytes = 4 << 20 // 4 MiB per image — keeps MCP payloads manageable

var client = &http.Client{
	Timeout: 30 * time.Second,
}

// Fetched is a downloaded image ready for MCP ImageContent.
type Fetched struct {
	URL      string
	MIMEType string
	Base64   string
	Bytes    int
}

// UpgradeResolution rewrites Dribbble CDN resize params for a larger preview.
// size is one of: "thumb" (400x300), "medium" (800x600), "large" (1600x1200).
func UpgradeResolution(rawURL, size string) string {
	if rawURL == "" {
		return rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := u.Query()
	// Drop size-limiting params, then set a controlled resize.
	q.Del("resize")
	q.Del("format")
	q.Del("crop")
	q.Del("vertical")
	switch size {
	case "large":
		q.Set("resize", "1600x1200")
	case "medium":
		q.Set("resize", "800x600")
	default:
		q.Set("resize", "400x300")
	}
	// Prefer webp for smaller payloads when possible.
	q.Set("format", "webp")
	u.RawQuery = q.Encode()
	return u.String()
}

// Fetch downloads an image and returns base64-encoded data.
func Fetch(ctx context.Context, imageURL string) (*Fetched, error) {
	if imageURL == "" {
		return nil, fmt.Errorf("empty image url")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; dribbblemcp/1.0)")
	req.Header.Set("Accept", "image/webp,image/avif,image/*,*/*;q=0.8")
	req.Header.Set("Referer", "https://dribbble.com/")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download image: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download image: status %d", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}
	if len(data) > maxBytes {
		return nil, fmt.Errorf("image exceeds %d byte limit", maxBytes)
	}

	mime := resp.Header.Get("Content-Type")
	if mime == "" || strings.HasPrefix(mime, "application/octet-stream") {
		mime = guessMIME(imageURL, data)
	}
	// Strip parameters like "image/webp; charset=utf-8"
	if i := strings.IndexByte(mime, ';'); i >= 0 {
		mime = strings.TrimSpace(mime[:i])
	}

	return &Fetched{
		URL:      imageURL,
		MIMEType: mime,
		Base64:   base64.StdEncoding.EncodeToString(data),
		Bytes:    len(data),
	}, nil
}

func guessMIME(u string, data []byte) string {
	if len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff {
		return "image/jpeg"
	}
	if len(data) >= 8 && data[0] == 0x89 && data[1] == 0x50 {
		return "image/png"
	}
	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}
	if len(data) >= 6 && (string(data[0:6]) == "GIF87a" || string(data[0:6]) == "GIF89a") {
		return "image/gif"
	}
	ext := strings.ToLower(path.Ext(strings.Split(u, "?")[0]))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return "image/png"
	}
}
