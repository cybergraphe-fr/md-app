package desktop

import (
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"md/internal/api"
	"md/internal/config"
)

const (
	defaultDesktopDataDir = "md-desktop"
	maxDesktopProxyBody   = 32 << 20 // 32 MB
)

// Runtime represents the local HTTP runtime used by the desktop shell.
type Runtime struct {
	handler     http.Handler
	storagePath string
	webRoot     string
	remoteAPI   string
	mode        string
}

// Start boots the local HTTP stack consumed by the desktop webview.
// embeddedWeb is an optional embedded filesystem containing the built frontend
// (with a "dist" top-level directory). When the frontend cannot be found on
// disk, the embedded FS is used instead, making the binary fully portable.
func Start(version, defaultRemoteAPI string, embeddedWeb fs.FS) (*Runtime, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	storagePath, err := resolveStoragePath()
	if err != nil {
		return nil, err
	}

	// Resolve web assets: try on-disk first, then fall back to embedded FS.
	webRoot, webFS, err := resolveWebAssets(embeddedWeb)
	if err != nil {
		return nil, err
	}

	remoteAPI, err := resolveRemoteAPI(defaultRemoteAPI)
	if err != nil {
		return nil, err
	}

	cfg.StoragePath = storagePath
	if strings.TrimSpace(os.Getenv("MD_DESKTOP_DOWNLOADS_DIR")) == "" {
		cfg.DesktopDownloadsDir = filepath.Join(storagePath, "downloads")
	}
	cfg.WebRoot = webRoot
	cfg.AppURL = "http://127.0.0.1"
	cfg.CORSRoots = []string{"http://127.0.0.1", "http://localhost"}

	var (
		router http.Handler
		mode   = "local"
	)

	if remoteAPI != "" {
		router, err = newConnectedHandler(cfg.WebRoot, remoteAPI, webFS)
		if err != nil {
			return nil, err
		}
		mode = "connected"
	} else {
		cfg.RedisURL = "" // keep desktop bootstrap dependency-free by default
		cfg.APIKey = ""   // local loopback access only

		if err := os.MkdirAll(cfg.StoragePath, 0750); err != nil {
			return nil, fmt.Errorf("create desktop storage path %q: %w", cfg.StoragePath, err)
		}

		wsRegistry := api.NewWorkspaceRegistry(cfg.StoragePath)
		api.MigrateLegacyData(cfg.StoragePath, wsRegistry)

		router = api.NewRouter(cfg, nil, version)
	}

	rt := &Runtime{
		handler:     router,
		storagePath: cfg.StoragePath,
		webRoot:     cfg.WebRoot,
		remoteAPI:   remoteAPI,
		mode:        mode,
	}

	if rt.mode == "connected" {
		slog.Info("desktop runtime started",
			"mode", rt.mode,
			"remote_api", rt.remoteAPI,
			"web_root", rt.webRoot,
		)
	} else {
		slog.Info("desktop runtime started",
			"mode", rt.mode,
			"storage_path", rt.storagePath,
			"web_root", rt.webRoot,
		)
	}

	return rt, nil
}

// Handler returns the runtime HTTP handler used by Wails AssetServer.
func (r *Runtime) Handler() http.Handler {
	return r.handler
}

// Stop keeps the shutdown contract explicit for future runtime resources.
func (r *Runtime) Stop() {
}

func resolveRemoteAPI(defaultRemoteAPI string) (string, error) {
	if explicit := strings.TrimSpace(os.Getenv("MD_DESKTOP_REMOTE_API_URL")); explicit != "" {
		return validateRemoteAPIURL(explicit)
	}
	if fallback := strings.TrimSpace(defaultRemoteAPI); fallback != "" {
		return validateRemoteAPIURL(fallback)
	}
	return "", nil
}

func validateRemoteAPIURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("invalid desktop remote api url %q: %w", raw, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("desktop remote api url must include scheme and host: %q", raw)
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "https" && scheme != "http" {
		return "", fmt.Errorf("desktop remote api url must be http or https: %q", raw)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func newConnectedHandler(webRoot, remoteAPI string, webFS fs.FS) (http.Handler, error) {
	target, err := url.Parse(remoteAPI)
	if err != nil {
		return nil, fmt.Errorf("parse desktop remote api %q: %w", remoteAPI, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = newDesktopProxyTransport()
	proxy.FlushInterval = 100 * time.Millisecond
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
		req.Header.Set("X-MD-Desktop-Proxy", "1")
	}
	proxy.ModifyResponse = relaxSetCookieSecureAttribute
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, proxyErr error) {
		slog.Error("desktop remote proxy error", "path", r.URL.Path, "error", proxyErr)
		writeDesktopProxyError(w)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", proxy)
	mux.Handle("/health", proxy)
	mux.Handle("/ready", proxy)

	// Serve frontend from embedded FS when available, otherwise from disk.
	if webFS != nil {
		fileServer := http.FileServer(http.FS(webFS))
		mux.Handle("/assets/", fileServer)
		mux.Handle("/fonts/", fileServer)
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			f, err := webFS.Open("index.html")
			if err != nil {
				http.Error(w, "frontend not found", http.StatusInternalServerError)
				return
			}
			defer f.Close()
			stat, _ := f.Stat()
			http.ServeContent(w, r, "index.html", stat.ModTime(), f.(readSeeker))
		})
	} else {
		assetsDir := filepath.Join(webRoot, "assets")
		fontsDir := filepath.Join(webRoot, "fonts")
		indexFile := filepath.Join(webRoot, "index.html")
		mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(assetsDir))))
		mux.Handle("/fonts/", http.StripPrefix("/fonts/", http.FileServer(http.Dir(fontsDir))))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, indexFile)
		})
	}

	return withDesktopProxyBodyLimit(mux), nil
}

func newDesktopProxyTransport() *http.Transport {
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          50,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
}

func withDesktopProxyBodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			if r.ContentLength > maxDesktopProxyBody {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				_, _ = w.Write([]byte(`{"error":"request body too large"}`))
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxDesktopProxyBody)
		}
		next.ServeHTTP(w, r)
	})
}

func writeDesktopProxyError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadGateway)
	_, _ = w.Write([]byte(`{"error":"desktop remote sync unavailable"}`))
}

func relaxSetCookieSecureAttribute(resp *http.Response) error {
	setCookies := resp.Header.Values("Set-Cookie")
	if len(setCookies) == 0 {
		return nil
	}

	resp.Header.Del("Set-Cookie")
	for _, raw := range setCookies {
		resp.Header.Add("Set-Cookie", stripSecureCookieFlag(raw))
	}
	return nil
}

func stripSecureCookieFlag(raw string) string {
	parts := strings.Split(raw, ";")
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.EqualFold(strings.TrimSpace(part), "secure") {
			continue
		}
		filtered = append(filtered, part)
	}
	return strings.Join(filtered, ";")
}

// readSeeker combines io.ReadSeeker for http.ServeContent from embed.
type readSeeker = interface {
	Read(p []byte) (n int, err error)
	Seek(offset int64, whence int) (int64, error)
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

// resolveWebAssets returns an on-disk web root path and/or an fs.FS for the
// built frontend. It checks the filesystem first (for development), then falls
// back to the embedded FS from the webdist package (for portable builds).
// At least one must be available.
func resolveWebAssets(embeddedWeb fs.FS) (string, fs.FS, error) {
	// 1. Explicit env var always wins
	if explicit := os.Getenv("MD_WEB_ROOT"); explicit != "" {
		if hasIndex(explicit) {
			return explicit, nil, nil
		}
		return "", nil, fmt.Errorf("MD_WEB_ROOT does not contain index.html: %s", explicit)
	}

	// 2. Try standard filesystem locations
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
			return candidate, nil, nil
		}
	}

	// 3. Fall back to embedded frontend (portable single-binary mode)
	if embeddedWeb != nil {
		sub, err := fs.Sub(embeddedWeb, "dist")
		if err == nil {
			if f, err2 := sub.Open("index.html"); err2 == nil {
				f.Close()
				slog.Info("using embedded web assets (portable mode)")
				return "", sub, nil
			}
		}
	}

	return "", nil, fmt.Errorf("unable to resolve web root for desktop runtime (checked filesystem: %v, embedded: available=%v)", candidates, embeddedWeb != nil)
}

func hasIndex(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "index.html"))
	if err != nil {
		return false
	}
	return !info.IsDir()
}
