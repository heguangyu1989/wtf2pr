package git

import (
	"testing"

	"github.com/wtf2pr/wtf2pr/pkg/models"
)

func TestApplyRawDiffPathsEmptySides(t *testing.T) {
	files := []models.FileDiff{{IsNew: true}, {IsDeleted: true}}
	paths := []rawDiffPath{{status: "A", oldPath: "", newPath: "a.txt"}, {status: "D", oldPath: "b.txt", newPath: ""}}
	if err := applyRawDiffPaths(files, paths); err != nil {
		t.Fatal(err)
	}
	if files[0].OldFile != "" || files[0].NewFile != "a.txt" {
		t.Errorf("unexpected added paths: %#v", files[0])
	}
	if files[1].OldFile != "b.txt" || files[1].NewFile != "" {
		t.Errorf("unexpected deleted paths: %#v", files[1])
	}
}

func TestGetCommitDiffHandlesMergeCommit(t *testing.T) {
	repo := t.TempDir()
	runGitForTest(t, repo, "init", "-b", "main")
	runGitForTest(t, repo, "config", "user.email", "test@test.com")
	runGitForTest(t, repo, "config", "user.name", "Test User")
	writeFileForTest(t, repo, "a.txt", "base\n")
	runGitForTest(t, repo, "add", ".")
	runGitForTest(t, repo, "commit", "-m", "base")
	runGitForTest(t, repo, "checkout", "-b", "feature")
	writeFileForTest(t, repo, "feature.txt", "feature\n")
	runGitForTest(t, repo, "add", ".")
	runGitForTest(t, repo, "commit", "-m", "feature")
	runGitForTest(t, repo, "checkout", "main")
	writeFileForTest(t, repo, "main.txt", "main\n")
	runGitForTest(t, repo, "add", ".")
	runGitForTest(t, repo, "commit", "-m", "main")
	runGitForTest(t, repo, "checkout", "feature")
	runGitForTest(t, repo, "merge", "main", "--no-ff", "-m", "merge main")
	if _, err := GetDiff(repo, models.DiffRequest{Type: models.DiffTypeCommit, Commit: "HEAD"}); err != nil {
		t.Fatalf("merge commit diff failed: %v", err)
	}
}

func TestGetFileContentRenameCycleUsesExactSidePath(t *testing.T) {
	repo := t.TempDir()
	runGitForTest(t, repo, "init", "-b", "main")
	runGitForTest(t, repo, "config", "user.email", "test@test.com")
	runGitForTest(t, repo, "config", "user.name", "Test User")
	writeFileForTest(t, repo, "a.txt", "A\n")
	writeFileForTest(t, repo, "b.txt", "B\n")
	runGitForTest(t, repo, "add", ".")
	runGitForTest(t, repo, "commit", "-m", "base")
	runGitForTest(t, repo, "mv", "a.txt", "c.txt")
	runGitForTest(t, repo, "mv", "b.txt", "a.txt")
	runGitForTest(t, repo, "mv", "c.txt", "b.txt")
	runGitForTest(t, repo, "commit", "-m", "swap names")
	oldContent, err := GetFileContent(repo, fileReq(models.DiffTypeCommit, "b.txt", models.FileSideOld))
	if err != nil || oldContent.Content != "B\n" {
		t.Fatalf("expected original B content, got %#v, %v", oldContent, err)
	}
	newContent, err := GetFileContent(repo, fileReq(models.DiffTypeCommit, "b.txt", models.FileSideNew))
	if err != nil || newContent.Content != "A\n" {
		t.Fatalf("expected renamed A content, got %#v, %v", newContent, err)
	}
}
