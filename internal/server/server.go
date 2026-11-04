package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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

// Server HTTP server
type Server struct {
	engine     *gin.Engine
	store      *review.Store
	workDir    string
	reviewFile string
	staticFS   fs.FS
	reviewMu   sync.RWMutex
}

// NewServer 创建 server
func NewServer(workDir string, staticFS fs.FS, reviewFile string) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())

	s := &Server{
		engine:     r,
		store:      review.NewStore(reviewFile),
		workDir:    workDir,
		reviewFile: reviewFile,
		staticFS:   staticFS,
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	api := s.engine.Group("/api")
	{
		api.GET("/diff", s.handleGetDiff)
		api.GET("/commits", s.handleGetCommits)
		api.GET("/branches", s.handleGetBranches)
		api.GET("/file/content", s.handleGetFileContent)
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
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func (s *Server) handleGetConfig(c *gin.Context) {
	s.reviewMu.RLock()
	defer s.reviewMu.RUnlock()
	reviewFileName := filepath.Base(s.reviewFile)
	reviewID := ""
	if strings.HasPrefix(reviewFileName, "review_") && strings.HasSuffix(reviewFileName, ".json") {
		reviewID = strings.TrimSuffix(strings.TrimPrefix(reviewFileName, "review_"), ".json")
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: map[string]string{
		"reviewFile": reviewFileName,
		"reviewID":   reviewID,
	}})
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
	resp, err := git.GetCommits(s.workDir, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: resp})
}

func (s *Server) handleGetBranches(c *gin.Context) {
	branches, err := git.GetBranches(s.workDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: branches})
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
	switch req.Type {
	case models.DiffTypeWorking:
	case models.DiffTypeCommit:
		if strings.TrimSpace(req.Commit) == "" {
			c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "commit is required"})
			return
		}
	case models.DiffTypeBranch:
		if strings.TrimSpace(req.Base) == "" {
			c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "base branch is required"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "unsupported diff type"})
		return
	}
	diff, err := git.GetDiff(s.workDir, req)
	if err != nil {
		writeGitError(c, err)
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: diff})
}

func (s *Server) handleGetFileContent(c *gin.Context) {
	var req models.FileContentRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: err.Error()})
		return
	}
	content, err := git.GetFileContent(s.workDir, req)
	if err != nil {
		writeGitError(c, err)
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: content})
}

func (s *Server) handleGetReview(c *gin.Context) {
	s.reviewMu.RLock()
	defer s.reviewMu.RUnlock()
	persisted := s.store.GetPersisted()
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: persisted})
}

func (s *Server) handleGetReviewDetail(c *gin.Context) {
	s.reviewMu.RLock()
	defer s.reviewMu.RUnlock()
	id := c.Query("id")
	if !validReviewID(id) {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "invalid review id"})
		return
	}
	reviewFile := filepath.Join(filepath.Dir(s.reviewFile), "review_"+id+".json")
	if s.reviewFile == "" {
		reviewFile = "review_" + id + ".json"
	}
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
		pr.CommitExists = git.CommitExists(s.workDir, pr.Commit)
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: pr})
}

func (s *Server) handleSaveReview(c *gin.Context) {
	s.reviewMu.Lock()
	defer s.reviewMu.Unlock()
	var req models.SaveReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: err.Error()})
		return
	}
	if req.Type == "" {
		req.Type = models.DiffTypeWorking
	}
	if req.Type != models.DiffTypeWorking && req.Type != models.DiffTypeCommit {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "only working and commit reviews can be saved"})
		return
	}
	if req.Type == models.DiffTypeCommit && strings.TrimSpace(req.Commit) == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "commit is required"})
		return
	}
	if req.Type == models.DiffTypeCommit {
		resolvedCommit, err := git.ResolveCommit(s.workDir, req.Commit)
		if err != nil {
			writeGitError(c, err)
			return
		}
		req.Commit = resolvedCommit
	}

	reviewFileName := filepath.Base(s.reviewFile)
	reviewID := ""
	if strings.HasPrefix(reviewFileName, "review_") && strings.HasSuffix(reviewFileName, ".json") {
		reviewID = strings.TrimSuffix(strings.TrimPrefix(reviewFileName, "review_"), ".json")
	}

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
	diff, err := git.GetDiff(s.workDir, models.DiffRequest{Type: req.Type, Commit: req.Commit})
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: err.Error()})
		return
	}
	diffData = diff

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
	s.reviewMu.RLock()
	defer s.reviewMu.RUnlock()
	items := make([]models.ReviewListItem, 0)
	dir := filepath.Dir(s.reviewFile)
	if dir == "." || dir == "" {
		dir, _ = os.Getwd()
	}
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
							item.CommitExists = git.CommitExists(s.workDir, item.Commit)
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
	s.reviewMu.Lock()
	defer s.reviewMu.Unlock()
	id, err := uuid.NewRandom()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: err.Error()})
		return
	}
	reviewID := id.String()
	reviewFile := filepath.Join(filepath.Dir(s.reviewFile), "review_"+reviewID+".json")
	if s.reviewFile == "" {
		reviewFile = "review_" + reviewID + ".json"
	}
	s.reviewFile = reviewFile
	s.store.SwitchFile(reviewFile)
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: models.NewReviewResponse{
		ReviewID:   reviewID,
		ReviewFile: filepath.Base(reviewFile),
	}})
}

