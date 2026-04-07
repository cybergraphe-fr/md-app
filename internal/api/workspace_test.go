package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestWorkspaceHandlerInfo_RepairsMissingMapping(t *testing.T) {
	basePath := t.TempDir()
	registry := NewWorkspaceRegistry(basePath)

	wsID, err := generateWorkspaceID()
	if err != nil {
		t.Fatalf("generateWorkspaceID: %v", err)
	}

	initial, err := registry.Register(wsID)
	if err != nil {
		t.Fatalf("register initial workspace: %v", err)
	}
	if !validSyncCode(initial.SyncCode) {
		t.Fatalf("expected valid initial sync code, got %q", initial.SyncCode)
	}

	if err := os.Remove(registry.codePath(initial.SyncCode)); err != nil {
		t.Fatalf("remove mapping file: %v", err)
	}

	h := newWorkspaceHandler(registry)
	req := httptest.NewRequest(http.MethodGet, "/api/workspace", nil)
	req = req.WithContext(context.WithValue(req.Context(), workspaceCtxKey, wsID))
	res := httptest.NewRecorder()

	h.info(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", res.Code, res.Body.String())
	}

	var payload struct {
		WorkspaceID string `json:"workspace_id"`
		SyncCode    string `json:"sync_code"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.WorkspaceID != wsID {
		t.Fatalf("expected workspace %q, got %q", wsID, payload.WorkspaceID)
	}
	if !validSyncCode(payload.SyncCode) {
		t.Fatalf("expected repaired valid sync code, got %q", payload.SyncCode)
	}

	repaired, err := registry.LookupByWorkspace(wsID)
	if err != nil {
		t.Fatalf("lookup repaired workspace: %v", err)
	}
	if repaired.SyncCode != payload.SyncCode {
		t.Fatalf("expected persisted repaired code %q, got %q", payload.SyncCode, repaired.SyncCode)
	}
}

func TestWorkspaceHandlerInfo_MissingWorkspaceContext(t *testing.T) {
	h := newWorkspaceHandler(NewWorkspaceRegistry(t.TempDir()))
	req := httptest.NewRequest(http.MethodGet, "/api/workspace", nil)
	res := httptest.NewRecorder()

	h.info(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestWorkspaceHandlerLink_UnknownCodeReturnsNotFound(t *testing.T) {
	h := newWorkspaceHandler(NewWorkspaceRegistry(t.TempDir()))
	body := bytes.NewBufferString(`{"code":"abcdefgh"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/workspace/link", body)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	h.link(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", res.Code, res.Body.String())
	}
}
