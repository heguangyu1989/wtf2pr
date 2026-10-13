package git

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/wtf2pr/wtf2pr/pkg/models"
)

// maxGitOutputBytes 限制单次 git 命令输出大小，防止巨型 diff 撑爆内存（var 以便测试调小）
var maxGitOutputBytes = int64(20 << 20) // 20MB

// maxUntrackedFileBytes 限制未跟踪文件的读取大小，超限按二进制占位处理（var 以便测试调小）
var maxUntrackedFileBytes = int64(1 << 20) // 1MB

// maxUntrackedFiles 限制 working diff 中包含的未跟踪文件数量，超限截断（var 以便测试调小）
var maxUntrackedFiles = 500

// gitCmd 构造带环境净化的 git 命令：去除可劫持仓库定位或注入配置的 GIT_* 变量
func gitCmd(workDir string, args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Dir = workDir
	cmd.Env = gitEnv()
	return cmd
}

// gitEnv 返回过滤后的进程环境
func gitEnv() []string {
	blocked := map[string]bool{
		"GIT_DIR": true, "GIT_WORK_TREE": true, "GIT_EXTERNAL_DIFF": true,
		"GIT_CONFIG": true, "GIT_CONFIG_PARAMETERS": true, "GIT_CONFIG_COUNT": true,
		// git ≥ 2.32 的全局/系统配置注入向量
		"GIT_CONFIG_GLOBAL": true, "GIT_CONFIG_SYSTEM": true,
		// 仓库定位劫持向量
		"GIT_OBJECT_DIRECTORY": true, "GIT_ALTERNATE_OBJECT_DIRECTORIES": true, "GIT_INDEX_FILE": true,
	}
	env := make([]string, 0, 64)
	for _, kv := range os.Environ() {
		k := kv[:strings.IndexByte(kv, '=')]
		if blocked[k] || strings.HasPrefix(k, "GIT_CONFIG_KEY_") || strings.HasPrefix(k, "GIT_CONFIG_VALUE_") {
			continue
		}
		env = append(env, kv)
	}
	return env
}

// runGitCapped 执行 git 命令并限制输出大小；超限截断到最后一个完整行并返回 truncated=true
func runGitCapped(workDir string, args ...string) ([]byte, bool, error) {
	cmd := gitCmd(workDir, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, false, err
	}
	if err := cmd.Start(); err != nil {
		return nil, false, err
	}
	buf, rerr := io.ReadAll(io.LimitReader(stdout, maxGitOutputBytes+1))
	truncated := int64(len(buf)) > maxGitOutputBytes
	if truncated {
		// 必须继续排空管道：否则剩余输出超过管道缓冲后 git 阻塞在 write，Wait 永不返回
		_, _ = io.Copy(io.Discard, stdout)
	}
	werr := cmd.Wait()
	if truncated {
		// 截断到最后一个完整行，避免半截行进入解析器
		if i := bytes.LastIndexByte(buf[:maxGitOutputBytes], '\n'); i >= 0 {
			buf = buf[:i+1]
		} else {
			buf = buf[:maxGitOutputBytes]
		}
		return buf, true, nil
	}
	if rerr != nil {
		return nil, false, rerr
	}
	if werr != nil {
		return nil, false, fmt.Errorf("git %s failed: %w", strings.Join(args, " "), werr)
	}
	return buf, false, nil
}

// GetDiff 获取 git diff 数据
func GetDiff(workDir string, req models.DiffRequest) (*models.DiffResponse, error) {
	if req.Type == models.DiffTypeWorking {
		return getWorkingDiff(workDir)
	}
	return getCommitDiff(workDir, req.Commit)
}

