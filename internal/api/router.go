package api

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"md/internal/cache"
	"md/internal/config"
	"md/internal/plugins"
	"md/internal/webhooks"
)

// NewRouter assembles and returns the full HTTP router.
func NewRouter(cfg *config.Config, c *cache.Client, version string) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Compress(5, "application/json", "text/html", "text/css", "application/javascript"))
	r.Use(loggingMiddleware)
	r.Use(securityHeaders)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSRoots,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-API-Key", "X-Request-ID"},
		ExposedHeaders:   []string{"Content-Disposition", "X-Cache"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Optional OIDC/SSO authentication
	oidcCfg := LoadOIDCConfig()
	r.Use(OIDCMiddleware(oidcCfg))

	// Workspace isolation: cookie-based workspace assignment
	wsRegistry := NewWorkspaceRegistry(cfg.StoragePath)
	r.Use(WorkspaceMiddleware(wsRegistry))

	// Public endpoints
	r.Get("/health", handleHealth(version))
	r.Get("/ready", handleHealth(version)) // k8s readiness compat

	webRoot := cfg.WebRoot
	if webRoot == "" {
		webRoot = "/app/web"
	}
	assetsDir := filepath.Join(webRoot, "assets")
	fontsDir := filepath.Join(webRoot, "fonts")
	indexFile := filepath.Join(webRoot, "index.html")

	// Static frontend assets (served from embedded filesystem or /app/web)
	r.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir(assetsDir))))
	r.Handle("/fonts/*", http.StripPrefix("/fonts/", http.FileServer(http.Dir(fontsDir))))

	// Auth endpoints (always public, handled before OIDC middleware)
	if oidcCfg != nil {
		ah := newAuthHandler(oidcCfg)
		r.Route("/api/auth", func(r chi.Router) {
			r.Get("/login", ah.login)
			r.Get("/callback", ah.callback)
			r.Get("/me", ah.me)
			r.Get("/logout", ah.logout)
		})
	}

	// Initialize shared components
	webhookMgr := webhooks.New(cfg.StoragePath + "/.webhooks.json")
	collabHub := NewCollabHub()
	pluginReg := plugins.NewRegistry()
	_ = pluginReg // available for future render pipeline integration

	// API routes (protected by optional API key)
	r.Group(func(r chi.Router) {
		r.Use(apiKeyMiddleware(cfg))

		fh := newFilesHandler(cfg.StoragePath, c)
		eh := newExportHandler(cfg.StoragePath, cfg)
		th := newTemplatesHandler()
		sh := newSearchHandler(cfg.StoragePath)
		vh := newVersionsHandler(cfg.StoragePath)
		wh := newWebhookHandler(webhookMgr)
		ch := newCollabHandler(collabHub)
		wsh := newWorkspaceHandler(wsRegistry)

		// Workspace endpoints
		r.Get("/api/workspace", wsh.info)
		r.Post("/api/workspace/link", wsh.link)

		r.Route("/api/files", func(r chi.Router) {
			r.Get("/", fh.list)
			r.Post("/", fh.create)
			r.Post("/render", fh.renderRaw)
			r.Post("/import", fh.importFile)

			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", fh.get)
				r.Put("/", fh.update)
				r.Delete("/", fh.delete)
				r.Get("/render", fh.render)
				r.Get("/export/html", fh.exportHTML)
				r.Post("/export/{format}", eh.export)

				// Version history
				r.Get("/versions", vh.list)
				r.Get("/versions/{vid}", vh.get)
				r.Post("/versions/{vid}/restore", vh.restore)

				// Collaborative editing (SSE)
				r.Get("/events", ch.events)
				r.Post("/broadcast", ch.broadcast)
			})
		})

		// Templates
		r.Get("/api/templates", th.list)
		r.Get("/api/templates/{id}", th.get)

		// Search
		r.Get("/api/search", sh.search)

		// Webhooks
		r.Route("/api/webhooks", func(r chi.Router) {
			r.Get("/", wh.list)
			r.Post("/", wh.create)
			r.Put("/{id}", wh.update)
			r.Delete("/{id}", wh.delete)
		})

		// Plugins
		r.Get("/api/plugins", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, map[string]any{"plugins": pluginReg.List()})
		})

		r.Get("/api/export/formats", eh.listFormats)
		r.Post("/api/export/raw/{format}", eh.exportRaw)
	})

	// SPA catch-all – serve index.html for all other routes
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, indexFile)
	})

	return r
}

// ---- helper: JSON decode ----

func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 10<<20)).Decode(v)
}

func marshalJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}
