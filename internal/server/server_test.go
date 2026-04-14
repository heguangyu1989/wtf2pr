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

func TestHandleNewReview(t *testing.T) {
	srv, _ := setupTestServer(t)
	// First save a comment
	body, _ := json.Marshal(models.SaveReviewRequest{Comments: []models.Comment{{ID: "1", Content: "old"}}})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/review", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w, req)

	// Now create new review
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/review/new", nil)
	srv.engine.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	var resp models.APIResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object data")
	}
	if data["reviewID"] == "" {
		t.Errorf("expected non-empty reviewID")
	}
	if data["reviewFile"] == "" {
		t.Errorf("expected non-empty reviewFile")
	}

	// Verify comments cleared
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/api/review", nil)
	srv.engine.ServeHTTP(w3, req3)
	var resp3 models.APIResponse
	if err := json.Unmarshal(w3.Body.Bytes(), &resp3); err != nil {
		t.Fatalf("failed to unmarshal review get response: %v", err)
	}
	list, ok := resp3.Data.([]interface{})
	if !ok || len(list) != 0 {
		t.Errorf("expected empty comments after new review")
	}
}

func TestHandleGetReviews(t *testing.T) {
	srv, _ := setupTestServer(t)
	// First create a new review to ensure a review file exists
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/review/new", nil)
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for new review, got %d", w.Code)
	}

	// Now list reviews
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/reviews", nil)
	srv.engine.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	var resp models.APIResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	data, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("expected array data, got %T", resp.Data)
	}
	if len(data) == 0 {
		t.Errorf("expected at least one review item")
	}
}

func TestHandleSwitchReview(t *testing.T) {
	srv, _ := setupTestServer(t)
	// Create two reviews with different comments
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/api/review/new", nil)
	srv.engine.ServeHTTP(w1, req1)
	var resp1 models.APIResponse
	if err := json.Unmarshal(w1.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("failed to unmarshal new review response: %v", err)
	}
	id1 := resp1.Data.(map[string]interface{})["reviewID"].(string)

	// Save comment to review1
	body, _ := json.Marshal(models.SaveReviewRequest{Comments: []models.Comment{{ID: "1", Content: "in-review1"}}})
	ws := httptest.NewRecorder()
	reqs, _ := http.NewRequest("POST", "/api/review", bytes.NewReader(body))
	reqs.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(ws, reqs)

	// Create review2
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/review/new", nil)
	srv.engine.ServeHTTP(w2, req2)
	// Save comment to review2
	body2, _ := json.Marshal(models.SaveReviewRequest{Comments: []models.Comment{{ID: "2", Content: "in-review2"}}})
	ws2 := httptest.NewRecorder()
	reqs2, _ := http.NewRequest("POST", "/api/review", bytes.NewReader(body2))
	reqs2.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(ws2, reqs2)

	// Switch back to review1
	w3 := httptest.NewRecorder()
	switchBody, _ := json.Marshal(models.SwitchReviewRequest{ReviewID: id1})
	req3, _ := http.NewRequest("POST", "/api/review/switch", bytes.NewReader(switchBody))
	req3.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("expected 200 for switch, got %d", w3.Code)
	}

	// Verify current review is review1's content
	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("GET", "/api/review", nil)
	srv.engine.ServeHTTP(w4, req4)
	var resp4 models.APIResponse
	if err := json.Unmarshal(w4.Body.Bytes(), &resp4); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	list, ok := resp4.Data.([]interface{})
	if !ok || len(list) != 1 {
		t.Fatalf("expected 1 comment after switch")
	}
	first := list[0].(map[string]interface{})
	if first["content"] != "in-review1" {
		t.Errorf("expected review1 content, got %v", first["content"])
	}

	// Switch to non-existent review should fail
	w5 := httptest.NewRecorder()
	switchBody5, _ := json.Marshal(models.SwitchReviewRequest{ReviewID: "non-existent-uuid"})
	req5, _ := http.NewRequest("POST", "/api/review/switch", bytes.NewReader(switchBody5))
	req5.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w5, req5)
	if w5.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-existent review, got %d", w5.Code)
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
