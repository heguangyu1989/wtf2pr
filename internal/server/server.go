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
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wtf2pr/wtf2pr/internal/config"
	"github.com/wtf2pr/wtf2pr/internal/export"
	"github.com/wtf2pr/wtf2pr/internal/git"
	"github.com/wtf2pr/wtf2pr/internal/review"
	"github.com/wtf2pr/wtf2pr/pkg/models"
)

// Options 创建 server 的参数
type Options struct {
	WorkDir     string
	StaticFS    fs.FS
	Token       string   // 访问令牌，为空则不启用鉴权
	AllowedDirs []string // 允许访问的目录列表（已规范化），为空表示不限制

	dataRoot string // 数据根目录（默认 ~/.wtf2pr），仅包内注入，测试隔离用
}

// Server HTTP server
type Server struct {
	engine      *gin.Engine
	store       *review.Store
	staticFS    fs.FS
	token       string
	dataRoot    string
	allowedDirs []string

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

	dataRoot := opts.dataRoot
	if dataRoot == "" {
		if home, err := os.UserHomeDir(); err == nil {
			dataRoot = filepath.Join(home, ".wtf2pr")
		} else {
			// 无 HOME 时退到临时目录，避免数据静默写入进程工作目录
			if tmp, terr := os.MkdirTemp("", "wtf2pr-data-"); terr == nil {
				dataRoot = tmp
			} else {
				dataRoot = filepath.Join(os.TempDir(), "wtf2pr-data")
			}
			fmt.Fprintf(os.Stderr, "warning: home dir unavailable, using temp data root %s\n", dataRoot)
		}
	}
	// 旧版本可能以 0755 创建了数据根目录，启动时统一收紧
	if info, err := os.Stat(dataRoot); err == nil && info.IsDir() && info.Mode().Perm() != 0700 {
		_ = os.Chmod(dataRoot, 0700)
	}

	workDir := normalizeOrClean(opts.WorkDir)

	s := &Server{
		engine:      r,
		staticFS:    opts.StaticFS,
		token:       opts.Token,
		dataRoot:    dataRoot,
		allowedDirs: opts.AllowedDirs,
		workDir:     workDir,
	}

	wsDir, err := s.setupWorkspace(workDir)
	if err == nil {
		s.storageDir = wsDir
	} else {
		fmt.Fprintf(os.Stderr, "warning: workspace setup failed (%v), data will be stored in %s\n", err, s.dataRoot)
		s.storageDir = s.dataRoot
	}
	s.reviewFile = filepath.Join(s.storageDir, "review.json")

	s.store = review.NewStore(s.reviewFile)
	s.setupRoutes()
	return s
}

// normalizeOrClean 规范化路径，失败时退化为 Clean 后的原路径
func normalizeOrClean(path string) string {
	if normalized, err := config.NormalizePath(path); err == nil {
		return normalized
	}
	return filepath.Clean(path)
}

// workspaceID 根据工作目录路径生成稳定的 workspace ID（解析符号链接，保证同一路径同一 ID）
func workspaceID(workDir string) string {
	sum := sha1.Sum([]byte(normalizeOrClean(workDir)))
	return hex.EncodeToString(sum[:])[:12]
}

// setupWorkspace 为工作目录准备独立的存储目录；首次创建时迁移旧版扁平存储的数据
func (s *Server) setupWorkspace(workDir string) (string, error) {
	wsDir := filepath.Join(s.dataRoot, "workspaces", workspaceID(workDir))
	if _, err := os.Stat(wsDir); err == nil {
		return wsDir, nil
	}
	if err := os.MkdirAll(wsDir, 0700); err != nil {
		return "", err
	}
	meta, _ := json.MarshalIndent(models.WorkspaceMeta{Path: normalizeOrClean(workDir)}, "", "  ")
	_ = os.WriteFile(filepath.Join(wsDir, "workspace.json"), meta, 0600)
	s.migrateLegacyData(wsDir)
	return wsDir, nil
}