func getWorkingDiff(workDir string) (*models.DiffResponse, error) {
	// 1. 获取所有已跟踪文件的未提交变更（包含 staged + unstaged）
	// --no-ext-diff/--no-textconv 防止恶意仓库配置注入命令；core.fsmonitor=false 避免 fsmonitor hook
	out, truncated, err := runGitCapped(workDir, "-c", "core.quotePath=false", "-c", "core.fsmonitor=false", "diff", "--no-ext-diff", "--no-textconv", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}
	files, err := ParseDiff(string(out))
	if err != nil {
		return nil, err
	}

	// 2. 补充尚未被 git 跟踪的 untracked 文件（数量与单文件大小均有限制）
	untracked, _ := getUntrackedFiles(workDir)
	for i, f := range untracked {
		if i >= maxUntrackedFiles {
			truncated = true
			break
		}
		fd, err := readUntrackedFileAsDiff(workDir, f)
		if err == nil {
			files = append(files, fd)
		}
	}

	return &models.DiffResponse{
		Type:      models.DiffTypeWorking,
		Files:     files,
		Truncated: truncated,
	}, nil
}

func getCommitDiff(workDir, commit string) (*models.DiffResponse, error) {
	if err := validateCommitRef(commit); err != nil {
		return nil, err
	}
	// 获取 commit 信息（--end-of-options 防止选项注入）
	infoCmd := gitCmd(workDir, "show", "-s", "--format=%H%n%an%n%ad%n%s", "--end-of-options", commit)
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

	// 获取 diff（选项需在 --end-of-options 之前；--no-ext-diff/--no-textconv 防止仓库配置注入命令）
	diffOut, truncated, err := runGitCapped(workDir, "-c", "core.quotePath=false", "show", "--no-ext-diff", "--no-textconv", "--format=", "-p", "--end-of-options", commit)
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
		Truncated:  truncated,
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
	countOut, err := gitCmd(workDir, "rev-list", "--count", "HEAD").Output()
	if err != nil {
		return nil, fmt.Errorf("git rev-list count failed: %w", err)
	}
	total, _ := strconv.Atoi(strings.TrimSpace(string(countOut)))

	// 获取当前页数据
	skip := (req.Page - 1) * req.PageSize
	logOut, err := gitCmd(workDir, "log", "--pretty=format:%H%x00%an%x00%ad%x00%s", "--skip", strconv.Itoa(skip), "-n", strconv.Itoa(req.PageSize)).Output()
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

// getUntrackedFiles 列出未跟踪文件（-z 按 NUL 分割避免空格/非 ASCII 文件名错位；core.quotePath=false 禁止转义）
func getUntrackedFiles(workDir string) ([]string, error) {
	out, err := gitCmd(workDir, "-c", "core.quotePath=false", "-c", "core.fsmonitor=false", "ls-files", "--others", "--exclude-standard", "-z").Output()
	if err != nil {
		return nil, err
	}
	parts := strings.Split(string(out), "\x00")
	files := parts[:0]
	for _, p := range parts {
		if p != "" {
			files = append(files, p)
		}
	}
	return files, nil
}

// CommitsExist 批量检查多个 commit 是否存在（单次 git cat-file --batch-check 调用，避免 N+1 子进程）
func CommitsExist(workDir string, hashes []string) map[string]bool {
	result := make(map[string]bool, len(hashes))
	var valid []string
	for _, h := range hashes {
		h = strings.TrimSpace(h)
		if err := validateCommitRef(h); err != nil {
			result[h] = false
			continue
		}
		valid = append(valid, h)
	}
	if len(valid) == 0 {
		return result
	}
	cmd := gitCmd(workDir, "cat-file", "--batch-check")
	cmd.Stdin = strings.NewReader(strings.Join(valid, "\n") + "\n")
	out, err := cmd.Output()
	if err != nil {
		return result
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for i, h := range valid {
		if i < len(lines) {
			parts := strings.Fields(lines[i])
			result[h] = len(parts) >= 2 && parts[1] == "commit"
		}
	}
	return result
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
	out, err := gitCmd(workDir, "rev-parse", "--is-inside-work-tree").Output()
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
	// 超限文件不读入内存，按二进制占位展示
	if info.Size() > maxUntrackedFileBytes {
		return models.FileDiff{
			OldFile:  "",
			NewFile:  filePath,
			IsNew:    true,
			IsBinary: true,
		}, nil
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
