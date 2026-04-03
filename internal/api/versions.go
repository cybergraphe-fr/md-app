package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"md/internal/storage"
)

// versionsHandler provides version history endpoints for files.
type versionsHandler struct {
	basePath string
}

func newVersionsHandler(basePath string) *versionsHandler {
	return &versionsHandler{basePath: basePath}
}

func (h *versionsHandler) store(r *http.Request) *storage.Storage {
	return ScopedStorage(h.basePath, r)
}

// GET /api/files/{id}/versions — list all versions for a file (newest first).
func (h *versionsHandler) list(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")

	// Ensure the file exists.
	if _, err := h.store(r).GetMeta(fileID); err != nil {
		if err == storage.ErrNotFound {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		slog.Error("versions list: get meta", "file_id", fileID, "error", err)
		writeError(w, http.StatusInternalServerError, "could not verify file")
		return
	}

	versions, err := h.store(r).ListVersions(fileID)
	if err != nil {
		slog.Error("versions list", "file_id", fileID, "error", err)
		writeError(w, http.StatusInternalServerError, "could not list versions")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"file_id":  fileID,
		"versions": versions,
		"count":    len(versions),
	})
}

// GET /api/files/{id}/versions/{vid} — get a specific version with content.
func (h *versionsHandler) get(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	vid := chi.URLParam(r, "vid")

	vc, err := h.store(r).GetVersion(fileID, vid)
	if err != nil {
		if err == storage.ErrNotFound {
			writeError(w, http.StatusNotFound, "version not found")
			return
		}
		slog.Error("versions get", "file_id", fileID, "version_id", vid, "error", err)
		writeError(w, http.StatusInternalServerError, "could not get version")
		return
	}

	writeJSON(w, http.StatusOK, vc)
}

// POST /api/files/{id}/versions/{vid}/restore — restore a file to a specific version.
func (h *versionsHandler) restore(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	vid := chi.URLParam(r, "vid")

	file, err := h.store(r).RestoreVersion(fileID, vid)
	if err != nil {
		if err == storage.ErrNotFound {
			writeError(w, http.StatusNotFound, "file or version not found")
			return
		}
		slog.Error("versions restore", "file_id", fileID, "version_id", vid, "error", err)
		writeError(w, http.StatusInternalServerError, "could not restore version")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message":  "version restored",
		"file":     file,
		"restored": vid,
	})
}
