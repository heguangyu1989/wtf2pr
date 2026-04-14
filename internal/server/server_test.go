package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/wtf2pr/wtf2pr/pkg/models"
)

func initTestRepo(t *testing.T) string {
	dir := t.TempDir()
	commands := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test User"},
	}
	for _, cmd := range commands {
		c := exec.Command(cmd[0], cmd[1:]...)
		c.Dir = dir
		if err := c.Run(); err != nil {
			t.Fatalf("failed to run %v: %v", cmd, err)
		}
	}
	// create and commit a file
	f1 := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(f1, []byte("hello\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	c := exec.Command("git", "add", ".")
	c.Dir = dir
	if err := c.Run(); err != nil {
		t.Fatalf("failed to git add: %v", err)
	}
	c = exec.Command("git", "commit", "-m", "first commit")
	c.Dir = dir
	if err := c.Run(); err != nil {
		t.Fatalf("failed to git commit: %v", err)
	}
	return dir
}

func setupTestServer(t *testing.T) (*Server, string) {
	dir := initTestRepo(t)
	// create a temp static FS with dist/index.html for embed compatibility in tests
	staticDir := t.TempDir()
	distDir := filepath.Join(staticDir, "dist")
	if err := os.MkdirAll(distDir, 0755); err != nil {
		t.Fatalf("failed to create dist dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distDir, "index.html"), []byte("<html></html>"), 0644); err != nil {
		t.Fatalf("failed to write index.html: %v", err)
	}
	gin.SetMode(gin.TestMode)
	srv := NewServer(dir, os.DirFS(staticDir), "")
	return srv, dir
}

func TestHandleGetConfig(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/config", nil)
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
}

func TestHandleGetReview(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/review", nil)
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	data, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("expected array data, got %T", resp.Data)
	}
	if len(data) != 0 {
		t.Errorf("expected empty review, got %d items", len(data))
	}
}

func TestHandleSaveReview(t *testing.T) {
	srv, _ := setupTestServer(t)
	payload := models.SaveReviewRequest{
		Comments: []models.Comment{
			{ID: "1", FilePath: "a.go", Content: "test"},
		},
	}
	body, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/review", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Verify saved
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/review", nil)
	srv.engine.ServeHTTP(w2, req2)
	var resp models.APIResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal review response: %v", err)
	}
	data, ok := resp.Data.([]interface{})
	if !ok || len(data) != 1 {
		t.Fatalf("expected 1 review item after save")
	}
}

func TestHandleGetDiff_Working(t *testing.T) {
	srv, dir := setupTestServer(t)
	// modify file without committing
	f1 := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(f1, []byte("hello world\n"), 0644); err != nil {
		t.Fatalf("failed to write a.txt: %v", err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/diff?type=working", nil)
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object data")
	}
	files, ok := data["files"].([]interface{})
	if !ok || len(files) == 0 {
		t.Errorf("expected working diff to contain at least 1 file")
	}
}

func TestHandleGetCommits(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/commits?page=1&page_size=10", nil)
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object data")
	}
	list, ok := data["list"].([]interface{})
	if !ok || len(list) == 0 {
		t.Errorf("expected at least one commit")
	}
}

func TestHandleExport(t *testing.T) {
	srv, _ := setupTestServer(t)
	payload := models.ExportRequest{Format: models.ExportFormatMarkdown, Type: models.DiffTypeWorking}
	body, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object data")
	}
	if data["format"] != "markdown" {
		t.Errorf("expected markdown format")
	}
}
