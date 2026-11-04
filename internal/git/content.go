package git

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/wtf2pr/wtf2pr/pkg/models"
)

var ErrInvalidFileContent = errors.New("invalid file content request")

func GetFileContent(workDir string, req models.FileContentRequest) (*models.FileContentResponse, error) {
	cleanPath, err := cleanDiffPath(req.Path)
	if err != nil {
		return nil, err
	}
	if req.Side != models.FileSideOld && req.Side != models.FileSideNew {
		return nil, invalidFileContent("side must be old or new")
	}
	diffReq := models.DiffRequest{Type: req.Type, Commit: req.Commit, Base: req.Base}
	resolvedCommit := ""
	switch req.Type {
	case models.DiffTypeWorking:
	case models.DiffTypeCommit:
		resolvedCommit, err = ResolveCommit(workDir, req.Commit)
		if err != nil {
			return nil, err
		}
		diffReq.Commit = resolvedCommit
	case models.DiffTypeBranch:
		if err := validateLocalBranch(workDir, req.Base); err != nil {
			return nil, err
		}
	default:
		return nil, invalidFileContent("unsupported diff type %q", req.Type)
	}
	diff, err := GetDiff(workDir, diffReq)
	if err != nil {
		return nil, err
	}

	var selectedFile *models.FileDiff
	matched := false
	missingSide := false
	for i := range diff.Files {
		file := &diff.Files[i]
		if file.OldFile != cleanPath && file.NewFile != cleanPath {
			continue
		}
		matched = true
		if (req.Side == models.FileSideOld && file.IsNew) || (req.Side == models.FileSideNew && file.IsDeleted) {
			missingSide = true
		}
		sidePath := file.NewFile
		if req.Side == models.FileSideOld {
			sidePath = file.OldFile
		}
		if ((req.Side == models.FileSideOld && !file.IsNew) || (req.Side == models.FileSideNew && !file.IsDeleted)) && sidePath == cleanPath {
			selectedFile = file
			break
		}
	}
	if !matched {
		return nil, invalidFileContent("path is not part of the requested diff")
	}
	if selectedFile == nil {
		if missingSide {
			return &models.FileContentResponse{}, nil
		}
		return nil, invalidFileContent("path does not match the requested diff side")
	}
	file := selectedFile
	response := &models.FileContentResponse{}
	selectedPath := file.NewFile
	if req.Side == models.FileSideOld {
		selectedPath = file.OldFile
	}
	if selectedPath != cleanPath {
		return nil, invalidFileContent("path does not match the requested diff side")
	}
	response.Exists = true
	var data []byte
	if req.Type == models.DiffTypeWorking && req.Side == models.FileSideNew {
		data, err = readWorkingFile(workDir, selectedPath)
	} else {
		revision, revErr := contentRevision(req.Type, req.Side, resolvedCommit, diff.Comparison)
		if revErr != nil {
			return nil, revErr
		}
		data, err = readGitFile(workDir, revision, selectedPath)
	}
	if err != nil {
		return nil, err
	}
	if bytes.IndexByte(data, 0) >= 0 {
		response.Content = ""
		response.IsBinary = true
		return response, nil
	}
	response.Content = string(data)
	return response, nil
}

func cleanDiffPath(filePath string) (string, error) {
	if filePath == "" || strings.ContainsRune(filePath, 0) || path.IsAbs(filePath) || filepath.IsAbs(filepath.FromSlash(filePath)) {
		return "", invalidFileContent("invalid file path")
	}
	clean := path.Clean(filePath)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != filePath {
		return "", invalidFileContent("invalid file path")
	}
	return clean, nil
}

func invalidFileContent(format string, args ...interface{}) error {
	return fmt.Errorf("%w: %s", ErrInvalidFileContent, fmt.Sprintf(format, args...))
}

// ResolveCommit 将符号 revision 解析为不可变的完整 Commit 哈希
func ResolveCommit(workDir, commit string) (string, error) {
	if strings.TrimSpace(commit) == "" {
		return "", invalidFileContent("commit is required")
	}
	cmd := exec.Command("git", "rev-parse", "--verify", "--end-of-options", commit+"^{commit}")
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		message := string(out)
		if strings.Contains(message, "unknown revision") || strings.Contains(message, "bad revision") || strings.Contains(message, "Needed a single revision") {
			return "", invalidFileContent("unknown commit %q", commit)
		}
		return "", fmt.Errorf("git rev-parse failed: %w: %s", err, message)
	}
	return strings.TrimSpace(string(out)), nil
}

func contentRevision(kind models.DiffType, side models.FileSide, commit string, comparison *models.ComparisonInfo) (string, error) {
	switch kind {
	case models.DiffTypeWorking:
		return "HEAD", nil
	case models.DiffTypeCommit:
		if side == models.FileSideOld {
			return commit + "^", nil
		}
		return commit, nil
	case models.DiffTypeBranch:
		if comparison == nil {
			return "", invalidFileContent("branch comparison is missing")
		}
		if side == models.FileSideOld {
			return comparison.MergeBase, nil
		}
		return "HEAD", nil
	default:
		return "", invalidFileContent("unsupported diff type %q", kind)
	}
}

func readWorkingFile(workDir, filePath string) ([]byte, error) {
	fullPath := filepath.Join(workDir, filePath)
	info, err := os.Lstat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("inspect working tree file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(fullPath)
		if err != nil {
			return nil, fmt.Errorf("read working tree symlink: %w", err)
		}
		return []byte(target), nil
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("unsupported working tree file type %s", info.Mode())
	}
	root, err := os.OpenRoot(workDir)
	if err != nil {
		return nil, fmt.Errorf("open repository root: %w", err)
	}
	defer root.Close()
	file, err := root.Open(filepath.FromSlash(filePath))
	if err != nil {
		return nil, fmt.Errorf("open working tree file: %w", err)
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("inspect open working tree file: %w", err)
	}
	if !openedInfo.Mode().IsRegular() || !os.SameFile(info, openedInfo) {
		return nil, fmt.Errorf("working tree file changed while opening")
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read working tree file: %w", err)
	}
	return data, nil
}

func readGitFile(workDir, revision, filePath string) ([]byte, error) {
	if revision == "" {
		return nil, invalidFileContent("revision is required")
	}
	cmd := exec.Command("git", "show", revision+":"+filePath, "--")
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("read %s:%s: %w", revision, filePath, err)
	}
	return out, nil
}
