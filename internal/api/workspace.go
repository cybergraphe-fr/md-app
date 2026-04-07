package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"md/internal/storage"
)

// ---- context key ----

type ctxKey string

const workspaceCtxKey ctxKey = "workspace"

// WorkspaceFromContext returns the workspace ID from the request context.
func WorkspaceFromContext(r *http.Request) string {
	if v, ok := r.Context().Value(workspaceCtxKey).(string); ok {
		return v
	}
	return ""
}

// ---- workspace registry ----

// WorkspaceInfo maps a sync code to a workspace UUID.
type WorkspaceInfo struct {
	WorkspaceID string    `json:"workspace_id"`
	SyncCode    string    `json:"sync_code"`
	CreatedAt   time.Time `json:"created_at"`
}

// WorkspaceRegistry manages workspace ↔ sync-code mappings on disk.
type WorkspaceRegistry struct {
	basePath string
	mu       sync.RWMutex
}

func NewWorkspaceRegistry(basePath string) *WorkspaceRegistry {
	dir := filepath.Join(basePath, ".workspace-codes")
	_ = os.MkdirAll(dir, 0750)
	return &WorkspaceRegistry{basePath: basePath}
}

var (
	codeRe = regexp.MustCompile(`^[a-z0-9]{8}$`)
	uuidWs = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	errInvalidSyncCode = errors.New("invalid sync code")
	errUnknownSyncCode = errors.New("unknown sync code")
	errWorkspaceNotFound = errors.New("workspace not found")
)

func validWorkspaceID(id string) bool { return uuidWs.MatchString(id) }
func validSyncCode(c string) bool     { return codeRe.MatchString(c) }

// codePath returns the JSON path for a given sync code.
func (wr *WorkspaceRegistry) codePath(code string) string {
	return filepath.Join(wr.basePath, ".workspace-codes", code+".json")
}

// Register creates a new workspace with a unique sync code.
func (wr *WorkspaceRegistry) Register(wsID string) (WorkspaceInfo, error) {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	code, err := generateSyncCode()
	if err != nil {
		return WorkspaceInfo{}, err
	}
	// Ensure uniqueness (very unlikely collision with 8-char codes)
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(wr.codePath(code)); os.IsNotExist(err) {
			break
		}
		code, err = generateSyncCode()
		if err != nil {
			return WorkspaceInfo{}, err
		}
	}

	info := WorkspaceInfo{
		WorkspaceID: wsID,
		SyncCode:    code,
		CreatedAt:   time.Now().UTC(),
	}

	b, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return WorkspaceInfo{}, err
	}
	if err := os.WriteFile(wr.codePath(code), b, 0600); err != nil {
		return WorkspaceInfo{}, err
	}

	// Ensure workspace data dir exists
	wsDir := filepath.Join(wr.basePath, "workspaces", wsID)
	_ = os.MkdirAll(filepath.Join(wsDir, "files"), 0750)
	_ = os.MkdirAll(filepath.Join(wsDir, ".meta"), 0750)

	return info, nil
}

// LookupByCode finds the workspace UUID for a given sync code.
func (wr *WorkspaceRegistry) LookupByCode(code string) (WorkspaceInfo, error) {
	wr.mu.RLock()
	defer wr.mu.RUnlock()

	if !validSyncCode(code) {
		return WorkspaceInfo{}, errInvalidSyncCode
	}

	b, err := os.ReadFile(wr.codePath(code))
	if err != nil {
		if os.IsNotExist(err) {
			return WorkspaceInfo{}, errUnknownSyncCode
		}
		return WorkspaceInfo{}, err
	}

	var info WorkspaceInfo
	if err := json.Unmarshal(b, &info); err != nil {
		return WorkspaceInfo{}, err
	}
	return info, nil
}

// LookupByWorkspace finds the sync code for a given workspace UUID.
func (wr *WorkspaceRegistry) LookupByWorkspace(wsID string) (WorkspaceInfo, error) {
	wr.mu.RLock()
	defer wr.mu.RUnlock()

	codesDir := filepath.Join(wr.basePath, ".workspace-codes")
	entries, err := os.ReadDir(codesDir)
	if err != nil {
		return WorkspaceInfo{}, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(codesDir, e.Name()))
		if err != nil {
			continue
		}
		var info WorkspaceInfo
		if err := json.Unmarshal(b, &info); err != nil {
			continue
		}
		if info.WorkspaceID == wsID {
			return info, nil
		}
	}
	return WorkspaceInfo{}, errWorkspaceNotFound
}

