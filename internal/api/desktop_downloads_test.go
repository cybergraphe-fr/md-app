package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"md/internal/config"
)

func TestSanitizeDesktopDownloadURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "https url", in: "https://example.com/md/windows.exe", want: "https://example.com/md/windows.exe"},
		{name: "http url", in: "http://localhost:8080/downloads/md.zip", want: "http://localhost:8080/downloads/md.zip"},
		{name: "relative path", in: "/downloads/md/windows.exe", want: "/downloads/md/windows.exe"},
		{name: "javascript rejected", in: "javascript:alert(1)", want: ""},
		{name: "protocol relative rejected", in: "//evil.example/md.exe", want: ""},
		{name: "invalid rejected", in: "not a url", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeDesktopDownloadURL(tc.in)
			if got != tc.want {
				t.Fatalf("sanitizeDesktopDownloadURL(%q)=%q want=%q", tc.in, got, tc.want)
			}
		})
	}
}

func TestDesktopDownloadsHandlerList(t *testing.T) {
	cfg := &config.Config{}
	cfg.DesktopDownloads.WindowsX64URL = "https://downloads.example.com/md/windows-x64.exe"
	cfg.DesktopDownloads.MacOSARM64URL = "javascript:alert(1)"
	cfg.DesktopDownloads.PageURL = "/downloads/md"

	h := newDesktopDownloadsHandler(cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/desktop/downloads", nil)
	res := httptest.NewRecorder()

	h.list(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	var payload struct {
		Variants []desktopDownloadVariant `json:"variants"`
		PageURL  string                   `json:"page_url"`
		HasAny   bool                     `json:"has_any"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.PageURL != "/downloads/md" {
		t.Fatalf("expected page_url to be preserved, got %q", payload.PageURL)
	}
	if !payload.HasAny {
		t.Fatalf("expected has_any=true")
	}
	if len(payload.Variants) != 4 {
		t.Fatalf("expected 4 variants, got %d", len(payload.Variants))
	}
	if payload.Variants[0].ID != "windows-x64" || !payload.Variants[0].Available {
		t.Fatalf("expected windows variant to be available")
	}
	if payload.Variants[1].ID != "macos-arm64" || payload.Variants[1].Available {
		t.Fatalf("expected macos-arm64 variant to be unavailable after sanitization")
	}
}

func TestDesktopDownloadsHandlerHasAnyFalse(t *testing.T) {
	h := newDesktopDownloadsHandler(&config.Config{})
	req := httptest.NewRequest(http.MethodGet, "/api/desktop/downloads", nil)
	res := httptest.NewRecorder()

	h.list(res, req)

	var payload struct {
		HasAny bool `json:"has_any"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.HasAny {
		t.Fatalf("expected has_any=false when no desktop links configured")
	}
}
