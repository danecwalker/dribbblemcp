//go:build integration

package dribbble_test

import (
	"context"
	"testing"
	"time"

	"github.com/danecwalker/dribbblemcp/internal/browser"
	"github.com/danecwalker/dribbblemcp/internal/dribbble"
	"github.com/danecwalker/dribbblemcp/internal/images"
)

func TestSearchShotsLive(t *testing.T) {
	session, err := browser.New()
	if err != nil {
		t.Fatalf("browser: %v", err)
	}
	defer session.Close()

	c := dribbble.NewClient(session)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	_ = ctx

	result, err := c.SearchShots("fintech dashboard", 4)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Count == 0 {
		t.Fatal("expected at least one shot")
	}
	t.Logf("found %d shots; first=%s", result.Count, result.Shots[0].Title)

	// Image download should work from CDN without browser.
	imgURL := images.UpgradeResolution(result.Shots[0].ImageURL, "thumb")
	fetched, err := images.Fetch(context.Background(), imgURL)
	if err != nil {
		t.Fatalf("fetch image: %v", err)
	}
	if fetched.Bytes < 1000 {
		t.Fatalf("image too small: %d bytes", fetched.Bytes)
	}
	t.Logf("image %s bytes=%d mime=%s", imgURL, fetched.Bytes, fetched.MIMEType)

	detail, err := c.GetShot(result.Shots[0].URL)
	if err != nil {
		t.Fatalf("get shot: %v", err)
	}
	if detail.Title == "" {
		t.Fatal("expected title on detail")
	}
	t.Logf("detail title=%q designer=%q og=%s", detail.Title, detail.Designer, detail.OGImage)
}

func TestSearchByTagLive(t *testing.T) {
	session, err := browser.New()
	if err != nil {
		t.Fatalf("browser: %v", err)
	}
	defer session.Close()

	c := dribbble.NewClient(session)
	result, err := c.SearchByTag("dashboard", 3)
	if err != nil {
		t.Fatalf("tag search: %v", err)
	}
	if result.Count == 0 {
		t.Fatal("expected tag results")
	}
	t.Logf("tag hits=%d first=%s", result.Count, result.Shots[0].URL)
}
