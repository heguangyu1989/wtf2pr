package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
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
	srv := NewServer(dir, os.DirFS(staticDir), filepath.Join(t.TempDir(), "review.json"))
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
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object data, got %T", resp.Data)
	}
	comments, ok := data["comments"].([]interface{})
	if !ok || len(comments) != 0 {
		t.Errorf("expected empty comments, got %v", data["comments"])
	}
}

func TestHandleSaveReview(t *testing.T) {
	srv, _ := setupTestServer(t)
	payload := models.SaveReviewRequest{
		Comments: []models.Comment{
			{ID: "1", FilePath: "a.go", Content: "test"},
		},
		Type: models.DiffTypeWorking,
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
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object data")
	}
	comments, ok := data["comments"].([]interface{})
	if !ok || len(comments) != 1 {
		t.Fatalf("expected 1 review comment after save")
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
	body, _ := json.Marshal(models.SaveReviewRequest{Comments: []models.Comment{{ID: "1", Content: "old"}}, Type: models.DiffTypeWorking})
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
	pr, ok := resp3.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object data")
	}
	comments, ok := pr["comments"].([]interface{})
	if !ok || len(comments) != 0 {
		t.Errorf("expected empty comments after new review")
	}
}

func TestHandleGetReviews(t *testing.T) {
	srv, _ := setupTestServer(t)
	// Create a new review and save comment so file exists
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/review/new", nil)
	srv.engine.ServeHTTP(w, req)
	var resp1 models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	id1 := resp1.Data.(map[string]interface{})["reviewID"].(string)

	body, _ := json.Marshal(models.SaveReviewRequest{Comments: []models.Comment{{ID: "1", Content: "x"}}, Type: models.DiffTypeWorking})
	ws := httptest.NewRecorder()
	reqs, _ := http.NewRequest("POST", "/api/review", bytes.NewReader(body))
	reqs.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(ws, reqs)

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
	found := false
	for _, item := range data {
		m := item.(map[string]interface{})
		if m["reviewID"] == id1 {
			found = true
			if m["commentCount"].(float64) != 1 {
				t.Errorf("expected commentCount 1")
			}
		}
	}
	if !found {
		t.Errorf("expected review %s in list", id1)
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
	body, _ := json.Marshal(models.SaveReviewRequest{Comments: []models.Comment{{ID: "1", Content: "in-review1"}}, Type: models.DiffTypeWorking})
	ws := httptest.NewRecorder()
	reqs, _ := http.NewRequest("POST", "/api/review", bytes.NewReader(body))
	reqs.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(ws, reqs)

	// Create review2
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/review/new", nil)
	srv.engine.ServeHTTP(w2, req2)
	// Save comment to review2
	body2, _ := json.Marshal(models.SaveReviewRequest{Comments: []models.Comment{{ID: "2", Content: "in-review2"}}, Type: models.DiffTypeWorking})
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
	pr, ok := resp4.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object data")
	}
	comments, ok := pr["comments"].([]interface{})
	if !ok || len(comments) != 1 {
		t.Fatalf("expected 1 comment after switch")
	}
	first := comments[0].(map[string]interface{})
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

func TestHandleGetReviewDetail(t *testing.T) {
	srv, _ := setupTestServer(t)
	// Create and save a review
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/api/review/new", nil)
	srv.engine.ServeHTTP(w1, req1)
	var resp1 models.APIResponse
	if err := json.Unmarshal(w1.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	id1 := resp1.Data.(map[string]interface{})["reviewID"].(string)

	body, _ := json.Marshal(models.SaveReviewRequest{Comments: []models.Comment{{ID: "1", Content: "detail-test"}}, Type: models.DiffTypeWorking})
	ws := httptest.NewRecorder()
	reqs, _ := http.NewRequest("POST", "/api/review", bytes.NewReader(body))
	reqs.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(ws, reqs)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/review/detail?id="+id1, nil)
	srv.engine.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	var resp models.APIResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	pr, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object data")
	}
	comments, ok := pr["comments"].([]interface{})
	if !ok || len(comments) != 1 {
		t.Fatalf("expected 1 comment in detail")
	}

	// well-formed but non-existent id
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/api/review/detail?id=00000000-0000-0000-0000-000000000000", nil)
	srv.engine.ServeHTTP(w3, req3)
	if w3.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w3.Code)
	}
}

