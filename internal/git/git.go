package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
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
	if err := validateCommitRef(commit); err != nil {
		return nil, err
	}
	// 获取 commit 信息（--end-of-options 防止选项注入）
	infoCmd := exec.Command("git", "show", "-s", "--format=%H%n%an%n%ad%n%s", "--end-of-options", commit)
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

	// 获取 diff（选项需在 --end-of-options 之前）
	diffCmd := exec.Command("git", "show", "--format=", "-p", "--end-of-options", commit)
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

// GetCommits 获取分页的 git log
func GetCommits(workDir string, req models.CommitListRequest) (*models.CommitListResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 10
	}

	// 先获取总数
	countCmd := exec.Command("git", "rev-list", "--count", "HEAD")
	countCmd.Dir = workDir
	countOut, err := countCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git rev-list count failed: %w", err)
	}
	total, _ := strconv.Atoi(strings.TrimSpace(string(countOut)))

	// 获取当前页数据
	skip := (req.Page - 1) * req.PageSize
	logCmd := exec.Command("git", "log", "--pretty=format:%H%x00%an%x00%ad%x00%s", "--skip", strconv.Itoa(skip), "-n", strconv.Itoa(req.PageSize))
	logCmd.Dir = workDir
	logOut, err := logCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	var list []models.CommitLog
	if len(logOut) > 0 {
		entries := strings.Split(strings.TrimSpace(string(logOut)), "\n")
		for _, entry := range entries {
			parts := strings.Split(entry, "\x00")
			if len(parts) >= 4 {
				list = append(list, models.CommitLog{
					Hash:    parts[0],
					Author:  parts[1],
					Date:    parts[2],
					Message: parts[3],
				})
			}
		}
	}

	totalPages := total / req.PageSize
	if total%req.PageSize != 0 {
		totalPages++
	}

	return &models.CommitListResponse{
		List:       list,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
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

// CommitExists 检查指定的 commit hash 是否存在于当前仓库且类型为 commit
func CommitExists(workDir, hash string) bool {
	hash = strings.TrimSpace(hash)
	if err := validateCommitRef(hash); err != nil {
		return false
	}
	cmd := exec.Command("git", "cat-file", "-t", "--end-of-options", hash)
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "commit"
}

// commitRefPattern 合法的 commit 引用（hash/分支/tag/HEAD~3 等 rev 形式），拒绝以 "-" 开头防止选项注入。
// 仅允许 ASCII 常见字符，非 ASCII 分支名（UTF-8）会被拒绝——前端只传 git log 返回的 hex hash，属于有意的收窄
var commitRefPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/^{}~-]{0,255}$`)

// validateCommitRef 校验 commit 引用合法性
func validateCommitRef(commit string) error {
	if commit == "" {
		return fmt.Errorf("commit hash is required")
	}
	if !commitRefPattern.MatchString(commit) {
		return fmt.Errorf("invalid commit ref: %q", commit)
	}
	return nil
}

// IsGitRepo 检查指定目录是否位于一个 git 仓库中
func IsGitRepo(workDir string) bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

// HasGitDir 检查目录下是否存在 .git（目录或文件，覆盖 submodule/worktree）；轻量探测，适合高频调用
func HasGitDir(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

func readUntrackedFileAsDiff(workDir, filePath string) (models.FileDiff, error) {
	fullPath := filepath.Join(workDir, filePath)
	// 只读取普通文件：跳过符号链接等，避免越出仓库边界读取任意文件
	info, err := os.Lstat(fullPath)
	if err != nil {
		return models.FileDiff{}, err
	}
	if !info.Mode().IsRegular() {
		return models.FileDiff{}, fmt.Errorf("not a regular file: %s", filePath)
	}
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
