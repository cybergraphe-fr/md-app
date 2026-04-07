package desktop

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"md/internal/api"
	"md/internal/config"
)

const (
	defaultDesktopDataDir = "md-desktop"
)

// Runtime represents the local HTTP runtime used by the desktop shell.
type Runtime struct {
	handler     http.Handler
	storagePath string
	webRoot     string
}

// Start boots the local HTTP stack consumed by the desktop webview.
func Start(version string) (*Runtime, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	storagePath, err := resolveStoragePath()
	if err != nil {
		return nil, err
	}
	webRoot, err := resolveWebRoot()
	if err != nil {
		return nil, err
	}

	cfg.StoragePath = storagePath
	cfg.WebRoot = webRoot
	cfg.RedisURL = "" // keep desktop bootstrap dependency-free by default
	cfg.APIKey = ""   // local loopback access only
	cfg.AppURL = "http://127.0.0.1"
	cfg.CORSRoots = []string{"http://127.0.0.1", "http://localhost"}

	if err := os.MkdirAll(cfg.StoragePath, 0750); err != nil {
		return nil, fmt.Errorf("create desktop storage path %q: %w", cfg.StoragePath, err)
	}

	wsRegistry := api.NewWorkspaceRegistry(cfg.StoragePath)
	api.MigrateLegacyData(cfg.StoragePath, wsRegistry)

	router := api.NewRouter(cfg, nil, version)

	rt := &Runtime{
		handler:     router,
		storagePath: cfg.StoragePath,
		webRoot:     cfg.WebRoot,
	}

	slog.Info("desktop runtime started",
		"storage_path", rt.storagePath,
		"web_root", rt.webRoot,
	)

	return rt, nil
}

// Handler returns the runtime HTTP handler used by Wails AssetServer.
func (r *Runtime) Handler() http.Handler {
	return r.handler
}

// Stop keeps the shutdown contract explicit for future runtime resources.
func (r *Runtime) Stop() {
}

func resolveStoragePath() (string, error) {
	if explicit := os.Getenv("MD_DESKTOP_STORAGE"); explicit != "" {
		return explicit, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(base, defaultDesktopDataDir, "files"), nil
}

func resolveWebRoot() (string, error) {
	if explicit := os.Getenv("MD_WEB_ROOT"); explicit != "" {
		if hasIndex(explicit) {
			return explicit, nil
		}
		return "", fmt.Errorf("MD_WEB_ROOT does not contain index.html: %s", explicit)
	}

	candidates := []string{}
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, "web"),
			filepath.Clean(filepath.Join(exeDir, "..", "Resources", "web")),
		)
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "web", "dist"))
	}
	candidates = append(candidates, "/app/web")

	for _, candidate := range candidates {
		if hasIndex(candidate) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("unable to resolve web root for desktop runtime (checked: %v)", candidates)
}

func hasIndex(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "index.html"))
	if err != nil {
		return false
	}
	return !info.IsDir()
}