func TestHandleGetReviewsFiltersEmpty(t *testing.T) {
	srv, _ := setupTestServer(t)
	// Create a review but do NOT save comments
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/api/review/new", nil)
	srv.engine.ServeHTTP(w1, req1)

	// list should not contain empty review
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/reviews", nil)
	srv.engine.ServeHTTP(w2, req2)
	var resp models.APIResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	data, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("expected array data, got %T", resp.Data)
	}
	for _, item := range data {
		m := item.(map[string]interface{})
		if m["commentCount"].(float64) == 0 {
			t.Errorf("empty review should be filtered")
		}
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

func TestHandleGetTemplate(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/template", nil)
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
	if _, ok := data["content"]; !ok {
		t.Errorf("expected content field")
	}
}

func TestHandleSaveTemplate(t *testing.T) {
	srv, dir := setupTestServer(t)
	tplPath := filepath.Join(dir, ".wtf2pr", "export.tpl")
	_ = os.MkdirAll(filepath.Dir(tplPath), 0755)
	srv.reviewFile = tplPath
	body, _ := json.Marshal(map[string]string{"content": "{{.Type}}"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/template", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	b, err := os.ReadFile(tplPath)
	if err != nil {
		t.Fatalf("failed to read template file: %v", err)
	}
	if string(b) != "{{.Type}}" {
		t.Errorf("expected saved template content")
	}
}

func TestHandleBranchesAndBranchDiff(t *testing.T) {
	srv, dir := setupTestServer(t)
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	base := string(bytes.TrimSpace(out))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/branches", nil)
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected branches 200, got %d: %s", w.Code, w.Body.String())
	}
	var branchResp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &branchResp); err != nil {
		t.Fatal(err)
	}
	data, ok := branchResp.Data.(map[string]interface{})
	if !ok || data["current"] != base {
		t.Fatalf("unexpected branches response: %#v", branchResp.Data)
	}

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/diff?type=branch&base="+base, nil)
	srv.engine.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected branch diff 200, got %d: %s", w2.Code, w2.Body.String())
	}

	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/api/diff?type=branch", nil)
	srv.engine.ServeHTTP(w3, req3)
	if w3.Code != http.StatusBadRequest {
		t.Fatalf("expected missing base 400, got %d", w3.Code)
	}
}

func TestHandleGetFileContent(t *testing.T) {
	srv, dir := setupTestServer(t)
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello world\n"), 0644); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/file/content?type=working&path=a.txt&side=new", nil)
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected file content 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok || data["content"] != "hello world\n" || data["exists"] != true {
		t.Fatalf("unexpected file content: %#v", resp.Data)
	}

	for _, target := range []string{
		"/api/file/content?type=working&path=..%2Foutside.txt&side=new",
		"/api/file/content?type=working&path=.git%2Fconfig&side=new",
		"/api/file/content?type=working&path=a.txt&side=invalid",
	} {
		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest("GET", target, nil)
		srv.engine.ServeHTTP(w2, req2)
		if w2.Code == http.StatusOK || bytes.Contains(w2.Body.Bytes(), []byte("hello world")) {
			t.Fatalf("expected safe rejection for %q, got %d: %s", target, w2.Code, w2.Body.String())
		}
	}
}

func TestHandleExportBranch(t *testing.T) {
	srv, dir := setupTestServer(t)
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	base := string(bytes.TrimSpace(out))
	payload := models.ExportRequest{Format: models.ExportFormatMarkdown, Type: models.DiffTypeBranch, Base: base}
	body, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected branch export 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok || data["format"] != "markdown" {
		t.Fatalf("unexpected branch export response: %#v", resp.Data)
	}
}

func TestHandleGetReviewsEmptyReturnsArray(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/reviews", nil)
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	items, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("expected an empty array, got %#v", resp.Data)
	}
	if len(items) != 0 {
		t.Fatalf("expected no reviews, got %#v", items)
	}
}

