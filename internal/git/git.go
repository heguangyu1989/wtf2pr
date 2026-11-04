package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/wtf2pr/wtf2pr/pkg/models"
)

// GetDiff 获取 git diff 数据
func GetDiff(workDir string, req models.DiffRequest) (*models.DiffResponse, error) {
	switch req.Type {
	case models.DiffTypeWorking:
		return getWorkingDiff(workDir)
	case models.DiffTypeCommit:
		return getCommitDiff(workDir, req.Commit)
	case models.DiffTypeBranch:
		return getBranchDiff(workDir, req.Base)
	default:
		return nil, fmt.Errorf("unsupported diff type %q", req.Type)
	}
}

func getBranchDiff(workDir, base string) (*models.DiffResponse, error) {
	base = strings.TrimSpace(base)
	if err := validateLocalBranch(workDir, base); err != nil {
		return nil, err
	}
	baseRef := "refs/heads/" + base
	mergeCmd := exec.Command("git", "merge-base", baseRef, "HEAD")
	mergeCmd.Dir = workDir
	mergeOut, err := mergeCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git merge-base failed: %w", err)
	}
	mergeBase := strings.TrimSpace(string(mergeOut))
	diffCmd := exec.Command("git", "diff", "--no-ext-diff", "-M", baseRef+"...HEAD", "--")
	diffCmd.Dir = workDir
	diffOut, err := diffCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git branch diff failed: %w", err)
	}
	files, err := ParseDiff(string(diffOut))
	if err != nil {
		return nil, err
	}
	paths, err := getRawDiffPaths(workDir, models.DiffRequest{Type: models.DiffTypeBranch, Base: base})
	if err != nil {
		return nil, err
	}
	if err := applyRawDiffPaths(files, paths); err != nil {
		return nil, err
	}
	return &models.DiffResponse{
		Type: models.DiffTypeBranch,
		Comparison: &models.ComparisonInfo{
			Base:      base,
			Head:      "HEAD",
			MergeBase: mergeBase,
		},
		Files: files,
	}, nil
}

func validateLocalBranch(workDir, base string) error {
	if strings.TrimSpace(base) == "" {
		return invalidFileContent("base branch is required")
	}
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+base)
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return invalidFileContent("unknown local branch %q", base)
	}
	return fmt.Errorf("git show-ref failed: %w: %s", err, out)
}

// GetBranches 获取本地分支与默认对比分支
func GetBranches(workDir string) (*models.BranchListResponse, error) {
	currentCmd := exec.Command("git", "branch", "--show-current")
	currentCmd.Dir = workDir
	currentOut, err := currentCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git current branch failed: %w", err)
	}
	current := strings.TrimSpace(string(currentOut))

	listCmd := exec.Command("git", "for-each-ref", "--format=%(refname:lstrip=2)%00%(HEAD)", "refs/heads")
	listCmd.Dir = workDir
	listOut, err := listCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git branch list failed: %w", err)
	}
	branches := make([]models.BranchInfo, 0)
	for _, line := range strings.Split(strings.TrimSpace(string(listOut)), "\n") {
		parts := strings.Split(line, "\x00")
		if len(parts) != 2 || parts[0] == "" {
			continue
		}
		branches = append(branches, models.BranchInfo{Name: parts[0], Current: parts[1] == "*"})
	}

	defaultBranch := ""
	for _, preferred := range []string{"main", "master"} {
		for _, branch := range branches {
			if branch.Name == preferred && branch.Name != current {
				defaultBranch = branch.Name
				break
			}
		}
		if defaultBranch != "" {
			break
		}
	}
	if defaultBranch == "" {
		for _, branch := range branches {
			if branch.Name != current {
				defaultBranch = branch.Name
				break
			}
		}
	}
	return &models.BranchListResponse{Current: current, Default: defaultBranch, List: branches}, nil
}

func getWorkingDiff(workDir string) (*models.DiffResponse, error) {
	// 1. 获取所有已跟踪文件的未提交变更（包含 staged + unstaged）
	cmd := exec.Command("git", "diff", "-M", "HEAD")
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}
	files, err := ParseDiff(string(out))
	if err != nil {
		return nil, err
	}
	paths, err := getRawDiffPaths(workDir, models.DiffRequest{Type: models.DiffTypeWorking})
	if err != nil {
		return nil, err
	}
	if err := applyRawDiffPaths(files, paths); err != nil {
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
	resolvedCommit, err := ResolveCommit(workDir, commit)
	if err != nil {
		return nil, err
	}
	commit = resolvedCommit
	// 获取 commit 信息
	infoCmd := exec.Command("git", "show", "-s", "--format=%H%n%an%n%ad%n%s", commit, "--")
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
	diffCmd := exec.Command("git", "show", commit, "--format=", "-p", "-M", "--")
	diffCmd.Dir = workDir
	diffOut, err := diffCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git show diff failed: %w", err)
	}
	files, err := ParseDiff(string(diffOut))
	if err != nil {
		return nil, err
	}
	merge, err := isMergeCommit(workDir, commit)
	if err != nil {
		return nil, err
	}
	if !merge {
		paths, err := getRawDiffPaths(workDir, models.DiffRequest{Type: models.DiffTypeCommit, Commit: commit})
		if err != nil {
			return nil, err
		}
		if err := applyRawDiffPaths(files, paths); err != nil {
			return nil, err
		}
	}
	return &models.DiffResponse{
		Type:       models.DiffTypeCommit,
		Commit:     commit,
		CommitInfo: commitInfo,
		Files:      files,
	}, nil
}

func isMergeCommit(workDir, commit string) (bool, error) {
	cmd := exec.Command("git", "rev-list", "--parents", "-n", "1", commit)
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git commit parents failed: %w", err)
	}
	return len(strings.Fields(string(out))) > 2, nil
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
	cmd := exec.Command("git", "ls-files", "--others", "--exclude-standard", "-z")
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	s := strings.TrimSuffix(string(out), "\x00")
	if s == "" {
		return nil, nil
	}
	return strings.Split(s, "\x00"), nil
}

// CommitExists 检查指定的 commit hash 是否存在于当前仓库且类型为 commit
func CommitExists(workDir, hash string) bool {
	_, err := ResolveCommit(workDir, hash)
	return err == nil
}

func readUntrackedFileAsDiff(workDir, filePath string) (models.FileDiff, error) {
	data, err := readWorkingFile(workDir, filePath)
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