func (s *Server) handleSwitchReview(c *gin.Context) {
	s.reviewMu.Lock()
	defer s.reviewMu.Unlock()
	var req models.SwitchReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: err.Error()})
		return
	}
	if !validReviewID(req.ReviewID) {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "invalid review id"})
		return
	}
	reviewFile := filepath.Join(filepath.Dir(s.reviewFile), "review_"+req.ReviewID+".json")
	if s.reviewFile == "" {
		reviewFile = "review_" + req.ReviewID + ".json"
	}
	if _, err := os.Stat(reviewFile); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "review not found"})
		return
	}
	s.reviewFile = reviewFile
	s.store.SwitchFile(reviewFile)
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: models.NewReviewResponse{
		ReviewID:   req.ReviewID,
		ReviewFile: filepath.Base(reviewFile),
	}})
}

func (s *Server) handleExport(c *gin.Context) {
	s.reviewMu.RLock()
	defer s.reviewMu.RUnlock()
	var req models.ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: err.Error()})
		return
	}
	if req.ReviewID != "" && !validReviewID(req.ReviewID) {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "invalid review id"})
		return
	}
	var diff *models.DiffResponse
	var comments []models.Comment

	if req.ReviewID != "" {
		reviewFile := filepath.Join(filepath.Dir(s.reviewFile), "review_"+req.ReviewID+".json")
		if s.reviewFile == "" {
			reviewFile = "review_" + req.ReviewID + ".json"
		}
		if data, err := os.ReadFile(reviewFile); err == nil {
			var pr models.PersistedReview
			if err := json.Unmarshal(data, &pr); err == nil {
				if pr.Diff != nil && req.Type != models.DiffTypeBranch {
					diff = pr.Diff
				}
				comments = pr.Comments
			}
		}
	}

	if req.Type == models.DiffTypeCommit && diff == nil {
		resolvedCommit, err := git.ResolveCommit(s.workDir, req.Commit)
		if err != nil {
			writeGitError(c, err)
			return
		}
		req.Commit = resolvedCommit
	}
	if diff == nil {
		persisted := s.store.GetPersisted()
		if persisted.Diff != nil && req.Type != models.DiffTypeBranch && req.Type != models.DiffTypeWorking && string(req.Type) == persisted.Type && req.Commit == persisted.Commit {
			diff = persisted.Diff
		} else {
			diffReq := models.DiffRequest{Type: req.Type, Commit: req.Commit, Base: req.Base}
			got, err := git.GetDiff(s.workDir, diffReq)
			if err != nil {
				writeGitError(c, err)
				return
			}
			diff = got
		}
	}

	if req.ReviewID != "" {
		if comments == nil {
			comments = []models.Comment{}
		}
	} else if (req.Type == models.DiffTypeWorking || req.Type == models.DiffTypeCommit) && req.Comments == nil {
		persisted := s.store.GetPersisted()
		if string(req.Type) == persisted.Type && req.Commit == persisted.Commit {
			comments = s.store.Get()
		} else {
			comments = []models.Comment{}
		}
	} else {
		comments = req.Comments
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
	dir := filepath.Dir(s.reviewFile)
	if dir == "." || dir == "" {
		dir, _ = os.Getwd()
	}
	return filepath.Join(dir, "export.tpl")
}

func (s *Server) handleGetTemplate(c *gin.Context) {
	s.reviewMu.RLock()
	defer s.reviewMu.RUnlock()
	content, _ := os.ReadFile(s.templatePath())
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: map[string]string{
		"content": string(content),
	}})
}

func (s *Server) handleSaveTemplate(c *gin.Context) {
	s.reviewMu.Lock()
	defer s.reviewMu.Unlock()
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

func writeGitError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	if errors.Is(err, git.ErrInvalidFileContent) {
		status = http.StatusBadRequest
	}
	c.JSON(status, models.APIResponse{Code: status, Message: err.Error()})
}

var reviewIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func validReviewID(id string) bool {
	return len(id) > 0 && len(id) <= 128 && !strings.Contains(id, "..") && reviewIDPattern.MatchString(id)
}

func getTimestamp() int64 {
	return time.Now().Unix()
}