func TestHandleGetFileContentErrorStatus(t *testing.T) {
	srv, dir := setupTestServer(t)
	srv.workDir = filepath.Join(t.TempDir(), "missing")
	for _, target := range []string{
		"/api/file/content?type=working&path=a.txt&side=new",
		"/api/file/content?type=commit&commit=HEAD&path=a.txt&side=new",
		"/api/file/content?type=branch&base=main&path=a.txt&side=new",
	} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", target, nil)
		srv.engine.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected repository failure 500, got %d: %s", w.Code, w.Body.String())
		}
	}

	srv.workDir = dir
	for _, target := range []string{
		"/api/file/content?type=commit&commit=missing&path=a.txt&side=new",
		"/api/file/content?type=branch&base=missing&path=a.txt&side=new",
		"/api/file/content?type=branch&path=a.txt&side=new",
	} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", target, nil)
		srv.engine.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected invalid source 400, got %d: %s", w.Code, w.Body.String())
		}
	}
}

func TestHandleSaveReviewRejectsBranch(t *testing.T) {
	srv, _ := setupTestServer(t)
	payload := models.SaveReviewRequest{Comments: []models.Comment{{ID: "1", Content: "branch"}}, Type: models.DiffTypeBranch, Commit: ""}
	body, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/review", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected branch save 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReviewIDRejectsPathTraversal(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/review/detail?id=..%2F..%2Foutside", nil)
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected detail traversal 400, got %d", w.Code)
	}

	payload := models.ExportRequest{Format: models.ExportFormatMarkdown, Type: models.DiffTypeWorking, ReviewID: "../../outside"}
	body, _ := json.Marshal(payload)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/export", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w2, req2)
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("expected export traversal 400, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestHandleExportUsesRequestComments(t *testing.T) {
	srv, dir := setupTestServer(t)
	srv.store.Save(&models.PersistedReview{Type: string(models.DiffTypeCommit), Comments: []models.Comment{{ID: "old", FilePath: "a.txt", Content: "stored commit comment"}}})
	payload := models.ExportRequest{Format: models.ExportFormatMarkdown, Type: models.DiffTypeWorking, Comments: []models.Comment{}}
	body, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected working export 200, got %d", w.Code)
	}
	if bytes.Contains(w.Body.Bytes(), []byte("stored commit comment")) {
		t.Fatalf("working export leaked commit comments: %s", w.Body.String())
	}

	commitCmd := exec.Command("git", "rev-parse", "HEAD")
	commitCmd.Dir = dir
	commitOut, err := commitCmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	draft := models.ExportRequest{
		Format:   models.ExportFormatMarkdown,
		Type:     models.DiffTypeCommit,
		Commit:   string(bytes.TrimSpace(commitOut)),
		Comments: []models.Comment{{ID: "draft", FilePath: "a.txt", Content: "unsaved draft comment"}},
	}
	body2, _ := json.Marshal(draft)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/export", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK || !bytes.Contains(w2.Body.Bytes(), []byte("unsaved draft comment")) {
		t.Fatalf("expected unsaved comments in export, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestHandleSaveReviewDefaultsToWorking(t *testing.T) {
	srv, _ := setupTestServer(t)
	payload := models.SaveReviewRequest{Comments: []models.Comment{{ID: "1", FilePath: "a.txt", Content: "legacy working"}}}
	body, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/review", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected legacy working save 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := srv.store.GetPersisted().Type; got != string(models.DiffTypeWorking) {
		t.Fatalf("expected working type, got %q", got)
	}
}

