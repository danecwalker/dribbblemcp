// Package browser manages a shared headless Chrome session via chromedp.
package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

// Session holds a reusable Chrome allocator + browser context.
type Session struct {
	mu          sync.Mutex
	allocCancel context.CancelFunc
	browserCtx  context.Context
	cancel      context.CancelFunc
}

var (
	shared   *Session
	sharedMu sync.Mutex
)

// Shared returns a process-wide browser session, creating it on first use.
func Shared() (*Session, error) {
	sharedMu.Lock()
	defer sharedMu.Unlock()
	if shared != nil {
		return shared, nil
	}
	s, err := New()
	if err != nil {
		return nil, err
	}
	shared = s
	return shared, nil
}

// New launches headless Chrome (system browser preferred).
func New() (*Session, error) {
	headless := os.Getenv("DRIBBBLE_MCP_HEADED") != "1"

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headless),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"),
		chromedp.WindowSize(1440, 1100),
	)

	if path := findChrome(); path != "" {
		opts = append(opts, chromedp.ExecPath(path))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	browserCtx, cancel := chromedp.NewContext(allocCtx)

	// Warm up: ensure Chrome actually starts.
	if err := chromedp.Run(browserCtx); err != nil {
		cancel()
		allocCancel()
		return nil, fmt.Errorf("start chrome: %w", err)
	}

	return &Session{
		allocCancel: allocCancel,
		browserCtx:  browserCtx,
		cancel:      cancel,
	}, nil
}

// WithPage runs fn inside a fresh tab derived from the shared browser.
// Concurrent calls are serialized to avoid hammering Dribbble.
func (s *Session) WithPage(fn func(ctx context.Context) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tabCtx, tabCancel := chromedp.NewContext(s.browserCtx)
	defer tabCancel()

	ctx, cancel := context.WithTimeout(tabCtx, 60*time.Second)
	defer cancel()

	return fn(ctx)
}

// Close shuts down the browser. Safe to call multiple times.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	if s.allocCancel != nil {
		s.allocCancel()
		s.allocCancel = nil
	}
	s.browserCtx = nil
	return nil
}

// CloseShared closes the process-wide session if any.
func CloseShared() {
	sharedMu.Lock()
	defer sharedMu.Unlock()
	if shared != nil {
		_ = shared.Close()
		shared = nil
	}
}

// Navigate loads url, waits for DOM + shot grid (best effort).
func Navigate(ctx context.Context, url string) error {
	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(2500*time.Millisecond),
	); err != nil {
		return fmt.Errorf("goto %s: %w", url, err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	_ = chromedp.Run(waitCtx,
		chromedp.WaitVisible(`li.shot-thumbnail, img[src*="cdn.dribbble.com/userupload"]`, chromedp.ByQuery),
	)
	_ = chromedp.Run(ctx, chromedp.Sleep(1000*time.Millisecond))
	return nil
}

// Evaluate runs a JS expression and decodes the result into out.
func Evaluate(ctx context.Context, js string, out any) error {
	return chromedp.Run(ctx, chromedp.Evaluate(js, out))
}

// EvaluateFunc runs a JS function (string form: `(arg) => { ... }`) with one JSON argument.
func EvaluateFunc(ctx context.Context, jsFunc string, arg any, out any) error {
	payload, err := json.Marshal(arg)
	if err != nil {
		return err
	}
	expr := fmt.Sprintf("(%s)(%s)", jsFunc, string(payload))
	return chromedp.Run(ctx, chromedp.Evaluate(expr, out))
}

func findChrome() string {
	if p := os.Getenv("CHROME_PATH"); p != "" {
		return p
	}
	candidates := []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
		"/usr/bin/google-chrome",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	for _, name := range []string{"google-chrome", "chromium", "chromium-browser", "chrome"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}