// migrateLegacyData 将旧版直接存放在数据根目录下的 review/export.tpl 迁移到 workspace 目录
func (s *Server) migrateLegacyData(wsDir string) {
	matches, _ := filepath.Glob(filepath.Join(s.dataRoot, "review*.json"))
	matches = append(matches, filepath.Join(s.dataRoot, "export.tpl"))
	activeFile := s.getReviewFile()
	for _, src := range matches {
		base := filepath.Base(src)
		// 仅迁移本工具生成的 review.json / review_<id>.json / export.tpl，避免误动其他文件
		if base != "review.json" && base != "export.tpl" && !strings.HasPrefix(base, "review_") {
			continue
		}
		// 跳过当前活动 review 文件（setupWorkspace 失败的降级场景下它位于数据根目录）
		if src == activeFile {
			continue
		}
		dst := filepath.Join(wsDir, base)
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		if err := os.Rename(src, dst); err != nil {
			// 跨文件系统时 rename 失败，退化为复制并删除源文件，避免残留导致重复迁移
			if data, rerr := os.ReadFile(src); rerr == nil {
				if werr := os.WriteFile(dst, data, 0600); werr == nil {
					_ = os.Remove(src)
				}
			}
		} else {
			// 旧版 0644 文件迁移后统一收紧权限
			_ = os.Chmod(dst, 0600)
		}
	}
}

func (s *Server) setupRoutes() {
	api := s.engine.Group("/api")
	api.Use(s.authMiddleware())
	api.Use(bodyLimitMiddleware())
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
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	return srv.ListenAndServe()
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

// maxRequestBodyBytes 限制 API 请求体大小，防止超大 POST 耗尽内存
const maxRequestBodyBytes = 10 << 20 // 10MB

func bodyLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBodyBytes)
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
		if v, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer "); ok {
			token = v
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
	return s.storageDir
}

func (s *Server) getReviewFile() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.reviewFile
}

// currentReviewIDLocked 与 currentReviewID 相同，但调用方必须已持有 s.mu（读或写）
func (s *Server) currentReviewIDLocked() string {
	reviewFileName := filepath.Base(s.reviewFile)
	if strings.HasPrefix(reviewFileName, "review_") && strings.HasSuffix(reviewFileName, ".json") {
		return strings.TrimSuffix(strings.TrimPrefix(reviewFileName, "review_"), ".json")
	}
	return ""
}

func (s *Server) currentReviewID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentReviewIDLocked()
}

// reviewIDPattern 校验 review ID 合法性，防止路径遍历越出存储目录
var reviewIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

// defaultReviewID 默认 review（review.json）在历史列表与切换中使用的标识
const defaultReviewID = "default"

// reviewFileForIDLocked 返回指定 ID 的 review 文件路径；ID 不合法时返回空字符串。调用方必须已持有 s.mu（读或写）
func (s *Server) reviewFileForIDLocked(id string) string {
	if id == defaultReviewID {
		return filepath.Join(s.storageDir, "review.json")
	}
	if !reviewIDPattern.MatchString(id) {
		return ""
	}
	return filepath.Join(s.storageDir, "review_"+id+".json")
}

// readReviewByID 持读锁按 ID 读取 review 文件内容；ID 非法返回空字符串路径
func (s *Server) readReviewByID(id string) (string, []byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	reviewFile := s.reviewFileForIDLocked(id)
	if reviewFile == "" {
		return "", nil, nil
	}
	data, err := os.ReadFile(reviewFile)
	return reviewFile, data, err
}

func (s *Server) handleAuthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: models.AuthCheckResponse{Authenticated: true}})
}

func (s *Server) handleGetConfig(c *gin.Context) {
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: models.ConfigResponse{
		ReviewFile: filepath.Base(s.getReviewFile()),
		ReviewID:   s.currentReviewID(),
		WorkDir:    s.getWorkDir(),
	}})
}

func (s *Server) handleGetWorkdir(c *gin.Context) {
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: s.workdirInfo()})
}

// workdirInfo 汇总当前工作目录与已知工作区信息
func (s *Server) workdirInfo() models.WorkdirInfo {
	return models.WorkdirInfo{
		WorkDir:     s.getWorkDir(),
		Workspaces:  s.listWorkspaces(),
		AllowedDirs: s.allowedDirsList(),
		Restricted:  len(s.allowedDirs) > 0,
	}
}