func TestHandleExportWorkingWithoutCommentsUsesStore(t *testing.T) {
	srv, dir := setupTestServer(t)
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("working change\n"), 0644); err != nil {
		t.Fatal(err)
	}
	srv.store.Save(&models.PersistedReview{Type: string(models.DiffTypeWorking), Comments: []models.Comment{{ID: "1", FilePath: "a.txt", Content: "stored working comment"}}})
	payload := models.ExportRequest{Format: models.ExportFormatMarkdown, Type: models.DiffTypeWorking}
	body, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte("stored working comment")) {
		t.Fatalf("expected stored working comment, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleExportBranchIgnoresStaleSnapshots(t *testing.T) {
	srv, dir := setupTestServer(t)
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %s", args, out)
		}
	}
	run("branch", "-m", "main")
	run("checkout", "-b", "feature")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("CURRENT_MARKER\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", "a.txt")
	run("commit", "-m", "current branch content")
	staleDiff := &models.DiffResponse{Type: models.DiffTypeBranch, Files: []models.FileDiff{{
		OldFile: "a.txt", NewFile: "a.txt",
		Hunks: []models.Hunk{{OldStart: 0, OldLines: 0, NewStart: 1, NewLines: 1, Lines: []models.DiffLine{{Type: models.LineTypeAddition, NewLineNo: 1, Content: "STALE_MARKER"}}}},
	}}}
	comments := []models.Comment{{ID: "1", FilePath: "a.txt", LineKey: "new:1", Content: "comment"}}
	srv.store.Save(&models.PersistedReview{Type: string(models.DiffTypeBranch), Diff: staleDiff, Comments: comments})
	reviewID := "36a0ffea-1b68-4434-91c1-2006ad5593e9"
	persisted := models.PersistedReview{ReviewID: reviewID, Type: string(models.DiffTypeBranch), Diff: staleDiff, Comments: comments}
	data, _ := json.Marshal(persisted)
	if err := os.WriteFile(filepath.Join(filepath.Dir(srv.reviewFile), "review_"+reviewID+".json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	for _, reviewID := range []string{"", reviewID} {
		payload := models.ExportRequest{Format: models.ExportFormatMarkdown, Type: models.DiffTypeBranch, Base: "main", ReviewID: reviewID, Comments: comments}
		body, _ := json.Marshal(payload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/export", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		srv.engine.ServeHTTP(w, req)
		if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte("CURRENT_MARKER")) || bytes.Contains(w.Body.Bytes(), []byte("STALE_MARKER")) {
			t.Errorf("expected fresh branch diff for review %q, got %d: %s", reviewID, w.Code, w.Body.String())
		}
	}
}

func TestHandleSaveReviewCanonicalizesCommit(t *testing.T) {
	srv, dir := setupTestServer(t)
	payload := models.SaveReviewRequest{Comments: []models.Comment{{ID: "1", FilePath: "a.txt", LineKey: "new:1", Content: "canonical"}}, Type: models.DiffTypeCommit, Commit: "HEAD"}
	body, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/review", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected save 200, got %d: %s", w.Code, w.Body.String())
	}
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	want := string(bytes.TrimSpace(out))
	persisted := srv.store.GetPersisted()
	if persisted.Commit != want || persisted.Diff.Commit != want {
		t.Fatalf("expected canonical commit %q, got %q / %#v", want, persisted.Commit, persisted.Diff)
	}
}

func TestHandleExportDoesNotFallbackAcrossCommits(t *testing.T) {
	srv, dir := setupTestServer(t)
	srv.store.Save(&models.PersistedReview{Type: string(models.DiffTypeCommit), Commit: "another-commit", Comments: []models.Comment{{ID: "1", FilePath: "a.txt", Content: "OTHER_COMMIT_PRIVATE_COMMENT"}}})
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	payload := models.ExportRequest{Format: models.ExportFormatMarkdown, Type: models.DiffTypeCommit, Commit: string(bytes.TrimSpace(out))}
	body, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected export 200, got %d", w.Code)
	}
	if bytes.Contains(w.Body.Bytes(), []byte("OTHER_COMMIT_PRIVATE_COMMENT")) {
		t.Fatalf("export leaked another commit's comments: %s", w.Body.String())
	}
}

func TestHandleExportRejectsMissingBase(t *testing.T) {
	srv, _ := setupTestServer(t)
	payload := models.ExportRequest{Format: models.ExportFormatMarkdown, Type: models.DiffTypeBranch}
	body, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected missing base 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleExportWorkingRecomputesDiff(t *testing.T) {
	srv, dir := setupTestServer(t)
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("FIRST\n"), 0644); err != nil {
		t.Fatal(err)
	}
	save := models.SaveReviewRequest{Comments: []models.Comment{{ID: "1", FilePath: "a.txt", LineKey: "new:1", Content: "working"}}, Type: models.DiffTypeWorking}
	body, _ := json.Marshal(save)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/review", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected working save 200, got %d", w.Code)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("SECOND\n"), 0644); err != nil {
		t.Fatal(err)
	}
	exportReq := models.ExportRequest{Format: models.ExportFormatMarkdown, Type: models.DiffTypeWorking}
	exportBody, _ := json.Marshal(exportReq)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/export", bytes.NewReader(exportBody))
	req2.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK || !bytes.Contains(w2.Body.Bytes(), []byte("SECOND")) || bytes.Contains(w2.Body.Bytes(), []byte("FIRST")) {
		t.Fatalf("expected fresh working diff, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestReviewFileAccessConcurrent(t *testing.T) {
	srv, _ := setupTestServer(t)
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/config", nil)
			srv.engine.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("expected config 200, got %d", w.Code)
			}
		}()
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/review/new", nil)
			srv.engine.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("expected new review 200, got %d", w.Code)
			}
		}()
	}
	wg.Wait()
}

