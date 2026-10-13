package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/wtf2pr/wtf2pr/internal/config"
	"github.com/wtf2pr/wtf2pr/pkg/models"
)

func initTestRepo(t *testing.T) string {
	dir := t.TempDir()
	// macOS 上 $TMPDIR 是 /var 符号链接，统一归一化，保证与服务端路径口径一致
	if normalized, err := config.NormalizePath(dir); err == nil {
		dir = normalized
	}
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
	srv := NewServer(Options{
		WorkDir:  dir,
		StaticFS: os.DirFS(staticDir),
		dataRoot: t.TempDir(),
	})
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

	// non-existent id
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/api/review/detail?id=bad-id", nil)
	srv.engine.ServeHTTP(w3, req3)
	if w3.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w3.Code)
	}
}

func TestHandleGetReviewsFiltersEmpty(t *testing.T) {
	srv, _ := setupTestServer(t)
	// 直接向存储目录写入一个零评论 review 文件和一个有评论 review 文件，真实验证过滤行为
	emptyReview := `{"reviewID":"empty1","type":"working","comments":[]}`
	if err := os.WriteFile(filepath.Join(srv.getStorageDir(), "review_empty1.json"), []byte(emptyReview), 0600); err != nil {
		t.Fatalf("failed to seed empty review: %v", err)
	}
	fullReview := `{"reviewID":"full1","type":"working","comments":[{"id":"1","filePath":"a.go","content":"x"}]}`
	if err := os.WriteFile(filepath.Join(srv.getStorageDir(), "review_full1.json"), []byte(fullReview), 0600); err != nil {
		t.Fatalf("failed to seed full review: %v", err)
	}

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
	foundFull := false
	for _, item := range data {
		m := item.(map[string]interface{})
		if m["reviewID"] == "empty1" {
			t.Errorf("empty review should be filtered")
		}
		if m["reviewID"] == "full1" {
			foundFull = true
		}
	}
	if !foundFull {
		t.Errorf("non-empty review full1 should be listed")
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
	srv.mu.Lock()
	srv.storageDir = filepath.Dir(tplPath)
	srv.mu.Unlock()
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

func TestAuthMiddleware(t *testing.T) {
	dir := initTestRepo(t)
	staticDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(staticDir, "dist"), 0755)
	_ = os.WriteFile(filepath.Join(staticDir, "dist", "index.html"), []byte("<html></html>"), 0644)
	gin.SetMode(gin.TestMode)
	srv := NewServer(Options{
		WorkDir:  dir,
		StaticFS: os.DirFS(staticDir),
		Token:    "secret-token",
		dataRoot: t.TempDir(),
	})

	// 无 token 应返回 401
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/config", nil)
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", w.Code)
	}

	// 错误 token 应返回 401
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/config", nil)
	req2.Header.Set("Authorization", "Bearer wrong-token")
	srv.engine.ServeHTTP(w2, req2)
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong token, got %d", w2.Code)
	}

	// Authorization 头携带正确 token 应放行
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/api/auth/check", nil)
	req3.Header.Set("Authorization", "Bearer secret-token")
	srv.engine.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid token, got %d", w3.Code)
	}

	// query 参数携带正确 token 应放行
	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("GET", "/api/auth/check?token=secret-token", nil)
	srv.engine.ServeHTTP(w4, req4)
	if w4.Code != http.StatusOK {
		t.Fatalf("expected 200 with token query param, got %d", w4.Code)
	}
}

