package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"md/internal/config"
)

func TestRouterServesDesktopDownloadsFiles(t *testing.T) {
	root := t.TempDir()
	webRoot := filepath.Join(root, "web")
	assetsDir := filepath.Join(webRoot, "assets")
	fontsDir := filepath.Join(webRoot, "fonts")
	downloadsDir := filepath.Join(root, "downloads")
	storageDir := filepath.Join(root, "storage")

	for _, dir := range []string{assetsDir, fontsDir, downloadsDir, storageDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(webRoot, "index.html"), []byte("<html><body>ok</body></html>"), 0o640); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(downloadsDir, "MD-latest-windows-x64.msi"), []byte("fake-msi"), 0o640); err != nil {
		t.Fatalf("write installer file: %v", err)
	}

	cfg := &config.Config{
		WebRoot:             webRoot,
		StoragePath:         storageDir,
		DesktopDownloadsDir: downloadsDir,
		AppURL:              "http://localhost",
		CORSRoots:           []string{"http://localhost"},
	}

	router := NewRouter(cfg, nil, "test")
	req := httptest.NewRequest(http.MethodGet, "/downloads/MD-latest-windows-x64.msi", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	if got := strings.TrimSpace(res.Body.String()); got != "fake-msi" {
		t.Fatalf("expected installer payload, got %q", got)
	}
}

func TestRouterRedirectsDownloadsRoot(t *testing.T) {
	root := t.TempDir()
	webRoot := filepath.Join(root, "web")
	assetsDir := filepath.Join(webRoot, "assets")
	fontsDir := filepath.Join(webRoot, "fonts")
	storageDir := filepath.Join(root, "storage")

	for _, dir := range []string{assetsDir, fontsDir, storageDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(webRoot, "index.html"), []byte("<html><body>ok</body></html>"), 0o640); err != nil {
		t.Fatalf("write index: %v", err)
	}

	cfg := &config.Config{
		WebRoot:     webRoot,
		StoragePath: storageDir,
		AppURL:      "http://localhost",
		CORSRoots:   []string{"http://localhost"},
	}

	router := NewRouter(cfg, nil, "test")
	req := httptest.NewRequest(http.MethodGet, "/downloads", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusMovedPermanently {
		t.Fatalf("expected 301, got %d body=%s", res.Code, res.Body.String())
	}
	if loc := res.Header().Get("Location"); loc != "/downloads/" {
		t.Fatalf("expected /downloads/ redirect, got %q", loc)
	}
}