func TestHandleExportHistoricalSnapshotWithoutCurrentCommit(t *testing.T) {
	srv, _ := setupTestServer(t)
	reviewID := "36a0ffea-1b68-4434-91c1-2006ad5593e9"
	persisted := models.PersistedReview{
		ReviewID: reviewID,
		Type:     string(models.DiffTypeCommit),
		Commit:   "missing-commit",
		Diff: &models.DiffResponse{Type: models.DiffTypeCommit, Files: []models.FileDiff{{
			OldFile: "a.txt", NewFile: "a.txt",
			Hunks: []models.Hunk{{OldStart: 0, OldLines: 0, NewStart: 1, NewLines: 1, Lines: []models.DiffLine{{Type: models.LineTypeAddition, NewLineNo: 1, Content: "HISTORICAL_SNAPSHOT"}}}},
		}}},
		Comments: []models.Comment{{ID: "1", FilePath: "a.txt", LineKey: "new:1", Content: "history"}},
	}
	data, _ := json.Marshal(persisted)
	if err := os.WriteFile(filepath.Join(filepath.Dir(srv.reviewFile), "review_"+reviewID+".json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	payload := models.ExportRequest{Format: models.ExportFormatMarkdown, Type: models.DiffTypeCommit, Commit: "missing-commit", ReviewID: reviewID}
	body, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte("HISTORICAL_SNAPSHOT")) {
		t.Fatalf("expected historical snapshot export, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCustomReviewIDCompatibility(t *testing.T) {
	srv, dir := setupTestServer(t)
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	commit := string(bytes.TrimSpace(out))
	persisted := models.PersistedReview{
		ReviewID: "feature_x",
		Type:     string(models.DiffTypeCommit),
		Commit:   commit,
		Diff: &models.DiffResponse{Type: models.DiffTypeCommit, Commit: commit, Files: []models.FileDiff{{
			OldFile: "a.txt", NewFile: "a.txt",
			Hunks: []models.Hunk{{OldStart: 0, OldLines: 0, NewStart: 1, NewLines: 1, Lines: []models.DiffLine{{Type: models.LineTypeAddition, NewLineNo: 1, Content: "hello"}}}},
		}}},
		Comments: []models.Comment{{ID: "1", FilePath: "a.txt", LineKey: "new:1", Content: "feature comment"}},
	}
	data, _ := json.Marshal(persisted)
	if err := os.WriteFile(filepath.Join(filepath.Dir(srv.reviewFile), "review_feature_x.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/review/detail?id=feature_x", nil)
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected custom id detail 200, got %d: %s", w.Code, w.Body.String())
	}

	switchBody, _ := json.Marshal(models.SwitchReviewRequest{ReviewID: "feature_x"})
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/review/switch", bytes.NewReader(switchBody))
	req2.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected custom id switch 200, got %d: %s", w2.Code, w2.Body.String())
	}

	exportBody, _ := json.Marshal(models.ExportRequest{Format: models.ExportFormatMarkdown, Type: models.DiffTypeCommit, Commit: commit, ReviewID: "feature_x"})
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("POST", "/api/export", bytes.NewReader(exportBody))
	req3.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK || !bytes.Contains(w3.Body.Bytes(), []byte("feature comment")) {
		t.Fatalf("expected custom id export 200 with comments, got %d: %s", w3.Code, w3.Body.String())
	}
}

func TestSwitchWorkdir(t *testing.T) {
	srv, dirA := setupTestServer(t)
	reviewFile := srv.reviewFile
	srv.store.Save(&models.PersistedReview{Type: "working", Comments: []models.Comment{{ID: "keep", FilePath: "a.txt", Content: "keep-review"}}})
	dirB := initTestRepo(t)
	if err := os.WriteFile(filepath.Join(dirA, "a.txt"), []byte("from-a\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirB, "only-b.txt"), []byte("from-b\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirB, "b-committed.txt"), []byte("committed in B\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "b-committed.txt"}, {"commit", "-m", "only-b-commit"}, {"branch", "only-b-branch"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dirB
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dirB, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	w := workdirRequest(t, srv, filepath.Join(dirB, "nested"), "203.0.113.10:1234", "https://review.example.com", "review.example.com")
	if w.Code != http.StatusOK {
		t.Fatalf("expected workdir switch 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	data, ok := resp.Data.(map[string]interface{})
	canonicalB, err := filepath.EvalSymlinks(dirB)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || data["workDir"] != canonicalB {
		t.Fatalf("unexpected workdir response: %#v", resp.Data)
	}
	if srv.reviewFile != reviewFile {
		t.Fatalf("workdir switch changed review file to %q", srv.reviewFile)
	}
	for _, tc := range []struct {
		name string
		path string
		want string
	}{
		{name: "config", path: "/api/config", want: `"workDir":"` + canonicalB + `"`},
		{name: "diff", path: "/api/diff?type=working", want: "only-b.txt"},
		{name: "commits", path: "/api/commits?page=1&page_size=10", want: "only-b-commit"},
		{name: "branches", path: "/api/branches", want: "only-b-branch"},
		{name: "file content", path: "/api/file/content?type=working&path=only-b.txt&side=new", want: "from-b"},
		{name: "review", path: "/api/review", want: "keep-review"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", tc.path, nil)
			srv.engine.ServeHTTP(w, req)
			if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte(tc.want)) {
				t.Fatalf("expected %q in %s response, got %d: %s", tc.want, tc.name, w.Code, w.Body.String())
			}
		})
	}
	w = httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/diff?type=working", nil)
	srv.engine.ServeHTTP(w, req)
	if bytes.Contains(w.Body.Bytes(), []byte("from-a")) {
		t.Fatalf("repository A content leaked into B diff: %s", w.Body.String())
	}
}

func workdirRequest(t *testing.T, srv *Server, workDir, remoteAddr, origin, host string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(models.WorkdirRequest{WorkDir: workDir})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/config/workdir", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = remoteAddr
	req.Header.Set("Origin", origin)
	req.Host = host
	srv.engine.ServeHTTP(w, req)
	return w
}

func TestSwitchWorkdirRejectsInvalid(t *testing.T) {
	tests := []struct {
		name    string
		workDir func(*testing.T) string
	}{
		{name: "empty", workDir: func(*testing.T) string { return "" }},
		{name: "blank", workDir: func(*testing.T) string { return "  " }},
		{name: "relative", workDir: func(*testing.T) string { return "relative/repo" }},
		{name: "file", workDir: func(t *testing.T) string {
			path := filepath.Join(t.TempDir(), "file.txt")
			if err := os.WriteFile(path, []byte("not a directory"), 0644); err != nil {
				t.Fatal(err)
			}
			return path
		}},
		{name: "missing", workDir: func(t *testing.T) string { return filepath.Join(t.TempDir(), "missing") }},
		{name: "not a repository", workDir: func(t *testing.T) string { return t.TempDir() }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv, _ := setupTestServer(t)
			before := srv.currentWorkDir()
			w := workdirRequest(t, srv, tc.workDir(t), "127.0.0.1:1234", "", "127.0.0.1:8322")
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected invalid workdir 400, got %d: %s", w.Code, w.Body.String())
			}
			if got := srv.currentWorkDir(); got != before {
				t.Errorf("invalid request changed workdir from %q to %q", before, got)
			}
		})
	}
}

func TestSwitchWorkdirConcurrent(t *testing.T) {
	srv, dirA := setupTestServer(t)
	dirB := initTestRepo(t)
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(2)
		go func(index int) {
			defer wg.Done()
			workDir := dirB
			if index%2 == 0 {
				workDir = dirA
			}
			w := workdirRequest(t, srv, workDir, "127.0.0.1:1234", "", "127.0.0.1:8322")
			if w.Code != http.StatusOK {
				t.Errorf("expected concurrent workdir switch 200, got %d", w.Code)
			}
		}(i)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/diff?type=working", nil)
			srv.engine.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("expected concurrent diff 200, got %d", w.Code)
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/branches", nil)
			srv.engine.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("expected concurrent branches 200, got %d", w.Code)
			}
		}()
	}
	wg.Wait()
}