func TestHandleSwitchWorkdir(t *testing.T) {
	srv, dir := setupTestServer(t)
	other := initTestRepo(t)

	// 切换到另一个 git 仓库
	body, _ := json.Marshal(models.SwitchWorkdirRequest{Path: other})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/workdir", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	data := resp.Data.(map[string]interface{})
	if data["workDir"] != other {
		t.Errorf("expected workDir %s, got %v", other, data["workDir"])
	}
	// 服务端存储目录应按工作目录隔离（响应体不再暴露该内部路径，直接断言服务端状态）
	if sd := srv.getStorageDir(); sd == "" || sd == dir || !strings.Contains(sd, "workspaces") {
		t.Errorf("expected per-workdir storage dir, got %v", sd)
	}

	// 新目录下的 review 数据应与旧目录隔离
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/review", nil)
	srv.engine.ServeHTTP(w2, req2)
	var resp2 models.APIResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	pr := resp2.Data.(map[string]interface{})
	if comments, ok := pr["comments"].([]interface{}); !ok || len(comments) != 0 {
		t.Errorf("expected empty comments in new workspace")
	}

	// 切换到非 git 目录应失败
	nonGit := t.TempDir()
	body2, _ := json.Marshal(models.SwitchWorkdirRequest{Path: nonGit})
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("POST", "/api/workdir", bytes.NewReader(body2))
	req3.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w3, req3)
	if w3.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-git dir, got %d", w3.Code)
	}

	// 切换到不存在的目录应失败
	body3, _ := json.Marshal(models.SwitchWorkdirRequest{Path: filepath.Join(nonGit, "not-exist")})
	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("POST", "/api/workdir", bytes.NewReader(body3))
	req4.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w4, req4)
	if w4.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing dir, got %d", w4.Code)
	}
}

func TestHandleGetWorkdir(t *testing.T) {
	srv, dir := setupTestServer(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/workdir", nil)
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	data := resp.Data.(map[string]interface{})
	if data["workDir"] != dir {
		t.Errorf("expected workDir %s, got %v", dir, data["workDir"])
	}
	workspaces, ok := data["workspaces"].([]interface{})
	if !ok || len(workspaces) == 0 {
		t.Fatalf("expected at least one workspace")
	}
	first := workspaces[0].(map[string]interface{})
	if first["current"] != true {
		t.Errorf("expected current workspace to be marked")
	}
}

func TestWorkspaceDataIsolation(t *testing.T) {
	srv, _ := setupTestServer(t)
	other := initTestRepo(t)

	// 在第一个工作目录保存评论
	body, _ := json.Marshal(models.SaveReviewRequest{Comments: []models.Comment{{ID: "1", Content: "repo1"}}, Type: models.DiffTypeWorking})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/review", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w, req)

	// 切换到第二个工作目录
	swBody, _ := json.Marshal(models.SwitchWorkdirRequest{Path: other})
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/workdir", bytes.NewReader(swBody))
	req2.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("switch failed: %s", w2.Body.String())
	}

	// 第二个工作目录不应看到第一个目录的评论
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/api/review", nil)
	srv.engine.ServeHTTP(w3, req3)
	var resp3 models.APIResponse
	_ = json.Unmarshal(w3.Body.Bytes(), &resp3)
	comments := resp3.Data.(map[string]interface{})["comments"].([]interface{})
	if len(comments) != 0 {
		t.Errorf("expected isolated storage, got %d comments", len(comments))
	}
}

func TestHandleBrowse(t *testing.T) {
	srv, _ := setupTestServer(t)
	parent := t.TempDir()
	_ = os.MkdirAll(filepath.Join(parent, "aaa"), 0755)
	_ = os.MkdirAll(filepath.Join(parent, "bbb"), 0755)
	_ = os.MkdirAll(filepath.Join(parent, ".hidden"), 0755)
	_ = os.WriteFile(filepath.Join(parent, "file.txt"), []byte("x"), 0644)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/fs/browse?path="+parent, nil)
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	data := resp.Data.(map[string]interface{})
	dirs := data["dirs"].([]interface{})
	if len(dirs) != 2 {
		t.Errorf("expected 2 visible dirs (hidden/files filtered), got %d", len(dirs))
	}
	if data["parent"] == "" {
		t.Errorf("expected parent to be set")
	}

	// 无效路径
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/fs/browse?path="+filepath.Join(parent, "not-exist"), nil)
	srv.engine.ServeHTTP(w2, req2)
	if w2.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing dir, got %d", w2.Code)
	}
}

