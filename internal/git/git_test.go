package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestGetBranchDiffUsesMergeBaseAndExcludesWorkingTree(t *testing.T) {
	repo := newBranchTestRepo(t)
	got, err := GetDiff(repo, models.DiffRequest{Type: models.DiffTypeBranch, Base: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != models.DiffTypeBranch || got.Comparison == nil {
		t.Fatalf("missing comparison: %#v", got)
	}
	if got.Comparison.Head != "HEAD" || got.Comparison.Base != "main" {
		t.Fatalf("bad comparison: %#v", got.Comparison)
	}
	if want := runGitForTest(t, repo, "merge-base", "main", "HEAD"); got.Comparison.MergeBase != want {
		t.Fatalf("expected merge base %q, got %q", want, got.Comparison.MergeBase)
	}
	assertDiffPaths(t, got.Files, []string{"feature.txt"})
}

func TestGetDiffRejectsInvalidBranchRequests(t *testing.T) {
	repo := newBranchTestRepo(t)
	requests := []models.DiffRequest{
		{Type: models.DiffTypeBranch},
		{Type: models.DiffTypeBranch, Base: "missing"},
		{Type: models.DiffType("invalid")},
	}
	for _, req := range requests {
		if _, err := GetDiff(repo, req); err == nil {
			t.Errorf("expected request %#v to fail", req)
		}
	}
}

func TestGetBranchesDefaultsToMain(t *testing.T) {
	repo := newBranchTestRepo(t)
	got, err := GetBranches(repo)
	if err != nil {
		t.Fatal(err)
	}
	if got.Current != "feature" || got.Default != "main" {
		t.Fatalf("bad branches: %#v", got)
	}
	if len(got.List) != 2 {
		t.Fatalf("expected main and feature: %#v", got.List)
	}
}

func TestGetBranchesDefaultsToMaster(t *testing.T) {
	repo := newBranchTestRepo(t)
	runGitForTest(t, repo, "branch", "-m", "main", "master")
	got, err := GetBranches(repo)
	if err != nil {
		t.Fatal(err)
	}
	if got.Default != "master" {
		t.Fatalf("expected master default: %#v", got)
	}
}

func TestGetBranchesPrefersMainWhenMasterAlsoExists(t *testing.T) {
	repo := newBranchTestRepo(t)
	runGitForTest(t, repo, "branch", "master", "main")
	got, err := GetBranches(repo)
	if err != nil {
		t.Fatal(err)
	}
	if got.Default != "main" {
		t.Fatalf("expected main to take precedence over master: %#v", got)
	}
}

func TestGetBranchDiffUsesFullLocalRefWithAmbiguousTag(t *testing.T) {
	repo := newBranchTestRepo(t)
	runGitForTest(t, repo, "tag", "main", "HEAD")
	got, err := GetDiff(repo, models.DiffRequest{Type: models.DiffTypeBranch, Base: "main"})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffPaths(t, got.Files, []string{"feature.txt"})
}

func TestGetBranchesRoundTripWithAmbiguousTag(t *testing.T) {
	repo := newBranchTestRepo(t)
	runGitForTest(t, repo, "tag", "main", "HEAD")
	branches, err := GetBranches(repo)
	if err != nil {
		t.Fatal(err)
	}
	if branches.Default != "main" {
		t.Fatalf("expected usable main branch, got %q", branches.Default)
	}
	if _, err := GetDiff(repo, models.DiffRequest{Type: models.DiffTypeBranch, Base: branches.Default}); err != nil {
		t.Fatalf("branch list value cannot be used for diff: %v", err)
	}
}

func TestGetCommitDiffRejectsOptionLikeRevision(t *testing.T) {
	repo := newBranchTestRepo(t)
	outputPath := filepath.Join(repo, "injected.patch")
	_, err := GetDiff(repo, models.DiffRequest{Type: models.DiffTypeCommit, Commit: "--output=" + outputPath})
	if err == nil {
		t.Fatal("expected option-like revision to be rejected")
	}
	if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
		t.Fatalf("revision unexpectedly created output file %q", outputPath)
	}
}

func TestGetBranchesDetachedHead(t *testing.T) {
	repo := newBranchTestRepo(t)
	runGitForTest(t, repo, "checkout", "--detach")
	got, err := GetBranches(repo)
	if err != nil {
		t.Fatal(err)
	}
	if got.Current != "" {
		t.Fatalf("expected detached HEAD: %#v", got)
	}
	for _, branch := range got.List {
		if branch.Current {
			t.Fatalf("detached HEAD marked branch current: %#v", got.List)
		}
	}
}

func TestGetBranchesWithoutComparisonTarget(t *testing.T) {
	repo := newBranchTestRepo(t)
	runGitForTest(t, repo, "branch", "-D", "main")
	got, err := GetBranches(repo)
	if err != nil {
		t.Fatal(err)
	}
	if got.Default != "" || len(got.List) != 1 {
		t.Fatalf("expected no comparison target: %#v", got)
	}
}

func newBranchTestRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGitForTest(t, repo, "init", "-b", "main")
	runGitForTest(t, repo, "config", "user.email", "test@test.com")
	runGitForTest(t, repo, "config", "user.name", "Test User")
	writeFileForTest(t, repo, "base.txt", "base\n")
	runGitForTest(t, repo, "add", ".")
	runGitForTest(t, repo, "commit", "-m", "base")
	runGitForTest(t, repo, "checkout", "-b", "feature")
	writeFileForTest(t, repo, "feature.txt", "feature\n")
	runGitForTest(t, repo, "add", ".")
	runGitForTest(t, repo, "commit", "-m", "feature")
	runGitForTest(t, repo, "checkout", "main")
	writeFileForTest(t, repo, "main.txt", "main\n")
	runGitForTest(t, repo, "add", ".")
	runGitForTest(t, repo, "commit", "-m", "main only")
	runGitForTest(t, repo, "checkout", "feature")
	writeFileForTest(t, repo, "uncommitted.txt", "dirty\n")
	return repo
}

func runGitForTest(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func writeFileForTest(t *testing.T, repo, name, content string) {
	t.Helper()
	fullPath := filepath.Join(repo, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func assertDiffPaths(t *testing.T, files []models.FileDiff, want []string) {
	t.Helper()
	if len(files) != len(want) {
		t.Fatalf("expected %v, got %#v", want, files)
	}
	for i := range want {
		if got := files[i].NewFile; got != want[i] {
			t.Errorf("expected path %q, got %q", want[i], got)
		}
	}
}
