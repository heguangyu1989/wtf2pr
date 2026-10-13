package server

import (
	"crypto/sha1"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wtf2pr/wtf2pr/internal/export"
	"github.com/wtf2pr/wtf2pr/internal/git"
	"github.com/wtf2pr/wtf2pr/internal/review"
	"github.com/wtf2pr/wtf2pr/pkg/models"
)

// Options 创建 server 的参数
type Options struct {
	WorkDir    string
	StaticFS   fs.FS
	ReviewFile string // 显式指定 review 文件路径（兼容旧行为，指定后不使用 workspace 存储）
	ReviewID   string
	Token      string // 访问令牌，为空则不启用鉴权
	DataRoot   string // 数据根目录，默认 ~/.wtf2pr
}

// Server HTTP server
type Server struct {
	engine   *gin.Engine
	store    *review.Store
	staticFS fs.FS
	token    string
	dataRoot string

	mu         sync.RWMutex
	workDir    string
	storageDir string
	reviewFile string
}

// NewServer 创建 server
func NewServer(opts Options) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())

	dataRoot := opts.DataRoot
	if dataRoot == "" {
		if home, err := os.UserHomeDir(); err == nil {
			dataRoot = filepath.Join(home, ".wtf2pr")
		}
	}

	s := &Server{
		engine:   r,
		staticFS: opts.StaticFS,
		token:    opts.Token,
		dataRoot: dataRoot,
		workDir:  opts.WorkDir,
	}

	if opts.ReviewFile != "" {
		s.storageDir = filepath.Dir(opts.ReviewFile)
		s.reviewFile = opts.ReviewFile
		_ = os.MkdirAll(s.storageDir, 0755)
	} else {
		wsDir, err := s.setupWorkspace(opts.WorkDir)
		if err == nil {
			s.storageDir = wsDir
		} else {
			s.storageDir = s.dataRoot
		}
		if opts.ReviewID != "" {
			s.reviewFile = filepath.Join(s.storageDir, fmt.Sprintf("review_%s.json", opts.ReviewID))
		} else {
			s.reviewFile = filepath.Join(s.storageDir, "review.json")
		}
	}

	s.store = review.NewStore(s.reviewFile)
	s.setupRoutes()
	return s
}

// workspaceID 根据工作目录绝对路径生成稳定的 workspace ID
func workspaceID(workDir string) string {
	abs, err := filepath.Abs(workDir)
	if err != nil {
		abs = workDir
	}
	sum := sha1.Sum([]byte(filepath.Clean(abs)))
	return hex.EncodeToString(sum[:])[:12]
}

// setupWorkspace 为工作目录准备独立的存储目录；首次创建时迁移旧版扁平存储的数据
func (s *Server) setupWorkspace(workDir string) (string, error) {
	if s.dataRoot == "" {
		return "", fmt.Errorf("data root not available")
	}
	wsDir := filepath.Join(s.dataRoot, "workspaces", workspaceID(workDir))
	if _, err := os.Stat(wsDir); err == nil {
		return wsDir, nil
	}
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		return "", err
	}
	abs, err := filepath.Abs(workDir)
	if err != nil {
		abs = workDir
	}
	meta, _ := json.MarshalIndent(map[string]interface{}{
		"path":      filepath.Clean(abs),
		"createdAt": time.Now().Unix(),
	}, "", "  ")
	_ = os.WriteFile(filepath.Join(wsDir, "workspace.json"), meta, 0644)
	s.migrateLegacyData(wsDir)
	return wsDir, nil
}

// migrateLegacyData 将旧版直接存放在数据根目录下的 review/export.tpl 迁移到 workspace 目录
func (s *Server) migrateLegacyData(wsDir string) {
	matches, _ := filepath.Glob(filepath.Join(s.dataRoot, "review*.json"))
	matches = append(matches, filepath.Join(s.dataRoot, "export.tpl"))
	for _, src := range matches {
		base := filepath.Base(src)
		if base == "workspace.json" {
			continue
		}
		dst := filepath.Join(wsDir, base)
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		if err := os.Rename(src, dst); err != nil {
			if data, rerr := os.ReadFile(src); rerr == nil {
				_ = os.WriteFile(dst, data, 0644)
			}
		}
	}
}

