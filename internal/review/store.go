package review

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/wtf2pr/wtf2pr/pkg/models"
)

// Store 支持文件持久化的 review 存储
type Store struct {
	mu       sync.RWMutex
	comments []models.Comment
	filePath string
}

// NewStore 创建新存储；如 filePath 存在则自动加载
func NewStore(filePath string) *Store {
	s := &Store{
		comments: []models.Comment{},
		filePath: filePath,
	}
	if filePath != "" {
		if data, err := os.ReadFile(filePath); err == nil {
			_ = json.Unmarshal(data, &s.comments)
		}
	}
	return s
}

// SwitchFile 切换存储文件并加载已有评论
func (s *Store) SwitchFile(filePath string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.filePath = filePath
	s.comments = []models.Comment{}
	if filePath != "" {
		if data, err := os.ReadFile(filePath); err == nil {
			_ = json.Unmarshal(data, &s.comments)
		}
	}
}

// Save 保存评论列表并持久化到 JSON
func (s *Store) Save(comments []models.Comment) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.comments = comments
	if s.filePath != "" {
		data, err := json.MarshalIndent(comments, "", "  ")
		if err == nil {
			_ = os.WriteFile(s.filePath, data, 0644)
		}
	}
}

// Get 获取评论列表
func (s *Store) Get() []models.Comment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]models.Comment, len(s.comments))
	copy(result, s.comments)
	return result
}

// Clear 清空评论
func (s *Store) Clear() {
	s.Save([]models.Comment{})
}