// allowedDirsList 返回允许目录列表的副本（保证 JSON 序列化为 [] 而非 null）
func (s *Server) allowedDirsList() []string {
	dirs := make([]string, len(s.allowedDirs))
	copy(dirs, s.allowedDirs)
	return dirs
}

// listWorkspaces 扫描数据根目录下已知的 workspace
func (s *Server) listWorkspaces() []models.WorkspaceItem {
	items := []models.WorkspaceItem{}
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
		var meta models.WorkspaceMeta
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
	abs, err := config.NormalizePath(req.Path)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "invalid path"})
		return
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "directory not found: " + abs})
		return
	}
	if !config.IsAllowed(s.allowedDirs, abs) {
		c.JSON(http.StatusForbidden, models.APIResponse{Code: 403, Message: "path is outside the allowed directories: " + abs})
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

	// 字段更新与存储切换必须在同一把锁内完成，避免与并发保存产生交错
	s.mu.Lock()
	s.workDir = abs
	s.storageDir = wsDir
	s.reviewFile = filepath.Join(wsDir, "review.json")
	s.store.SwitchFile(s.reviewFile)
	s.mu.Unlock()

	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: s.workdirInfo()})
}

func (s *Server) handleBrowse(c *gin.Context) {
	path := strings.TrimSpace(c.Query("path"))
	// 受限模式下未指定路径时，返回允许访问的目录列表
	if path == "" && len(s.allowedDirs) > 0 {
		resp := models.BrowseResponse{Path: "", Dirs: []models.DirEntry{}}
		for _, dir := range s.allowedDirs {
			item := models.DirEntry{Name: filepath.Base(dir), Path: dir, IsGit: git.HasGitDir(dir)}
			resp.Dirs = append(resp.Dirs, item)
		}
		c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: resp})
		return
	}
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: err.Error()})
			return
		}
		path = home
	}
	abs, err := config.NormalizePath(path)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "invalid path"})
		return
	}
	if !config.IsAllowed(s.allowedDirs, abs) {
		c.JSON(http.StatusForbidden, models.APIResponse{Code: 403, Message: "path is outside the allowed directories: " + abs})
		return
	}
	f, err := os.Open(abs)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "cannot read directory: " + err.Error()})
		return
	}
	defer f.Close()

	resp := models.BrowseResponse{
		Path:  abs,
		Dirs:  []models.DirEntry{},
		IsGit: git.HasGitDir(abs),
	}
	if parent := filepath.Dir(abs); parent != abs && config.IsAllowed(s.allowedDirs, parent) {
		resp.Parent = parent
	}
	// 流式分批读取，避免巨型目录一次性全部物化；截断时列表为目录序前 500 条（局部排序）
	const maxEntries = 500
	const batchSize = 128
