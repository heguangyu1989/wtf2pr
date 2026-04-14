package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/wtf2pr/wtf2pr/pkg/models"
)

// GetDiff 获取 git diff 数据
func GetDiff(workDir string, req models.DiffRequest) (*models.DiffResponse, error) {
	if req.Type == models.DiffTypeWorking {
		return getWorkingDiff(workDir)
	}
	return getCommitDiff(workDir, req.Commit)
}

func getWorkingDiff(workDir string) (*models.DiffResponse, error) {
	// 1. 获取所有已跟踪文件的未提交变更（包含 staged + unstaged）
	cmd := exec.Command("git", "diff", "HEAD")
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}
	files, err := ParseDiff(string(out))
	if err != nil {
		return nil, err
	}

	// 2. 补充尚未被 git 跟踪的 untracked 文件
	untracked, _ := getUntrackedFiles(workDir)
	for _, f := range untracked {
		fd, err := readUntrackedFileAsDiff(workDir, f)
		if err == nil {
			files = append(files, fd)
		}
	}

	return &models.DiffResponse{
		Type:  models.DiffTypeWorking,
		Files: files,
	}, nil
}

func getCommitDiff(workDir, commit string) (*models.DiffResponse, error) {
	if commit == "" {
		return nil, fmt.Errorf("commit hash is required")
	}
	// 获取 commit 信息
	infoCmd := exec.Command("git", "show", "-s", "--format=%H%n%an%n%ad%n%s", commit)
	infoCmd.Dir = workDir
	infoOut, err := infoCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git show info failed: %w", err)
	}
	infoLines := strings.Split(strings.TrimSpace(string(infoOut)), "\n")
	commitInfo := &models.CommitInfo{}
	if len(infoLines) >= 4 {
		commitInfo.Hash = infoLines[0]
		commitInfo.Author = infoLines[1]
		commitInfo.Date = infoLines[2]
		commitInfo.Message = strings.Join(infoLines[3:], "\n")
	}

	// 获取 diff
	diffCmd := exec.Command("git", "show", commit, "--format=", "-p")
	diffCmd.Dir = workDir
	diffOut, err := diffCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git show diff failed: %w", err)
	}
	files, err := ParseDiff(string(diffOut))
	if err != nil {
		return nil, err
	}
	return &models.DiffResponse{
		Type:       models.DiffTypeCommit,
		Commit:     commit,
		CommitInfo: commitInfo,
		Files:      files,
	}, nil
}

func getUntrackedFiles(workDir string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return nil, nil
	}
	return strings.Split(s, "\n"), nil
}

func readUntrackedFileAsDiff(workDir, filePath string) (models.FileDiff, error) {
	fullPath := filepath.Join(workDir, filePath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return models.FileDiff{}, err
	}
	if bytes.IndexByte(data, 0) != -1 {
		return models.FileDiff{
			OldFile:  "",
			NewFile:  filePath,
			IsNew:    true,
			IsBinary: true,
		}, nil
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	diffLines := make([]models.DiffLine, len(lines))
	for i, line := range lines {
		diffLines[i] = models.DiffLine{
			Type:      models.LineTypeAddition,
			NewLineNo: i + 1,
			Content:   line,
		}
	}
	return models.FileDiff{
		OldFile: "",
		NewFile: filePath,
		IsNew:   true,
		Hunks: []models.Hunk{
			{
				OldStart: 0,
				OldLines: 0,
				NewStart: 1,
				NewLines: len(lines),
				Lines:    diffLines,
			},
		},
	}, nil
}
