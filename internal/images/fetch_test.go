package images_test

import (
	"testing"

	"github.com/danecwalker/dribbblemcp/internal/images"
)

func TestUpgradeResolution(t *testing.T) {
	in := "https://cdn.dribbble.com/userupload/1/file/original-x.png?format=webp&resize=320x240&vertical=center"
	out := images.UpgradeResolution(in, "large")
	if out == in {
		t.Fatalf("expected rewritten url, got same")
	}
	if want := "resize=1600x1200"; !contains(out, want) {
		t.Fatalf("expected %q in %q", want, out)
	}
	if contains(out, "320x240") {
		t.Fatalf("old resize should be gone: %s", out)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}
