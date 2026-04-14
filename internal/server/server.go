package server

import (
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
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
		api.GET("/config", s.handleGetConfig)
		api.GET("/review", s.handleGetReview)
		api.POST("/review", s.handleSaveReview)
		api.POST("/export", s.handleExport)
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

func (s *Server) handleGetDiff(c *gin.Context) {
	var req models.DiffRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: err.Error()})
		return
	}
	if req.Type == "" {
		req.Type = models.DiffTypeWorking
	}
	diff, err := git.GetDiff(s.workDir, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: diff})
}

func (s *Server) handleGetReview(c *gin.Context) {
	comments := s.store.Get()
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: comments})
}

func (s *Server) handleSaveReview(c *gin.Context) {
	var req models.SaveReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: err.Error()})
		return
	}
	s.store.Save(req.Comments)
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok"})
}

func (s *Server) handleExport(c *gin.Context) {
	var req models.ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: err.Error()})
		return
	}

	// 获取 diff
	diffReq := models.DiffRequest{Type: req.Type, Commit: req.Commit}
	diff, err := git.GetDiff(s.workDir, diffReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	comments := s.store.Get()
	content, err := export.Export(diff, comments, req.Format)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "ok", Data: models.ExportResponse{
		Format:  req.Format,
		Content: content,
	}})
}
