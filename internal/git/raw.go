package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/wtf2pr/wtf2pr/pkg/models"
)

type rawDiffPath struct {
	status  string
	oldPath string
	newPath string
}

func getRawDiffPaths(workDir string, req models.DiffRequest) ([]rawDiffPath, error) {
	var cmd *exec.Cmd
	switch req.Type {
	case models.DiffTypeWorking:
		cmd = exec.Command("git", "diff", "--raw", "-z", "-M", "HEAD", "--")
	case models.DiffTypeCommit:
		parentCmd := exec.Command("git", "rev-parse", "--verify", req.Commit+"^")
		parentCmd.Dir = workDir
		if parentOut, err := parentCmd.Output(); err == nil {
			cmd = exec.Command("git", "diff-tree", "--raw", "-z", "-r", "-M", "--no-commit-id", strings.TrimSpace(string(parentOut)), req.Commit, "--")
		} else {
			cmd = exec.Command("git", "diff-tree", "--raw", "-z", "-r", "-M", "--no-commit-id", "--root", req.Commit, "--")
		}
	case models.DiffTypeBranch:
		cmd = exec.Command("git", "diff", "--raw", "-z", "-M", "refs/heads/"+req.Base+"...HEAD", "--")
	default:
		return nil, fmt.Errorf("unsupported diff type %q", req.Type)
	}
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git raw diff failed: %w", err)
	}
	return parseRawDiffPaths(out)
}

func parseRawDiffPaths(raw []byte) ([]rawDiffPath, error) {
	fields := bytes.Split(raw, []byte{0})
	paths := make([]rawDiffPath, 0)
	for index := 0; index < len(fields); {
		header := string(fields[index])
		index++
		if header == "" {
			continue
		}
		parts := strings.Fields(header)
		if len(parts) < 5 || index >= len(fields) {
			return nil, fmt.Errorf("invalid raw diff entry %q", header)
		}
		status := parts[4]
		entry := rawDiffPath{status: status}
		if strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C") {
			if index+1 >= len(fields) {
				return nil, fmt.Errorf("invalid raw rename entry %q", header)
			}
			entry.oldPath = string(fields[index])
			entry.newPath = string(fields[index+1])
			index += 2
		} else {
			path := string(fields[index])
			entry.oldPath = path
			entry.newPath = path
			if status == "A" {
				entry.oldPath = ""
			}
			if status == "D" {
				entry.newPath = ""
			}
			index++
		}
		paths = append(paths, entry)
	}
	return paths, nil
}

func applyRawDiffPaths(files []models.FileDiff, paths []rawDiffPath) error {
	if len(files) != len(paths) {
		return fmt.Errorf("diff path metadata mismatch: %d files / %d raw entries", len(files), len(paths))
	}
	for index, path := range paths {
		files[index].OldFile = path.oldPath
		files[index].NewFile = path.newPath
		files[index].IsNew = path.status == "A"
		files[index].IsDeleted = path.status == "D"
	}
	return nil
}
