package models

// DiffType 表示获取 diff 的类型
type DiffType string

const (
	DiffTypeWorking DiffType = "working"
	DiffTypeCommit  DiffType = "commit"
)

// LineType 表示 diff 行的类型
type LineType string

const (
	LineTypeContext  LineType = "context"
	LineTypeAddition LineType = "addition"
	LineTypeDeletion LineType = "deletion"
)

// CommitLog 用于 git log 列表展示
type CommitLog struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
	Author  string `json:"author"`
	Date    string `json:"date"`
}

// CommitListRequest 请求 git log 列表
type CommitListRequest struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"pageSize" form:"page_size"`
}

// CommitListResponse git log 列表响应
type CommitListResponse struct {
	List       []CommitLog `json:"list"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	TotalPages int         `json:"totalPages"`
}

// ExportFormat 表示导出格式
type ExportFormat string

const (
	ExportFormatMarkdown ExportFormat = "markdown"
	ExportFormatJSON     ExportFormat = "json"
	ExportFormatXML      ExportFormat = "xml"
)

// DiffRequest 请求 diff 的参数
type DiffRequest struct {
	Type   DiffType `json:"type" form:"type" binding:"required"`
	Commit string   `json:"commit,omitempty" form:"commit,omitempty"`
}

// DiffResponse 返回的 diff 数据
type DiffResponse struct {
	Type       DiffType    `json:"type"`
	Commit     string      `json:"commit,omitempty"`
	CommitInfo *CommitInfo `json:"commitInfo,omitempty"`
	Files      []FileDiff  `json:"files"`
}

// CommitInfo commit 信息
type CommitInfo struct {
	Hash    string `json:"hash"`
	Author  string `json:"author"`
	Date    string `json:"date"`
	Message string `json:"message"`
}

// FileDiff 文件级别的 diff
type FileDiff struct {
	OldFile   string `json:"oldFile"`
	NewFile   string `json:"newFile"`
	IsNew     bool   `json:"isNew"`
	IsDeleted bool   `json:"isDeleted"`
	IsBinary  bool   `json:"isBinary"`
	Hunks     []Hunk `json:"hunks,omitempty"`
}

// Hunk diff 块
type Hunk struct {
	OldStart int        `json:"oldStart"`
	OldLines int        `json:"oldLines"`
	NewStart int        `json:"newStart"`
	NewLines int        `json:"newLines"`
	Lines    []DiffLine `json:"lines"`
}

// DiffLine 单行 diff
type DiffLine struct {
	Type      LineType `json:"type"`
	OldLineNo int      `json:"oldLineNo"`
	NewLineNo int      `json:"newLineNo"`
	Content   string   `json:"content"`
}

// Comment review 评论
type Comment struct {
	ID        string `json:"id"`
	FilePath  string `json:"filePath"`
	LineKey   string `json:"lineKey,omitempty"` // 格式 "old:{num}" 或 "new:{num}"
	Content   string `json:"content"`
	CreatedAt int64  `json:"createdAt"`
}

// SaveReviewRequest 保存 review 请求
type SaveReviewRequest struct {
	Comments []Comment `json:"comments" binding:"required"`
}

// ExportRequest 导出请求
type ExportRequest struct {
	Format ExportFormat `json:"format" binding:"required,oneof=markdown json xml"`
	Type   DiffType     `json:"type" binding:"required"`
	Commit string       `json:"commit,omitempty"`
}

// ExportResponse 导出响应
type ExportResponse struct {
	Format  ExportFormat `json:"format"`
	Content string       `json:"content"`
}

// APIResponse 通用 API 响应包装
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