func TestLegacyDataMigration(t *testing.T) {
	dir := initTestRepo(t)
	staticDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(staticDir, "dist"), 0755)
	_ = os.WriteFile(filepath.Join(staticDir, "dist", "index.html"), []byte("<html></html>"), 0644)
	dataRoot := t.TempDir()
	// 旧版扁平存储：数据直接放在 dataRoot 下
	legacy := `{"reviewID":"","type":"working","comments":[{"id":"1","filePath":"a.go","content":"legacy"}]}`
	if err := os.WriteFile(filepath.Join(dataRoot, "review.json"), []byte(legacy), 0644); err != nil {
		t.Fatalf("failed to write legacy review: %v", err)
	}
	gin.SetMode(gin.TestMode)
	srv := NewServer(Options{
		WorkDir:  dir,
		StaticFS: os.DirFS(staticDir),
		dataRoot: dataRoot,
	})

	// 旧数据应被迁移到 workspace 目录并自动加载
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/review", nil)
	srv.engine.ServeHTTP(w, req)
	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	comments := resp.Data.(map[string]interface{})["comments"].([]interface{})
	if len(comments) != 1 {
		t.Fatalf("expected migrated legacy review to be loaded, got %d comments", len(comments))
	}
	if _, err := os.Stat(filepath.Join(dataRoot, "review.json")); !os.IsNotExist(err) {
		t.Errorf("expected legacy file to be moved into workspace dir")
	}
}

func setupRestrictedServer(t *testing.T, allowedDirs []string) (*Server, string) {
	dir := initTestRepo(t)
	staticDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(staticDir, "dist"), 0755); err != nil {
		t.Fatalf("failed to create dist dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "dist", "index.html"), []byte("<html></html>"), 0644); err != nil {
		t.Fatalf("failed to write index.html: %v", err)
	}
	gin.SetMode(gin.TestMode)
	// macOS 上 /var 符号链接会导致前缀比较失败，允许列表需与服务端同口径归一化
	normalized := make([]string, 0, len(allowedDirs))
	for _, d := range allowedDirs {
		nd, err := config.NormalizePath(d)
		if err != nil {
			t.Fatalf("invalid allowed dir %s: %v", d, err)
		}
		normalized = append(normalized, nd)
	}
	srv := NewServer(Options{
		WorkDir:     dir,
		StaticFS:    os.DirFS(staticDir),
		dataRoot:    t.TempDir(),
		AllowedDirs: normalized,
	})
	return srv, dir
}

func TestRestrictedSwitchWorkdir(t *testing.T) {
	allowedRepo := initTestRepo(t)
	srv, _ := setupRestrictedServer(t, []string{allowedRepo})

	// 允许列表内的目录可以切换
	body, _ := json.Marshal(models.SwitchWorkdirRequest{Path: allowedRepo})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/workdir", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for allowed dir, got %d, body: %s", w.Code, w.Body.String())
	}

	// 允许列表外的目录应被拒绝（即使是 git 仓库）
	other := initTestRepo(t)
	body2, _ := json.Marshal(models.SwitchWorkdirRequest{Path: other})
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/workdir", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for disallowed dir, got %d", w2.Code)
	}
}

func TestRestrictedBrowse(t *testing.T) {
	parent, err := config.NormalizePath(t.TempDir())
	if err != nil {
		t.Fatalf("failed to normalize: %v", err)
	}
	allowed := filepath.Join(parent, "allowed")
	_ = os.MkdirAll(allowed, 0755)
	srv, _ := setupRestrictedServer(t, []string{allowed})

	// 允许目录内可以浏览
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/fs/browse?path="+allowed, nil)
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for allowed dir, got %d", w.Code)
	}
	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	data := resp.Data.(map[string]interface{})
	if parentVal, _ := data["parent"].(string); parentVal != "" {
		t.Errorf("expected no parent outside allowed dirs, got %q", parentVal)
	}

	// 允许目录外浏览应被拒绝
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/fs/browse?path="+parent, nil)
	srv.engine.ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for disallowed browse, got %d", w2.Code)
	}

	// 空路径在受限模式下返回允许目录列表
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/api/fs/browse", nil)
	srv.engine.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("expected 200 for empty path browse, got %d", w3.Code)
	}
	var resp3 models.APIResponse
	if err := json.Unmarshal(w3.Body.Bytes(), &resp3); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	data3 := resp3.Data.(map[string]interface{})
	dirs := data3["dirs"].([]interface{})
	if len(dirs) != 1 || dirs[0].(map[string]interface{})["path"] != allowed {
		t.Errorf("expected allowed dir list, got %v", dirs)
	}
}

