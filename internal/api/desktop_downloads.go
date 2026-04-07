package api

import (
	"net/http"
	"net/url"
	"strings"

	"md/internal/config"
)

type desktopDownloadVariant struct {
	ID        string `json:"id"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	Label     string `json:"label"`
	URL       string `json:"url,omitempty"`
	Available bool   `json:"available"`
}

type desktopDownloadsHandler struct {
	variants []desktopDownloadVariant
	pageURL  string
	hasAny   bool
}

func newDesktopDownloadsHandler(cfg *config.Config) *desktopDownloadsHandler {
	variants := []desktopDownloadVariant{
		newDesktopVariant("windows-x64", "windows", "x64", "Windows 11 x64", cfg.DesktopDownloads.WindowsX64URL),
		newDesktopVariant("macos-arm64", "macos", "arm64", "macOS Apple Silicon", cfg.DesktopDownloads.MacOSARM64URL),
		newDesktopVariant("macos-amd64", "macos", "x64", "macOS Intel", cfg.DesktopDownloads.MacOSAMD64URL),
		newDesktopVariant("linux-x64", "linux", "x64", "Linux x64", cfg.DesktopDownloads.LinuxX64URL),
	}
	pageURL := sanitizeDesktopDownloadURL(cfg.DesktopDownloads.PageURL)

	hasAny := pageURL != ""
	if !hasAny {
		for _, variant := range variants {
			if variant.Available {
				hasAny = true
				break
			}
		}
	}

	return &desktopDownloadsHandler{
		variants: variants,
		pageURL:  pageURL,
		hasAny:   hasAny,
	}
}

func newDesktopVariant(id, osName, arch, label, rawURL string) desktopDownloadVariant {
	safeURL := sanitizeDesktopDownloadURL(rawURL)
	return desktopDownloadVariant{
		ID:        id,
		OS:        osName,
		Arch:      arch,
		Label:     label,
		URL:       safeURL,
		Available: safeURL != "",
	}
}

func sanitizeDesktopDownloadURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	if strings.HasPrefix(trimmed, "/") && !strings.HasPrefix(trimmed, "//") {
		return trimmed
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "https" && scheme != "http" {
		return ""
	}
	return parsed.String()
}

func (h *desktopDownloadsHandler) list(w http.ResponseWriter, r *http.Request) {
	variants := make([]desktopDownloadVariant, len(h.variants))
	copy(variants, h.variants)

	writeJSON(w, http.StatusOK, map[string]any{
		"variants": variants,
		"page_url": h.pageURL,
		"has_any":  h.hasAny,
	})
}
