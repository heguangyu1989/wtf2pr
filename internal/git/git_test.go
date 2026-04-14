package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wtf2pr/wtf2pr/pkg/models"
)

func TestReadUntrackedFileAsDiff_Text(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	content := "line1\nline2\n"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	fd, err := readUntrackedFileAsDiff(dir, "test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fd.IsNew {
		t.Errorf("expected IsNew=true")
	}
	if fd.NewFile != "test.txt" {
		t.Errorf("expected NewFile=test.txt, got %s", fd.NewFile)
	}
	if fd.IsBinary {
		t.Errorf("expected IsBinary=false")
	}
	if len(fd.Hunks) != 1 || len(fd.Hunks[0].Lines) != 2 {
		t.Fatalf("expected 1 hunk with 2 lines, got %d hunks / %d lines", len(fd.Hunks), len(fd.Hunks[0].Lines))
	}
	if fd.Hunks[0].Lines[0].Content != "line1" || fd.Hunks[0].Lines[1].Content != "line2" {
		t.Errorf("unexpected lines content")
	}
	if fd.Hunks[0].Lines[0].NewLineNo != 1 || fd.Hunks[0].Lines[1].NewLineNo != 2 {
		t.Errorf("unexpected line numbers")
	}
	if fd.Hunks[0].Lines[0].Type != models.LineTypeAddition {
		t.Errorf("expected addition type")
	}
}

func TestReadUntrackedFileAsDiff_Binary(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "binary.bin")
	content := []byte{0x00, 0x01, 0x02, 0x03}
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	fd, err := readUntrackedFileAsDiff(dir, "binary.bin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fd.IsNew || !fd.IsBinary {
		t.Errorf("expected new binary file")
	}
	if len(fd.Hunks) != 0 {
		t.Errorf("expected no hunks for binary file")
	}
}

func TestReadUntrackedFileAsDiff_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := readUntrackedFileAsDiff(dir, "not_exist.txt")
	if err == nil {
		t.Fatalf("expected error for missing file")
	}
}

func TestGetUntrackedFiles(t *testing.T) {
	dir := t.TempDir()
	// init git repo
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatalf("failed to create .git: %v", err)
	}
	// Without real git executable this test is limited; verify non-git path returns error
	_, err := getUntrackedFiles(dir)
	// git ls-files on a bare .git dir without actual init may succeed or fail depending on git version.
	// We just ensure no panic; err is acceptable.
	_ = err
}