func (s *Server) setupRoutes() {
	api := s.engine.Group("/api")
	api.Use(s.authMiddleware())
	{
		api.GET("/auth/check", s.handleAuthCheck)
		api.GET("/diff", s.handleGetDiff)
		api.GET("/commits", s.handleGetCommits)
		api.GET("/config", s.handleGetConfig)
		api.GET("/review", s.handleGetReview)
		api.GET("/review/detail", s.handleGetReviewDetail)
		api.GET("/reviews", s.handleGetReviews)
		api.POST("/review", s.handleSaveReview)
		api.POST("/review/new", s.handleNewReview)
		api.POST("/review/switch", s.handleSwitchReview)
		api.POST("/export", s.handleExport)
		api.GET("/template", s.handleGetTemplate)
		api.POST("/template", s.handleSaveTemplate)
		api.GET("/workdir", s.handleGetWorkdir)
		api.POST("/workdir", s.handleSwitchWorkdir)
		api.GET("/fs/browse", s.handleBrowse)
	}

	// 静态文件服务
	staticSub, err := fs.Sub(s.staticFS, "dist")
	if err != nil {
		// 如果没有 dist，则不注册静态服务（开发模式由 vite 处理）
		staticSub = s.staticFS
	}
	s.engine.NoRoute(gin.WrapH(http.FileServer(http.FS(staticSub))))
}

// Run 启动服务
func (s *Server) Run(host string, port int) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	return s.engine.Run(addr)
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Auth-Token")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// authMiddleware 校验访问令牌；server 未配置 token 时直接放行
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.token == "" {
			c.Next()
			return
		}
		token := ""
		if auth := c.GetHeader("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimPrefix(auth, "Bearer ")
		}
		if token == "" {
			token = c.GetHeader("X-Auth-Token")
		}
		if token == "" {
			token = c.Query("token")
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.token)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, models.APIResponse{Code: 401, Message: "unauthorized: invalid or missing token"})
			return
		}
		c.Next()
	}
}

func (s *Server) getWorkDir() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.workDir
}

func (s *Server) getStorageDir() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.storageDir == "." || s.storageDir == "" {
		dir, _ := os.Getwd()
		return dir
	}
	return s.storageDir
}

func (s *Server) getReviewFile() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.reviewFile
}

func (s *Server) currentReviewID() string {
	reviewFileName := filepath.Base(s.getReviewFile())
	if strings.HasPrefix(reviewFileName, "review_") && strings.HasSuffix(reviewFileName, ".json") {
		return strings.TrimSuffix(strings.TrimPrefix(reviewFileName, "review_"), ".json")
	}
	return ""
}

func (s *Server) handleAuthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: map[string]bool{
		"authenticated": true,
	}})
}

func (s *Server) handleGetConfig(c *gin.Context) {
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: map[string]string{
		"reviewFile": filepath.Base(s.getReviewFile()),
		"reviewID":   s.currentReviewID(),
		"workDir":    s.getWorkDir(),
		"storageDir": s.getStorageDir(),
	}})
}

func (s *Server) handleGetWorkdir(c *gin.Context) {
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: models.WorkdirInfo{
		WorkDir:    s.getWorkDir(),
		StorageDir: s.getStorageDir(),
		Workspaces: s.listWorkspaces(),
	}})
}

// listWorkspaces 扫描数据根目录下已知的 workspace
func (s *Server) listWorkspaces() []models.WorkspaceItem {
	items := []models.WorkspaceItem{}
	if s.dataRoot == "" {
		return items
	}
	root := filepath.Join(s.dataRoot, "workspaces")
	entries, err := os.ReadDir(root)
	if err != nil {
		return items
	}
	currentID := workspaceID(s.getWorkDir())
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		metaBytes, err := os.ReadFile(filepath.Join(root, entry.Name(), "workspace.json"))
		if err != nil {
			continue
		}
		var meta struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(metaBytes, &meta); err != nil || meta.Path == "" {
			continue
		}
		items = append(items, models.WorkspaceItem{
			ID:      entry.Name(),
			Path:    meta.Path,
			Current: entry.Name() == currentID,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Path < items[j].Path })
	return items
}

func (s *Server) handleSwitchWorkdir(c *gin.Context) {
	var req models.SwitchWorkdirRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: err.Error()})
		return
	}
	abs, err := filepath.Abs(strings.TrimSpace(req.Path))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "invalid path"})
		return
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "directory not found: " + abs})
		return
	}
	if !git.IsGitRepo(abs) {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "not a git repository: " + abs})
		return
	}

	wsDir, err := s.setupWorkspace(abs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: err.Error()})
		return
	}
	reviewFile := filepath.Join(wsDir, "review.json")

	s.mu.Lock()
	s.workDir = filepath.Clean(abs)
	s.storageDir = wsDir
	s.reviewFile = reviewFile
	s.mu.Unlock()
	s.store.SwitchFile(reviewFile)

	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: models.WorkdirInfo{
		WorkDir:    s.getWorkDir(),
		StorageDir: s.getStorageDir(),
		Workspaces: s.listWorkspaces(),
	}})
}