scan:
	for {
		batch, rerr := f.ReadDir(batchSize)
		for _, entry := range batch {
			name := entry.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			full := filepath.Join(abs, name)
			isDir := entry.IsDir()
			if !isDir && entry.Type()&os.ModeSymlink != 0 {
				// 符号链接需 stat 跟随，确认是否指向目录；受限模式下越界链接在切换/进入时被 IsAllowed 拦截
				if info, serr := os.Stat(full); serr == nil {
					isDir = info.IsDir()
				}
			}
			if !isDir {
				continue
			}
			if len(resp.Dirs) >= maxEntries {
				// 集满 500 条后仍存在可见目录才确认截断（恰好 500 条不误报）
				resp.Truncated = true
				break scan
			}
			resp.Dirs = append(resp.Dirs, models.DirEntry{Name: name, Path: full, IsGit: git.HasGitDir(full)})
		}
		if rerr != nil {
			// io.EOF 表示读完；分批读取的其他错误已体现在已收集的结果中
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
	if req.PageSize > 100 {
		req.PageSize = 100
	}
	// 页码上限，防止巨大 page 造成 skip 溢出与无谓的全历史遍历
	if req.Page > 1000000 {
		req.Page = 1000000
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
	// 前端只消费 type/commit/comments/reviewID 与 commitInfo，剥离内嵌 diff 文件内容避免每次全量传输
	if persisted.Diff != nil {
		persisted.Diff = &models.DiffResponse{
			Type:       persisted.Diff.Type,
			Commit:     persisted.Diff.Commit,
			CommitInfo: persisted.Diff.CommitInfo,
			Truncated:  persisted.Diff.Truncated,
		}
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: persisted})
}

func (s *Server) handleGetReviewDetail(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "id is required"})
		return
	}
	// 持读锁贯穿"取路径+读文件"，避免与 workdir 切换交错读到旧 workspace 的文件
	reviewFile, data, err := s.readReviewByID(id)
	if reviewFile == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "invalid review id"})
		return
	}
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "review not found"})
		return
	}
	var pr models.PersistedReview
	if err := json.Unmarshal(data, &pr); err != nil {
		// 旧版 []Comment 格式兜底：列表页对这些文件可见，详情页同样应可读
		var comments []models.Comment
		if lerr := json.Unmarshal(data, &comments); lerr != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "review file corrupted: " + err.Error()})
			return
		}
		pr = models.PersistedReview{Comments: comments}
	}
	pr.Commit = strings.TrimSpace(pr.Commit)
	pr.Type = inferReviewType(pr.Type, pr.Commit)
	if pr.Type == string(models.DiffTypeCommit) && pr.Commit != "" {
		pr.CommitExists = git.CommitsExist(s.getWorkDir(), []string{pr.Commit})[pr.Commit]
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: pr})
}

// inferReviewType 根据存储的 type 与 commit 推断 review 类型
func inferReviewType(t, commit string) string {
	if t != "" {
		return t
	}
	if commit != "" {
		return string(models.DiffTypeCommit)
	}
	return string(models.DiffTypeWorking)
}

func (s *Server) handleSaveReview(c *gin.Context) {
	var req models.SaveReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: err.Error()})
		return
	}

	// 读锁贯穿"读取当前 reviewID 到保存完成"，与切换 workdir/review 互斥
	s.mu.RLock()
	defer s.mu.RUnlock()

	reviewID := s.currentReviewIDLocked()
	now := time.Now().Unix()
	old := s.store.GetPersisted()

	// 如果评论为空，直接删除文件（不写入历史）
	if len(req.Comments) == 0 {
		if err := s.store.Save(&models.PersistedReview{
			ReviewID:  reviewID,
			Type:      string(req.Type),
			Commit:    req.Commit,
			CreatedAt: old.CreatedAt,
			UpdatedAt: now,
			Comments:  []models.Comment{},
		}); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: err.Error()})
			return
		}
		c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok"})
		return
	}

	// 获取当前 diff 上下文并一起保存
	var diffData *models.DiffResponse
	if req.Type != "" {
		diff, err := git.GetDiff(s.workDir, models.DiffRequest{Type: req.Type, Commit: req.Commit})
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
	if err := s.store.Save(persisted); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok"})
}

// reviewListMeta 列表场景下的轻量解析结构：只取所需字段，避免物化内嵌 diff
type reviewListMeta struct {
	Type      string           `json:"type"`
	Commit    string           `json:"commit"`
	CreatedAt int64            `json:"createdAt"`
	UpdatedAt int64            `json:"updatedAt"`
	Comments  []models.Comment `json:"comments"`
	Diff      *struct {
		CommitInfo *models.CommitInfo `json:"commitInfo"`
	} `json:"diff"`
}