func TestRestrictedWorkdirInfo(t *testing.T) {
	allowed, err := config.NormalizePath(t.TempDir())
	if err != nil {
		t.Fatalf("failed to normalize: %v", err)
	}
	srv, _ := setupRestrictedServer(t, []string{allowed})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/workdir", nil)
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	data := resp.Data.(map[string]interface{})
	if data["restricted"] != true {
		t.Errorf("expected restricted=true")
	}
	dirs := data["allowedDirs"].([]interface{})
	if len(dirs) != 1 || dirs[0] != allowed {
		t.Errorf("expected allowedDirs [%s], got %v", allowed, dirs)
	}
}

func TestReviewIDTraversalRejected(t *testing.T) {
	srv, _ := setupTestServer(t)
	evilIDs := []string{"../../../tmp/evil", "..%2F..%2Ftmp%2Fevil", "a/b", "../review_x"}

	for _, id := range evilIDs {
		// detail
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/review/detail?id="+id, nil)
		srv.engine.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("detail: traversal id %q expected 400, got %d", id, w.Code)
		}

		// switch
		body, _ := json.Marshal(models.SwitchReviewRequest{ReviewID: id})
		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest("POST", "/api/review/switch", bytes.NewReader(body))
		req2.Header.Set("Content-Type", "application/json")
		srv.engine.ServeHTTP(w2, req2)
		if w2.Code != http.StatusBadRequest {
			t.Errorf("switch: traversal id %q expected 400, got %d", id, w2.Code)
		}

		// export
		body3, _ := json.Marshal(models.ExportRequest{Format: models.ExportFormatMarkdown, Type: models.DiffTypeWorking, ReviewID: id})
		w3 := httptest.NewRecorder()
		req3, _ := http.NewRequest("POST", "/api/export", bytes.NewReader(body3))
		req3.Header.Set("Content-Type", "application/json")
		srv.engine.ServeHTTP(w3, req3)
		if w3.Code != http.StatusBadRequest {
			t.Errorf("export: traversal id %q expected 400, got %d", id, w3.Code)
		}
	}
}

func TestSymlinkEscapeRejected(t *testing.T) {
	allowed := t.TempDir()
	outside := initTestRepo(t) // 允许目录之外的 git 仓库
	link := filepath.Join(allowed, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}
	srv, _ := setupRestrictedServer(t, []string{allowed})

	// 浏览允许目录内指向外部的符号链接应被拒绝
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/fs/browse?path="+link, nil)
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for symlink escape browse, got %d", w.Code)
	}

	// 切换到该符号链接路径同样应被拒绝
	body, _ := json.Marshal(models.SwitchWorkdirRequest{Path: link})
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/workdir", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Errorf("expected 403 for symlink escape switch, got %d", w2.Code)
	}
}

func TestXAuthTokenHeader(t *testing.T) {
	dir := initTestRepo(t)
	staticDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(staticDir, "dist"), 0755)
	_ = os.WriteFile(filepath.Join(staticDir, "dist", "index.html"), []byte("<html></html>"), 0644)
	gin.SetMode(gin.TestMode)
	srv := NewServer(Options{WorkDir: dir, StaticFS: os.DirFS(staticDir), Token: "tok", dataRoot: t.TempDir()})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/auth/check", nil)
	req.Header.Set("X-Auth-Token", "tok")
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with X-Auth-Token, got %d", w.Code)
	}
}

func TestWorkspaceIDStable(t *testing.T) {
	dir := initTestRepo(t)
	absID := workspaceID(dir)

	// 相对路径与绝对路径应产生相同 ID
	cwd, _ := os.Getwd()
	rel, err := filepath.Rel(cwd, dir)
	if err == nil && !filepath.IsAbs(rel) {
		if relID := workspaceID(rel); relID != absID {
			t.Errorf("relative path ID %s != absolute ID %s", relID, absID)
		}
	}

	// 符号链接路径应产生相同 ID
	link := filepath.Join(t.TempDir(), "repo-link")
	if err := os.Symlink(dir, link); err == nil {
		if linkID := workspaceID(link); linkID != absID {
			t.Errorf("symlink path ID %s != real ID %s", linkID, absID)
		}
	}
}

