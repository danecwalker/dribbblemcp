// Package main implements an MCP server for Dribbble UI design inspiration.
//
// It uses a headless Chromium browser (Playwright) to search public Dribbble
// pages and return shot images the model can inspect.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/danecwalker/dribbblemcp/internal/browser"
	"github.com/danecwalker/dribbblemcp/internal/dribbble"
	"github.com/danecwalker/dribbblemcp/internal/images"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var (
	logger *log.Logger
	client *dribbble.Client
)

func main() {
	logger = log.New(os.Stderr, "[dribbblemcp] ", log.LstdFlags|log.Lmsgprefix)

	session, err := browser.Shared()
	if err != nil {
		logger.Fatalf("browser init: %v", err)
	}
	client = dribbble.NewClient(session)

	// Clean shutdown so Chromium doesn't leak.
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		browser.CloseShared()
		os.Exit(0)
	}()

	s := server.NewMCPServer(
		"dribbble",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	s.AddTool(mcp.NewTool("search_shots",
		mcp.WithDescription(`Search Dribbble for UI design shots matching a free-text query. Returns shot metadata plus inline preview images so you can visually inspect layouts, color, typography, and composition. Use specific UI language (e.g. "fintech dashboard dark mode", "mobile onboarding checklist", "SaaS pricing page"). Always cite each shot as a markdown link to its url when presenting results.`),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("What UI design to find. Be specific about product type, screen, and style. Good: \"crypto wallet mobile home\", \"B2B analytics dashboard light mode\". Avoid vague words alone like \"modern\" or \"clean\"."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Max shots to return (1–12). Default 6. Higher values use more context."),
		),
		mcp.WithBoolean("include_images",
			mcp.Description("If true (default), download and return preview images inline. Set false for metadata-only."),
		),
	), searchShotsHandler)

	s.AddTool(mcp.NewTool("get_shot",
		mcp.WithDescription(`Fetch a single Dribbble shot by URL or numeric ID. Returns title, description, designer, tags, and a higher-resolution image for close visual study. Use after search_shots when a result looks promising.`),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("Full dribbble.com/shots/… URL, or a numeric shot id (e.g. \"25214011\")."),
		),
		mcp.WithBoolean("include_images",
			mcp.Description("If true (default), download and return the high-res shot image(s) inline."),
		),
	), getShotHandler)

	s.AddTool(mcp.NewTool("search_by_tag",
		mcp.WithDescription(`Browse Dribbble shots for a specific tag (e.g. dashboard, mobile-app, landing-page, saas, fintech, ui, ux). Returns metadata plus preview images. Prefer search_shots for multi-word natural-language queries; use this for established single tags.`),
		mcp.WithString("tag",
			mcp.Required(),
			mcp.Description("Tag slug or name, e.g. \"dashboard\", \"mobile-app\", \"landing-page\", \"dark-mode\"."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Max shots to return (1–12). Default 6."),
		),
		mcp.WithBoolean("include_images",
			mcp.Description("If true (default), download and return preview images inline."),
		),
	), searchByTagHandler)

	logger.Println("starting stdio MCP server")
	if err := server.ServeStdio(s); err != nil {
		browser.CloseShared()
		logger.Fatalf("server error: %v", err)
	}
	browser.CloseShared()
}

func searchShotsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("query is required"), nil
	}
	limit := request.GetInt("limit", 6)
	if limit < 1 {
		limit = 1
	}
	if limit > 12 {
		limit = 12
	}
	includeImages := request.GetBool("include_images", true)

	result, err := client.SearchShots(query, limit)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("search failed", err), nil
	}
	if result.Count == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No shots found for %q. Try a more specific UI query (screen type + product domain + style).", query)), nil
	}
	return buildSearchResult(ctx, result, includeImages, "thumb")
}

func searchByTagHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag, err := request.RequireString("tag")
	if err != nil {
		return mcp.NewToolResultError("tag is required"), nil
	}
	limit := request.GetInt("limit", 6)
	if limit < 1 {
		limit = 1
	}
	if limit > 12 {
		limit = 12
	}
	includeImages := request.GetBool("include_images", true)

	result, err := client.SearchByTag(tag, limit)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("tag search failed", err), nil
	}
	if result.Count == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No shots found for tag %q.", tag)), nil
	}
	return buildSearchResult(ctx, result, includeImages, "thumb")
}

func getShotHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawURL, err := request.RequireString("url")
	if err != nil {
		return mcp.NewToolResultError("url is required"), nil
	}
	includeImages := request.GetBool("include_images", true)

	detail, err := client.GetShot(rawURL)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("get shot failed", err), nil
	}

	meta, _ := json.MarshalIndent(detail, "", "  ")
	contents := []mcp.Content{
		mcp.NewTextContent(fmt.Sprintf(
			"Shot: [%s](%s)\nDesigner: %s\n\n%s\n\nMetadata:\n%s\n\nExamine the image(s) carefully for layout structure, spacing, color palette, typography hierarchy, and component patterns. Credit the designer when referencing this work. Do not copy designs wholesale — extract principles.",
			detail.Title, detail.URL, nonEmpty(detail.Designer, "unknown"), detail.Description, string(meta),
		)),
	}

	if includeImages {
		// Prefer OG image (high-res), then fall back to other page images.
		urls := uniqueNonEmpty(append([]string{detail.OGImage, detail.ImageURL}, detail.Images...))
		maxImgs := 3
		for i, u := range urls {
			if i >= maxImgs {
				break
			}
			size := "large"
			if i > 0 {
				size = "medium"
			}
			fetched, ferr := images.Fetch(ctx, images.UpgradeResolution(u, size))
			if ferr != nil {
				logger.Printf("image fetch failed for %s: %v", u, ferr)
				continue
			}
			contents = append(contents, mcp.NewImageContent(fetched.Base64, fetched.MIMEType))
		}
	}

	return &mcp.CallToolResult{Content: contents}, nil
}

func buildSearchResult(ctx context.Context, result *dribbble.SearchResult, includeImages bool, size string) (*mcp.CallToolResult, error) {
	// Text summary first so the model always has structured references.
	type shotRef struct {
		Index    int      `json:"index"`
		ID       string   `json:"id"`
		Title    string   `json:"title"`
		URL      string   `json:"url"`
		ImageURL string   `json:"image_url"`
		Designer string   `json:"designer,omitempty"`
		Tags     []string `json:"tags,omitempty"`
	}
	refs := make([]shotRef, 0, len(result.Shots))
	for i, s := range result.Shots {
		refs = append(refs, shotRef{
			Index:    i + 1,
			ID:       s.ID,
			Title:    s.Title,
			URL:      s.URL,
			ImageURL: s.ImageURL,
			Designer: s.Designer,
			Tags:     s.Tags,
		})
	}
	payload := map[string]any{
		"query":  result.Query,
		"source": result.Source,
		"count":  result.Count,
		"shots":  refs,
		"note":   "Images follow this JSON when include_images=true, in the same order as shots[]. Examine each image visually — do not rely on titles alone. Always link to shot urls when presenting results. Use get_shot for a higher-res look at promising shots.",
	}
	metaJSON, _ := json.MarshalIndent(payload, "", "  ")

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d Dribbble shot(s) for %q.\n\n", result.Count, result.Query))
	for _, r := range refs {
		fmt.Fprintf(&b, "%d. [%s](%s)", r.Index, r.Title, r.URL)
		if r.Designer != "" {
			fmt.Fprintf(&b, " — %s", r.Designer)
		}
		b.WriteByte('\n')
	}
	b.WriteString("\n")
	b.Write(metaJSON)

	contents := []mcp.Content{mcp.NewTextContent(b.String())}

	if includeImages {
		for _, s := range result.Shots {
			imgURL := images.UpgradeResolution(s.ImageURL, size)
			fetched, err := images.Fetch(ctx, imgURL)
			if err != nil {
				logger.Printf("image fetch failed for %s: %v", imgURL, err)
				// Still include a text note so ordering stays understandable.
				contents = append(contents, mcp.NewTextContent(fmt.Sprintf("(image unavailable for %s: %v)", s.URL, err)))
				continue
			}
			// Label + image so the model can map visuals to links.
			contents = append(contents,
				mcp.NewTextContent(fmt.Sprintf("Image for [%s](%s)", s.Title, s.URL)),
				mcp.NewImageContent(fetched.Base64, fetched.MIMEType),
			)
		}
	}

	return &mcp.CallToolResult{Content: contents}, nil
}

func nonEmpty(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

func uniqueNonEmpty(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		// Normalize on path without query for dedupe.
		key := strings.Split(s, "?")[0]
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, s)
	}
	return out
}