// EnsureWorkspaceInfo returns a workspace mapping and regenerates a sync code when
// the workspace cookie exists but its code mapping was lost.
func (wr *WorkspaceRegistry) EnsureWorkspaceInfo(wsID string) (WorkspaceInfo, bool, error) {
	info, err := wr.LookupByWorkspace(wsID)
	if err == nil {
		return info, false, nil
	}
	if !errors.Is(err, errWorkspaceNotFound) {
		return WorkspaceInfo{}, false, err
	}

	info, err = wr.Register(wsID)
	if err != nil {
		return WorkspaceInfo{}, false, err
	}
	return info, true, nil
}

// generateSyncCode generates an 8-character lowercase alphanumeric code.
func generateSyncCode() (string, error) {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, 8)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		result[i] = chars[n.Int64()]
	}
	return string(result), nil
}

// generateWorkspaceID generates a UUID-like workspace identifier.
func generateWorkspaceID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 2
	return hex.EncodeToString(b[:4]) + "-" +
		hex.EncodeToString(b[4:6]) + "-" +
		hex.EncodeToString(b[6:8]) + "-" +
		hex.EncodeToString(b[8:10]) + "-" +
		hex.EncodeToString(b[10:]), nil
}

// ---- workspace middleware ----

const workspaceCookieName = "md-workspace"