func (s *Server) handleBrowse(c *gin.Context) {
	path := strings.TrimSpace(c.Query("path"))
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: err.Error()})
			return
		}
		path = home
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "invalid path"})
		return
	}
	abs = filepath.Clean(abs)
	entries, err := os.ReadDir(abs)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "cannot read directory: " + err.Error()})
		return
	}

	resp := models.BrowseResponse{
		Path: abs,
		Dirs: []models.DirEntry{},
	}
	if parent := filepath.Dir(abs); parent != abs {
		resp.Parent = parent
	}
	if _, err := os.Stat(filepath.Join(abs, ".git")); err == nil {
		resp.IsGit = true
	}
	const maxEntries = 500
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(abs, name)
		item := models.DirEntry{Name: name, Path: full}
		if _, err := os.Stat(filepath.Join(full, ".git")); err == nil {
			item.IsGit = true
		}
		resp.Dirs = append(resp.Dirs, item)
		if len(resp.Dirs) >= maxEntries {
			break
		}
	}
	sort.Slice(resp.Dirs, func(i, j int) bool { return resp.Dirs[i].Name < resp.Dirs[j].Name })
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: resp})
}

func (s *Server) handleGetCommits(c *gin.Context) {
	var req models.CommitListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: err.Error()})
		return
	}
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 10
	}
	resp, err := git.GetCommits(s.getWorkDir(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: resp})
}

func (s *Server) handleGetDiff(c *gin.Context) {
	var req models.DiffRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: err.Error()})
		return
	}
	if req.Type == "" {
		req.Type = models.DiffTypeWorking
	}
	diff, err := git.GetDiff(s.getWorkDir(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: diff})
}

func (s *Server) handleGetReview(c *gin.Context) {
	persisted := s.store.GetPersisted()
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: persisted})
}

func (s *Server) handleGetReviewDetail(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "id is required"})
		return
	}
	reviewFile := filepath.Join(s.getStorageDir(), "review_"+id+".json")
	data, err := os.ReadFile(reviewFile)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "review not found"})
		return
	}
	var pr models.PersistedReview
	if err := json.Unmarshal(data, &pr); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: err.Error()})
		return
	}
	pr.Commit = strings.TrimSpace(pr.Commit)
	if pr.Type == "" && pr.Commit != "" {
		pr.Type = string(models.DiffTypeCommit)
	} else if pr.Type == "" {
		pr.Type = string(models.DiffTypeWorking)
	}
	if pr.Type == string(models.DiffTypeCommit) && pr.Commit != "" {
		pr.CommitExists = git.CommitExists(s.getWorkDir(), pr.Commit)
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: pr})
}

func (s *Server) handleSaveReview(c *gin.Context) {
	var req models.SaveReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: err.Error()})
		return
	}

	reviewID := s.currentReviewID()
	now := getTimestamp()
	old := s.store.GetPersisted()

	// 如果评论为空，直接删除文件（不写入历史）
	if len(req.Comments) == 0 {
		s.store.Save(&models.PersistedReview{
			ReviewID:  reviewID,
			Type:      string(req.Type),
			Commit:    req.Commit,
			CreatedAt: old.CreatedAt,
			UpdatedAt: now,
			Comments:  []models.Comment{},
		})
		c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok"})
		return
	}

	// 获取当前 diff 上下文并一起保存
	var diffData *models.DiffResponse
	if req.Type != "" {
		diff, err := git.GetDiff(s.getWorkDir(), models.DiffRequest{Type: req.Type, Commit: req.Commit})
		if err == nil {
			diffData = diff
		}
	}

	createdAt := old.CreatedAt
	if createdAt == 0 {
		createdAt = now
	}

	persisted := &models.PersistedReview{
		ReviewID:  reviewID,
		Type:      string(req.Type),
		Commit:    req.Commit,
		CreatedAt: createdAt,
		UpdatedAt: now,
		Diff:      diffData,
		Comments:  req.Comments,
	}
	s.store.Save(persisted)
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok"})
}

