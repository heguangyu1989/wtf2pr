package review

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/wtf2pr/wtf2pr/pkg/models"
)

// Store 支持文件持久化的 review 存储
type Store struct {
	mu        sync.RWMutex
	filePath  string
	persisted *models.PersistedReview
}

// NewStore 创建新存储；如 filePath 存在则自动加载
func NewStore(filePath string) *Store {
	s := &Store{
		filePath: filePath,
		persisted: &models.PersistedReview{
			Comments: []models.Comment{},
		},
	}
	if filePath != "" {
		if data, err := os.ReadFile(filePath); err == nil {
			s.loadData(data)
		}
	}
	return s
}

func (s *Store) loadData(data []byte) {
	var pr models.PersistedReview
	if err := json.Unmarshal(data, &pr); err == nil {
		s.persisted = &pr
		return
	}
	// backward compatibility: old format is []Comment
	var comments []models.Comment
	if err := json.Unmarshal(data, &comments); err == nil {
		s.persisted.Comments = comments
	}
}

// Save 保存 review 记录；若评论为空则删除文件
func (s *Store) Save(persisted *models.PersistedReview) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.persisted = persisted
	if s.filePath == "" {
		return nil
	}
	if len(persisted.Comments) == 0 {
		if err := os.Remove(s.filePath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	data, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		return err
	}
	return WriteFileAtomic(s.filePath, data, 0600)
}

// WriteFileAtomic 写临时文件后原子替换，避免进程崩溃或并发读到半写内容
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	// WriteFile 对已存在文件不更新权限位（兼容旧版 0644 数据），统一收紧
	_ = os.Chmod(tmp, perm)
	return os.Rename(tmp, path)
}

// Get 获取评论列表
func (s *Store) Get() []models.Comment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]models.Comment, len(s.persisted.Comments))
	copy(result, s.persisted.Comments)
	return result
}

// GetPersisted 获取完整持久化记录
func (s *Store) GetPersisted() *models.PersistedReview {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pr := *s.persisted
	if s.persisted.Diff != nil {
		diffCopy := *s.persisted.Diff
		pr.Diff = &diffCopy
	}
	pr.Comments = make([]models.Comment, len(s.persisted.Comments))
	copy(pr.Comments, s.persisted.Comments)
	return &pr
}

// SwitchFile 切换存储文件并加载已有评论
func (s *Store) SwitchFile(filePath string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.filePath = filePath
	s.persisted = &models.PersistedReview{
		Comments: []models.Comment{},
	}
	if filePath != "" {
		if data, err := os.ReadFile(filePath); err == nil {
			s.loadData(data)
		}
	}
}