func TestMigrationSkipsNonToolFiles(t *testing.T) {
	dir := initTestRepo(t)
	staticDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(staticDir, "dist"), 0755)
	_ = os.WriteFile(filepath.Join(staticDir, "dist", "index.html"), []byte("<html></html>"), 0644)
	dataRoot := t.TempDir()
	// 不带下划线前缀的文件（reviewbackup.json）不属于本工具的 review 文件命名，不应被迁移
	notesPath := filepath.Join(dataRoot, "reviewbackup.json")
	if err := os.WriteFile(notesPath, []byte(`{"keep":true}`), 0644); err != nil {
		t.Fatalf("failed to write notes file: %v", err)
	}
	gin.SetMode(gin.TestMode)
	NewServer(Options{WorkDir: dir, StaticFS: os.DirFS(staticDir), dataRoot: dataRoot})

	if _, err := os.Stat(notesPath); err != nil {
		t.Errorf("non-tool file reviewbackup.json should stay in place: %v", err)
	}
	matches, _ := filepath.Glob(filepath.Join(dataRoot, "workspaces", "*", "reviewbackup.json"))
	if len(matches) > 0 {
		t.Errorf("non-tool file should not be migrated")
	}
}

func TestBrowseDotDotAndTrailingSlash(t *testing.T) {
	parent := t.TempDir()
	allowed := filepath.Join(parent, "allowed")
	_ = os.MkdirAll(filepath.Join(allowed, "sub"), 0755)
	srv, _ := setupRestrictedServer(t, []string{allowed})

	// 尾部斜杠与内部 .. 最终仍在允许范围内 → 放行
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/fs/browse?path="+allowed+"/sub/../", nil)
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for normalized in-scope path, got %d", w.Code)
	}

	// .. 越出允许范围 → 拒绝
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/fs/browse?path="+allowed+"/sub/../../", nil)
	srv.engine.ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Errorf("expected 403 for dot-dot escape, got %d", w2.Code)
	}
}

func TestConcurrentSwitchAndSave(t *testing.T) {
	srv, _ := setupTestServer(t)
	other := initTestRepo(t)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 20; i++ {
			body, _ := json.Marshal(models.SaveReviewRequest{Comments: []models.Comment{{ID: "1", Content: "x"}}, Type: models.DiffTypeWorking})
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/review", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			srv.engine.ServeHTTP(w, req)
		}
	}()
	for i := 0; i < 10; i++ {
		target := other
		if i%2 == 0 {
			target = srv.getWorkDir()
		}
		body, _ := json.Marshal(models.SwitchWorkdirRequest{Path: target})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/workdir", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		srv.engine.ServeHTTP(w, req)
	}
	<-done

	// 不变量：当前 review 文件必须始终位于当前存储目录内（否则说明切换与保存发生了交错）
	reviewFile := srv.getReviewFile()
	storageDir := srv.getStorageDir()
	if !strings.HasPrefix(reviewFile, storageDir+string(os.PathSeparator)) {
		t.Errorf("review file %s escaped storage dir %s", reviewFile, storageDir)
	}
}