func (s *Server) handleGetReviews(c *gin.Context) {
	items := []models.ReviewListItem{}
	dir := s.getStorageDir()
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, "review_") && strings.HasSuffix(name, ".json") {
				id := strings.TrimSuffix(strings.TrimPrefix(name, "review_"), ".json")
				item := models.ReviewListItem{
					ReviewID:   id,
					ReviewFile: name,
				}
				// 尝试读取元信息
				fp := filepath.Join(dir, name)
				if data, err := os.ReadFile(fp); err == nil {
					var pr models.PersistedReview
					if err := json.Unmarshal(data, &pr); err == nil {
						item.Type = pr.Type
						item.Commit = strings.TrimSpace(pr.Commit)
						if item.Type == "" && item.Commit != "" {
							item.Type = string(models.DiffTypeCommit)
						} else if item.Type == "" {
							item.Type = string(models.DiffTypeWorking)
						}
						item.CommentCount = len(pr.Comments)
						item.CreatedAt = pr.CreatedAt
						item.UpdatedAt = pr.UpdatedAt
						if pr.Diff != nil && pr.Diff.CommitInfo != nil {
							item.CommitMsg = pr.Diff.CommitInfo.Message
						}
						if item.Type == string(models.DiffTypeCommit) && item.Commit != "" {
							item.CommitExists = git.CommitExists(s.getWorkDir(), item.Commit)
						}
					} else {
						// old format: []Comment
						var comments []models.Comment
						if err := json.Unmarshal(data, &comments); err == nil {
							item.CommentCount = len(comments)
						}
					}
				}
				if item.CommentCount > 0 {
					items = append(items, item)
				}
			}
		}
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: items})
}

func (s *Server) handleNewReview(c *gin.Context) {
	id, err := uuid.NewRandom()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: err.Error()})
		return
	}
	reviewID := id.String()
	reviewFile := filepath.Join(s.getStorageDir(), "review_"+reviewID+".json")
	s.mu.Lock()
	s.reviewFile = reviewFile
	s.mu.Unlock()
	s.store.SwitchFile(reviewFile)
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: models.NewReviewResponse{
		ReviewID:   reviewID,
		ReviewFile: filepath.Base(reviewFile),
	}})
}

func (s *Server) handleSwitchReview(c *gin.Context) {
	var req models.SwitchReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: err.Error()})
		return
	}
	reviewFile := filepath.Join(s.getStorageDir(), "review_"+req.ReviewID+".json")
	if _, err := os.Stat(reviewFile); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "review not found"})
		return
	}
	s.mu.Lock()
	s.reviewFile = reviewFile
	s.mu.Unlock()
	s.store.SwitchFile(reviewFile)
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: models.NewReviewResponse{
		ReviewID:   req.ReviewID,
		ReviewFile: filepath.Base(reviewFile),
	}})
}

func (s *Server) handleExport(c *gin.Context) {
	var req models.ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: err.Error()})
		return
	}

	var diff *models.DiffResponse
	var comments []models.Comment

	if req.ReviewID != "" {
		reviewFile := filepath.Join(s.getStorageDir(), "review_"+req.ReviewID+".json")
		if data, err := os.ReadFile(reviewFile); err == nil {
			var pr models.PersistedReview
			if err := json.Unmarshal(data, &pr); err == nil {
				if pr.Diff != nil {
					diff = pr.Diff
				}
				comments = pr.Comments
			}
		}
	}

	if diff == nil {
		persisted := s.store.GetPersisted()
		if persisted.Diff != nil && string(req.Type) == persisted.Type && req.Commit == persisted.Commit {
			diff = persisted.Diff
		} else {
			diffReq := models.DiffRequest{Type: req.Type, Commit: req.Commit}
			got, err := git.GetDiff(s.getWorkDir(), diffReq)
			if err != nil {
				c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: err.Error()})
				return
			}
			diff = got
		}
	}

	if comments == nil {
		comments = s.store.Get()
	}

	var content string
	var err error
	if req.Format == models.ExportFormatTemplate {
		tplPath := s.templatePath()
		tplBytes, rerr := os.ReadFile(tplPath)
		if rerr != nil {
			c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "template not found, please configure in template tab"})
			return
		}
		content, err = export.RenderTemplate(diff, comments, string(tplBytes))
	} else {
		content, err = export.Export(diff, comments, req.Format)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: models.ExportResponse{
		Format:  req.Format,
		Content: content,
	}})
}

func (s *Server) templatePath() string {
	return filepath.Join(s.getStorageDir(), "export.tpl")
}

func (s *Server) handleGetTemplate(c *gin.Context) {
	content, _ := os.ReadFile(s.templatePath())
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: map[string]string{
		"content": string(content),
	}})
}

func (s *Server) handleSaveTemplate(c *gin.Context) {
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: err.Error()})
		return
	}
	if err := os.WriteFile(s.templatePath(), []byte(req.Content), 0644); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok"})
}

func getTimestamp() int64 {
	return time.Now().Unix()
}
