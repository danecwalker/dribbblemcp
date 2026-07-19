package dribbble

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/danecwalker/dribbblemcp/internal/browser"
)

var (
	shotIDRe   = regexp.MustCompile(`/shots/(\d+)`)
	viewPrefix = regexp.MustCompile(`(?i)^view\s+`)
	digitsOnly = regexp.MustCompile(`^\d+$`)
)

// Client scrapes public Dribbble pages via a shared browser session.
type Client struct {
	session *browser.Session
}

// NewClient wraps the given browser session.
func NewClient(session *browser.Session) *Client {
	return &Client{session: session}
}

// SearchShots finds shots matching a free-text query on Dribbble search.
func (c *Client) SearchShots(query string, limit int) (*SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query is required")
	}
	limit = clampLimit(limit)

	searchURL := "https://dribbble.com/search/" + url.PathEscape(strings.TrimSpace(query))
	var shots []Shot
	err := c.session.WithPage(func(ctx context.Context) error {
		if err := browser.Navigate(ctx, searchURL); err != nil {
			return err
		}
		var err error
		shots, err = extractShots(ctx, limit)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &SearchResult{
		Query:  query,
		Source: searchURL,
		Count:  len(shots),
		Shots:  shots,
	}, nil
}

// SearchByTag finds shots for a Dribbble tag (e.g. "dashboard", "mobile-app").
func (c *Client) SearchByTag(tag string, limit int) (*SearchResult, error) {
	tag = normalizeTag(tag)
	if tag == "" {
		return nil, fmt.Errorf("tag is required")
	}
	limit = clampLimit(limit)

	tagURL := "https://dribbble.com/tags/" + url.PathEscape(tag)
	var shots []Shot
	err := c.session.WithPage(func(ctx context.Context) error {
		if err := browser.Navigate(ctx, tagURL); err != nil {
			return err
		}
		var err error
		shots, err = extractShots(ctx, limit)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &SearchResult{
		Query:  "tag:" + tag,
		Source: tagURL,
		Count:  len(shots),
		Shots:  shots,
	}, nil
}

// GetShot loads a single shot page and returns full metadata + high-res images.
func (c *Client) GetShot(shotURL string) (*ShotDetail, error) {
	shotURL = strings.TrimSpace(shotURL)
	if shotURL == "" {
		return nil, fmt.Errorf("url is required")
	}
	if !strings.Contains(shotURL, "dribbble.com/shots/") {
		if digitsOnly.MatchString(shotURL) {
			shotURL = "https://dribbble.com/shots/" + shotURL
		} else {
			return nil, fmt.Errorf("url must be a dribbble.com/shots/… link or numeric shot id")
		}
	}
	if !strings.HasPrefix(shotURL, "http") {
		shotURL = "https://" + strings.TrimPrefix(shotURL, "//")
	}
	if u, err := url.Parse(shotURL); err == nil {
		u.RawQuery = ""
		u.Fragment = ""
		shotURL = u.String()
	}

	var detail ShotDetail
	err := c.session.WithPage(func(ctx context.Context) error {
		if err := browser.Navigate(ctx, shotURL); err != nil {
			return err
		}
		var raw map[string]any
		if err := browser.Evaluate(ctx, getShotDetailJS, &raw); err != nil {
			return fmt.Errorf("extract shot detail: %w", err)
		}
		detail.URL = shotURL
		detail.ID = extractShotID(shotURL)
		detail.Title = cleanTitle(asString(raw["title"]))
		detail.Description = asString(raw["desc"])
		detail.Designer = asString(raw["designer"])
		detail.OGImage = asString(raw["ogImage"])
		detail.ImageURL = detail.OGImage
		if imgs, ok := raw["images"].([]any); ok {
			for _, v := range imgs {
				if s := asString(v); s != "" {
					detail.Images = append(detail.Images, s)
				}
			}
		}
		if detail.ImageURL == "" && len(detail.Images) > 0 {
			detail.ImageURL = detail.Images[0]
		}
		if tags, ok := raw["tags"].([]any); ok {
			for _, v := range tags {
				if s := asString(v); s != "" {
					detail.Tags = append(detail.Tags, s)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if detail.Title == "" && detail.ImageURL == "" {
		return nil, fmt.Errorf("could not extract shot from %s (page may be blocked or private)", shotURL)
	}
	return &detail, nil
}

const getShotDetailJS = `(() => {
	const ogImage = document.querySelector('meta[property="og:image"]')?.content || '';
	const title = document.querySelector('meta[property="og:title"]')?.content
		|| document.querySelector('h1')?.textContent?.trim()
		|| document.title || '';
	const desc = document.querySelector('meta[property="og:description"]')?.content
		|| document.querySelector('meta[name="description"]')?.content || '';
	const images = [...document.querySelectorAll('img')]
		.map(i => i.currentSrc || i.src || '')
		.filter(s => s.includes('cdn.dribbble.com') && (s.includes('userupload') || s.includes('screenshots')))
		.filter((s, i, arr) => arr.indexOf(s) === i)
		.slice(0, 12);
	let designer = '';
	const byline = document.querySelector('a[rel="contact"], .shot-user-name a, .shot-byline a, a[data-user-login]');
	if (byline) designer = (byline.textContent || '').trim();
	if (!designer && desc) {
		const m = desc.match(/designed by\s+(.+?)\s+for\s+/i) || desc.match(/designed by\s+(.+?)\s*\./i);
		if (m) designer = m[1].trim();
	}
	const tags = [...document.querySelectorAll('a[href*="/tags/"], a[href*="/search/"]')]
		.map(a => (a.textContent || '').trim().toLowerCase())
		.filter(t => t && t.length < 40)
		.filter((t, i, arr) => arr.indexOf(t) === i)
		.slice(0, 16);
	return { ogImage, title, desc, images, designer, tags };
})()`

const extractShotsJS = `(limit) => {
	const results = [];
	const seen = new Set();
	const shotHref = /\/shots\/\d+/;
	const shotID = /\/shots\/(\d+)/;

	const cleanShotTitle = (raw) => {
		let title = (raw || '').replace(/^view\s+/i, '').trim();
		// alt often appends lowercase tags after a Title Case name
		const words = title.split(/\s+/);
		const kept = [];
		for (const w of words) {
			const isTagLike = w === w.toLowerCase() && /^[a-z0-9+#.-]+$/.test(w) && w.length < 28;
			if (kept.length >= 2 && isTagLike) break;
			kept.push(w);
			if (kept.length >= 12) break;
		}
		return kept.join(' ').slice(0, 160);
	};

	const cards = document.querySelectorAll('li.shot-thumbnail, li[id^="screenshot-"], [data-thumbnail-id]');
	for (const card of cards) {
		if (results.length >= limit) break;
		const id = card.getAttribute('data-thumbnail-id')
			|| (card.id || '').replace(/^screenshot-/, '')
			|| '';
		const link = card.querySelector('a[href*="/shots/"]');
		let href = link?.href || '';
		if (!href || !shotHref.test(href)) continue;
		href = href.split('?')[0];
		if (seen.has(href)) continue;
		seen.add(href);

		const img = card.querySelector('img');
		const src = img?.currentSrc || img?.src || img?.getAttribute('data-src') || '';
		if (!src || !src.includes('cdn.dribbble.com')) continue;
		if (src.includes('/avatars/') || src.includes('/assets/')) continue;

		const title = cleanShotTitle(img?.alt || link?.getAttribute('title') || link?.textContent || '');

		let designer = '';
		const designerA = card.querySelector('a[href^="https://dribbble.com/"]:not([href*="/shots/"]):not([href*="/tags/"])');
		if (designerA) designer = (designerA.textContent || '').trim();

		const tags = [];
		const alt = img?.alt || '';
		for (const t of alt.toLowerCase().split(/\s+/)) {
			if (t.length > 2 && t.length < 30 && !title.toLowerCase().includes(t)) {
				tags.push(t);
			}
		}

		results.push({
			id: id || (href.match(shotID) || [])[1] || '',
			title: title.slice(0, 160),
			url: href,
			image_url: src,
			designer,
			tags: tags.slice(0, 12),
		});
	}

	if (results.length === 0) {
		for (const img of document.querySelectorAll('img')) {
			if (results.length >= limit) break;
			const src = img.currentSrc || img.src || '';
			if (!src.includes('cdn.dribbble.com') || src.includes('/avatars/') || src.includes('/assets/')) continue;
			let a = img.closest('a[href*="/shots/"]');
			if (!a) {
				let el = img.parentElement;
				for (let i = 0; i < 6 && el; i++) {
					const found = el.querySelector?.('a[href*="/shots/"]');
					if (found) { a = found; break; }
					el = el.parentElement;
				}
			}
			const href = (a?.href || '').split('?')[0];
			if (!href || !shotHref.test(href) || seen.has(href)) continue;
			seen.add(href);
			const title = cleanShotTitle(img.alt || a?.getAttribute('title') || '');
			results.push({
				id: (href.match(shotID) || [])[1] || '',
				title: title.slice(0, 160),
				url: href,
				image_url: src,
				designer: '',
				tags: [],
			});
		}
	}
	return results;
}`

func extractShots(ctx context.Context, limit int) ([]Shot, error) {
	var list []any
	if err := browser.EvaluateFunc(ctx, extractShotsJS, limit, &list); err != nil {
		return nil, fmt.Errorf("extract shots: %w", err)
	}

	out := make([]Shot, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		s := Shot{
			ID:       asString(m["id"]),
			Title:    cleanTitle(asString(m["title"])),
			URL:      asString(m["url"]),
			ImageURL: asString(m["image_url"]),
			Designer: asString(m["designer"]),
		}
		if tags, ok := m["tags"].([]any); ok {
			for _, t := range tags {
				if ts := asString(t); ts != "" {
					s.Tags = append(s.Tags, ts)
				}
			}
		}
		if s.URL == "" || s.ImageURL == "" {
			continue
		}
		if s.ID == "" {
			s.ID = extractShotID(s.URL)
		}
		out = append(out, s)
	}
	return out, nil
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return 8
	}
	if limit > 20 {
		return 20
	}
	return limit
}

func cleanTitle(t string) string {
	t = strings.TrimSpace(viewPrefix.ReplaceAllString(t, ""))
	if parts := strings.Split(t, "  "); len(parts) > 1 {
		t = parts[0]
	}
	return strings.TrimSpace(t)
}

func extractShotID(u string) string {
	m := shotIDRe.FindStringSubmatch(u)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

func normalizeTag(tag string) string {
	tag = strings.TrimSpace(strings.ToLower(tag))
	tag = strings.TrimPrefix(tag, "#")
	tag = strings.ReplaceAll(tag, " ", "-")
	tag = strings.ReplaceAll(tag, "_", "-")
	return tag
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		if v == nil {
			return ""
		}
		return fmt.Sprint(v)
	}
}
