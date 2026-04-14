package review

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/wtf2pr/wtf2pr/pkg/models"
)

func TestNewStore_MemoryOnly(t *testing.T) {
	s := NewStore("")
	if s == nil {
		t.Fatal("expected non-nil store")
	}
	list := s.Get()
	if len(list) != 0 {
		t.Fatalf("expected empty comments")
	}
}

func TestStoreSaveAndGet(t *testing.T) {
	s := NewStore("")
	comments := []models.Comment{
		{ID: "1", FilePath: "a.go", Content: "comment1"},
		{ID: "2", FilePath: "b.go", Content: "comment2"},
	}
	s.Save(comments)
	got := s.Get()
	if len(got) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(got))
	}
	if got[0].Content != "comment1" || got[1].Content != "comment2" {
		t.Errorf("unexpected comments")
	}

	// Ensure copy is returned (modifying result should not affect store)
	got[0].Content = "modified"
	got2 := s.Get()
	if got2[0].Content != "comment1" {
		t.Errorf("expected original content unchanged")
	}
}

func TestStoreFilePersistence(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "review.json")

	// Create store and save
	s := NewStore(file)
	comments := []models.Comment{
		{ID: "1", FilePath: "a.go", LineKey: "new:1", Content: "hello"},
	}
	s.Save(comments)

	// Verify file exists
	if _, err := os.Stat(file); os.IsNotExist(err) {
		t.Fatalf("expected review file to exist")
	}

	// Load into new store
	s2 := NewStore(file)
	got := s2.Get()
	if len(got) != 1 || got[0].Content != "hello" {
		t.Errorf("failed to load persisted comments")
	}
}

func TestStoreClear(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "review.json")
	s := NewStore(file)
	s.Save([]models.Comment{{ID: "1", Content: "c1"}})
	s.Clear()
	got := s.Get()
	if len(got) != 0 {
		t.Errorf("expected empty after clear")
	}
	// Verify file cleared too
	s2 := NewStore(file)
	if len(s2.Get()) != 0 {
		t.Errorf("expected empty file after clear")
	}
}

func TestStoreSwitchFile(t *testing.T) {
	dir := t.TempDir()
	file1 := filepath.Join(dir, "review_a.json")
	file2 := filepath.Join(dir, "review_b.json")

	// Pre-create file2 with data
	if err := os.WriteFile(file2, []byte(`[{"id":"2","filePath":"b.go","content":"from-file2"}]`), 0644); err != nil {
		t.Fatalf("failed to create file2: %v", err)
	}

	s := NewStore(file1)
	s.Save([]models.Comment{{ID: "1", Content: "a"}})

	// Switch to file2 should load its data
	s.SwitchFile(file2)
	got := s.Get()
	if len(got) != 1 || got[0].Content != "from-file2" {
		t.Errorf("expected file2 data after switch, got %+v", got)
	}

	// Save to new file
	s.Save([]models.Comment{{ID: "2", Content: "b"}})

	// Old file should remain untouched
	s2 := NewStore(file1)
	if len(s2.Get()) != 1 || s2.Get()[0].Content != "a" {
		t.Errorf("old file should keep original data")
	}
}

func TestStoreConcurrency(t *testing.T) {
	s := NewStore("")
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			s.Save([]models.Comment{{ID: string(rune('a' + idx)), Content: "test"}})
			s.Get()
		}(i)
	}
	wg.Wait()
	// Just ensure no panic
}