func (s *Server) handleGetReviews(c *gin.Context) {
	items := []models.ReviewListItem{}
	dir := s.getStorageDir()
	entries, err := os.ReadDir(dir)
	if err == nil {
		// 候选文件：默认 review.json（id 为 default）+ 命名 review_<id>.json
		type candidate struct{ id, name string }
		candidates := []candidate{{id: defaultReviewID, name: "review.json"}}
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, "review_") && strings.HasSuffix(name, ".json") {
				id := strings.TrimSuffix(strings.TrimPrefix(name, "review_"), ".json")
				if id == defaultReviewID {
					// review_default.json 与 review.json 的 default 特判冲突，跳过避免死链条目
					continue
				}
				candidates = append(candidates, candidate{id: id, name: name})
			}
		}
		var hashes []string
		for _, cand := range candidates {
			item := models.ReviewListItem{
				ReviewID:   cand.id,
				ReviewFile: cand.name,
			}
			fp := filepath.Join(dir, cand.name)
			if data, err := os.ReadFile(fp); err == nil {
				var pr reviewListMeta
				if err := json.Unmarshal(data, &pr); err == nil {
					item.Type = inferReviewType(pr.Type, strings.TrimSpace(pr.Commit))
					item.Commit = strings.TrimSpace(pr.Commit)
					item.CommentCount = len(pr.Comments)
					item.CreatedAt = pr.CreatedAt
					item.UpdatedAt = pr.UpdatedAt
					if pr.Diff != nil && pr.Diff.CommitInfo != nil {
						item.CommitMsg = pr.Diff.CommitInfo.Message
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
				if item.Type == string(models.DiffTypeCommit) && item.Commit != "" {
					hashes = append(hashes, item.Commit)
				}
				items = append(items, item)
			}
		}
		// 批量校验 commit 是否存在（单次 git 子进程，避免 N+1）
		exists := git.CommitsExist(s.getWorkDir(), hashes)
		for i := range items {
			if items[i].Type == string(models.DiffTypeCommit) && items[i].Commit != "" {
				items[i].CommitExists = exists[items[i].Commit]
			}
		}
		// 最近更新在前（updatedAt 相同按 ReviewID 稳定次序）
		sort.Slice(items, func(i, j int) bool {
			if items[i].UpdatedAt != items[j].UpdatedAt {
				return items[i].UpdatedAt > items[j].UpdatedAt
			}
			return items[i].ReviewID < items[j].ReviewID
		})
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

	s.mu.Lock()
	s.reviewFile = s.reviewFileForIDLocked(reviewID)
	s.store.SwitchFile(s.reviewFile)
	reviewFile := s.reviewFile
	s.mu.Unlock()

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

	// 路径构造、存在性检查与切换在同一把锁内完成，避免与 workdir 切换交错
	s.mu.Lock()
	reviewFile := s.reviewFileForIDLocked(req.ReviewID)
	if reviewFile == "" {
		s.mu.Unlock()
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "invalid review id"})
		return
	}
	if _, err := os.Stat(reviewFile); err != nil {
		s.mu.Unlock()
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "review not found"})
		return
	}
	s.reviewFile = reviewFile
	s.store.SwitchFile(reviewFile)
	s.mu.Unlock()

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
		// 持读锁贯穿"取路径+读文件"，避免与 workdir 切换交错读到旧 workspace 的文件
		reviewFile, data, err := s.readReviewByID(req.ReviewID)
		if reviewFile == "" {
			c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "invalid review id"})
			return
		}
		if err != nil {
			// 显式指定的 review 不存在时拒绝导出，避免静默导出成当前 review 的内容
			c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "review not found"})
			return
		}
		var pr models.PersistedReview
		if err := json.Unmarshal(data, &pr); err == nil {
			if pr.Diff != nil {
				diff = pr.Diff
			}
			comments = pr.Comments
		} else {
			// 旧版 []Comment 格式：无内嵌 diff，仅评论可用于导出
			var legacy []models.Comment
			if lerr := json.Unmarshal(data, &legacy); lerr != nil {
				// 与"不存在→404"同一语义：损坏文件不能静默回退导出成当前 review 的内容
				c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "review file corrupted: " + err.Error()})
				return
			}
			comments = legacy
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
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: models.TemplateContent{
		Content: string(content),
	}})
}

func (s *Server) handleSaveTemplate(c *gin.Context) {
	var req models.TemplateContent
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: err.Error()})
		return
	}
	if err := export.ValidateTemplate(req.Content); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: err.Error()})
		return
	}
	if err := review.WriteFileAtomic(s.templatePath(), []byte(req.Content), 0600); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok"})
}