func TestHandleExportMissingReviewID(t *testing.T) {
	srv, _ := setupTestServer(t)
	// 显式指定但不存在的 review 应返回 404，而不是静默导出当前 review
	body, _ := json.Marshal(models.ExportRequest{Format: models.ExportFormatMarkdown, Type: models.DiffTypeWorking, ReviewID: "550e8400-e29b-41d4-a716-446655440000"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing review, got %d", w.Code)
	}
}

func TestConcurrentSwitchReviewAndSave(t *testing.T) {
	srv, _ := setupTestServer(t)
	// 先创建一个有内容的 review 供切换
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/api/review/new", nil)
	srv.engine.ServeHTTP(w1, req1)
	var resp1 models.APIResponse
	if err := json.Unmarshal(w1.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	id1 := resp1.Data.(map[string]interface{})["reviewID"].(string)
	body, _ := json.Marshal(models.SaveReviewRequest{Comments: []models.Comment{{ID: "1", Content: "seed"}}, Type: models.DiffTypeWorking})
	ws := httptest.NewRecorder()
	reqs, _ := http.NewRequest("POST", "/api/review", bytes.NewReader(body))
	reqs.Header.Set("Content-Type", "application/json")
	srv.engine.ServeHTTP(ws, reqs)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 20; i++ {
			b, _ := json.Marshal(models.SaveReviewRequest{Comments: []models.Comment{{ID: "x", Content: "concurrent"}}, Type: models.DiffTypeWorking})
			w := httptest.NewRecorder()
			r, _ := http.NewRequest("POST", "/api/review", bytes.NewReader(b))
			r.Header.Set("Content-Type", "application/json")
			srv.engine.ServeHTTP(w, r)
		}
	}()
	for i := 0; i < 10; i++ {
		sb, _ := json.Marshal(models.SwitchReviewRequest{ReviewID: id1})
		w := httptest.NewRecorder()
		r, _ := http.NewRequest("POST", "/api/review/switch", bytes.NewReader(sb))
		r.Header.Set("Content-Type", "application/json")
		srv.engine.ServeHTTP(w, r)
		// 再新建 review 增加交错面
		w2 := httptest.NewRecorder()
		r2, _ := http.NewRequest("POST", "/api/review/new", nil)
		srv.engine.ServeHTTP(w2, r2)
	}
	<-done

	// 不变量：当前 review 文件必须始终位于当前存储目录内
	reviewFile := srv.getReviewFile()
	storageDir := srv.getStorageDir()
	if !strings.HasPrefix(reviewFile, storageDir+string(os.PathSeparator)) {
		t.Errorf("review file %s escaped storage dir %s", reviewFile, storageDir)
	}
}

func TestLegacyExportTemplateMigration(t *testing.T) {
	dir := initTestRepo(t)
	staticDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(staticDir, "dist"), 0755)
	_ = os.WriteFile(filepath.Join(staticDir, "dist", "index.html"), []byte("<html></html>"), 0644)
	dataRoot := t.TempDir()
	// 旧版扁平存储的自定义导出模板应随数据一起迁移
	if err := os.WriteFile(filepath.Join(dataRoot, "export.tpl"), []byte("# {{.Type}}"), 0644); err != nil {
		t.Fatalf("failed to write legacy template: %v", err)
	}
	gin.SetMode(gin.TestMode)
	srv := NewServer(Options{WorkDir: dir, StaticFS: os.DirFS(staticDir), dataRoot: dataRoot})

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
	if got := resp.Data.(map[string]interface{})["content"]; got != "# {{.Type}}" {
		t.Errorf("expected migrated template content, got %v", got)
	}
	if _, err := os.Stat(filepath.Join(dataRoot, "export.tpl")); !os.IsNotExist(err) {
		t.Errorf("legacy export.tpl should be moved into workspace dir")
	}
}

func TestHandleGetCommitsPageSizeCap(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/commits?page=1&page_size=10000000", nil)
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	data := resp.Data.(map[string]interface{})
	if data["pageSize"].(float64) != 100 {
		t.Errorf("expected pageSize capped to 100, got %v", data["pageSize"])
	}
}

func TestHandleBrowseTruncated(t *testing.T) {
	srv, _ := setupTestServer(t)
	parent := t.TempDir()
	// 创建超过 500 上限的目录数
	for i := 0; i < 520; i++ {
		_ = os.MkdirAll(filepath.Join(parent, "d"+strconv.Itoa(i)), 0755)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/fs/browse?path="+parent, nil)
	srv.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	data := resp.Data.(map[string]interface{})
	dirs := data["dirs"].([]interface{})
	if len(dirs) != 500 {
		t.Errorf("expected exactly 500 dirs, got %d", len(dirs))
	}
	if data["truncated"] != true {
		t.Errorf("expected truncated=true")
	}
}