// WorkspaceMiddleware extracts or creates a workspace ID from the cookie.
// It injects the workspace ID into the request context.
func WorkspaceMiddleware(registry *WorkspaceRegistry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var wsID string

			// 1. Try cookie
			if c, err := r.Cookie(workspaceCookieName); err == nil && validWorkspaceID(c.Value) {
				wsID = c.Value
			}

			// 2. If no valid cookie, create new workspace
			if wsID == "" {
				newID, err := generateWorkspaceID()
				if err != nil {
					slog.Error("generate workspace ID", "error", err)
					writeError(w, http.StatusInternalServerError, "workspace creation failed")
					return
				}
				wsID = newID

				// Register with a sync code
				info, err := registry.Register(wsID)
				if err != nil {
					slog.Error("register workspace", "error", err)
					writeError(w, http.StatusInternalServerError, "workspace registration failed")
					return
				}
				slog.Info("new workspace created", "workspace_id", wsID, "sync_code", info.SyncCode)

				// Set cookie (365 days, HttpOnly, SameSite=Lax)
				setWorkspaceCookie(w, wsID)
			}

			// Ensure workspace directories exist
			wsRoot := filepath.Join(registry.basePath, "workspaces", wsID)
			_ = os.MkdirAll(filepath.Join(wsRoot, "files"), 0750)
			_ = os.MkdirAll(filepath.Join(wsRoot, ".meta"), 0750)

			// Inject into context
			ctx := context.WithValue(r.Context(), workspaceCtxKey, wsID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func setWorkspaceCookie(w http.ResponseWriter, wsID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     workspaceCookieName,
		Value:    wsID,
		Path:     "/",
		MaxAge:   365 * 24 * 60 * 60,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// ---- workspace-scoped storage helper ----

// ScopedStorage returns a Storage instance scoped to the workspace in the request context.
func ScopedStorage(basePath string, r *http.Request) *storage.Storage {
	wsID := WorkspaceFromContext(r)
	if wsID == "" {
		return storage.New(basePath) // fallback: unscoped
	}
	return storage.New(filepath.Join(basePath, "workspaces", wsID))
}

// ---- workspace API handlers ----

// MigrateLegacyData moves files from /data/files/ and /data/.meta/ into
// a "default" workspace when the workspaces directory doesn't exist yet.
// This is a one-shot migration for existing deployments.
func MigrateLegacyData(basePath string, registry *WorkspaceRegistry) {
	wsDir := filepath.Join(basePath, "workspaces")
	filesDir := filepath.Join(basePath, "files")
	metaDir := filepath.Join(basePath, ".meta")
	versionsDir := filepath.Join(basePath, ".versions")

	// Skip if workspaces dir already exists (already migrated)
	if _, err := os.Stat(wsDir); err == nil {
		return
	}

	// Skip if legacy files dir doesn't exist
	entries, err := os.ReadDir(filesDir)
	if err != nil || len(entries) == 0 {
		return
	}

	slog.Info("migrating legacy data to default workspace")

	// Create a default workspace
	defaultID, err := generateWorkspaceID()
	if err != nil {
		slog.Error("migration: generate workspace ID", "error", err)
		return
	}

	dstFiles := filepath.Join(wsDir, defaultID, "files")
	dstMeta := filepath.Join(wsDir, defaultID, ".meta")
	dstVersions := filepath.Join(wsDir, defaultID, ".versions")

	_ = os.MkdirAll(dstFiles, 0750)
	_ = os.MkdirAll(dstMeta, 0750)

	// Move files
	for _, e := range entries {
		src := filepath.Join(filesDir, e.Name())
		dst := filepath.Join(dstFiles, e.Name())
		if err := os.Rename(src, dst); err != nil {
			slog.Warn("migration: move file", "src", src, "error", err)
			// Fallback: copy
			if data, readErr := os.ReadFile(src); readErr == nil {
				_ = os.WriteFile(dst, data, 0640)
			}
		}
	}

	// Move meta
	if metaEntries, err := os.ReadDir(metaDir); err == nil {
		for _, e := range metaEntries {
			src := filepath.Join(metaDir, e.Name())
			dst := filepath.Join(dstMeta, e.Name())
			if err := os.Rename(src, dst); err != nil {
				if data, readErr := os.ReadFile(src); readErr == nil {
					_ = os.WriteFile(dst, data, 0640)
				}
			}
		}
	}

	// Move versions
	if vEntries, err := os.ReadDir(versionsDir); err == nil {
		_ = os.MkdirAll(dstVersions, 0750)
		for _, e := range vEntries {
			src := filepath.Join(versionsDir, e.Name())
			dst := filepath.Join(dstVersions, e.Name())
			_ = os.Rename(src, dst)
		}
	}

	// Register the default workspace
	if _, err := registry.Register(defaultID); err != nil {
		slog.Error("migration: register default workspace", "error", err)
		return
	}

	slog.Info("migration complete", "workspace_id", defaultID, "files_migrated", len(entries))
}

type workspaceHandler struct {
	registry *WorkspaceRegistry
}

func newWorkspaceHandler(registry *WorkspaceRegistry) *workspaceHandler {
	return &workspaceHandler{registry: registry}
}

// GET /api/workspace — returns current workspace info (sync code, id)
func (h *workspaceHandler) info(w http.ResponseWriter, r *http.Request) {
	wsID := WorkspaceFromContext(r)
	if wsID == "" {
		writeError(w, http.StatusBadRequest, "no workspace")
		return
	}

	info, repaired, err := h.registry.EnsureWorkspaceInfo(wsID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "workspace lookup failed")
		return
	}
	if repaired {
		slog.Warn("workspace mapping repaired", "workspace_id", wsID, "sync_code", info.SyncCode)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"workspace_id": info.WorkspaceID,
		"sync_code":    info.SyncCode,
		"created_at":   info.CreatedAt,
	})
}

// POST /api/workspace/link — link to an existing workspace by sync code
func (h *workspaceHandler) link(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	code := strings.TrimSpace(strings.ToLower(body.Code))
	if !validSyncCode(code) {
		writeError(w, http.StatusBadRequest, "invalid sync code format (8 lowercase alphanumeric characters)")
		return
	}

	info, err := h.registry.LookupByCode(code)
	if err != nil {
		if errors.Is(err, errUnknownSyncCode) || errors.Is(err, errInvalidSyncCode) {
			writeError(w, http.StatusNotFound, "unknown sync code")
			return
		}
		writeError(w, http.StatusInternalServerError, "workspace lookup failed")
		return
	}

	// Set cookie to the linked workspace
	setWorkspaceCookie(w, info.WorkspaceID)

	slog.Info("workspace linked", "workspace_id", info.WorkspaceID, "sync_code", code)

	writeJSON(w, http.StatusOK, map[string]any{
		"workspace_id": info.WorkspaceID,
		"sync_code":    info.SyncCode,
		"message":      "workspace linked successfully",
	})
}
