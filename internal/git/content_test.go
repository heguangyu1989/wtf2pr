package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wtf2pr/wtf2pr/pkg/models"
)

func TestGetFileContentSides(t *testing.T) {
	repo := newContentTestRepo(t)
	tests := []struct {
		name   string
		req    models.FileContentRequest
		want   string
		exists bool
	}{
		{name: "working old", req: fileReq(models.DiffTypeWorking, "a.txt", models.FileSideOld), want: "commit after\n", exists: true},
		{name: "working new", req: fileReq(models.DiffTypeWorking, "a.txt", models.FileSideNew), want: "after\n", exists: true},
		{name: "untracked old", req: fileReq(models.DiffTypeWorking, "new.txt", models.FileSideOld), exists: false},
		{name: "untracked new", req: fileReq(models.DiffTypeWorking, "new.txt", models.FileSideNew), want: "untracked\n", exists: true},
		{name: "commit modified old", req: fileReq(models.DiffTypeCommit, "a.txt", models.FileSideOld), want: "before\n", exists: true},
		{name: "commit modified new", req: fileReq(models.DiffTypeCommit, "a.txt", models.FileSideNew), want: "commit after\n", exists: true},
		{name: "commit renamed old", req: fileReq(models.DiffTypeCommit, "old-name.txt", models.FileSideOld), want: "rename me\n", exists: true},
		{name: "commit renamed new", req: fileReq(models.DiffTypeCommit, "new-name.txt", models.FileSideNew), want: "rename me\n", exists: true},
		{name: "commit added old", req: fileReq(models.DiffTypeCommit, "commit-added.txt", models.FileSideOld), exists: false},
		{name: "commit deleted new", req: fileReq(models.DiffTypeCommit, "deleted.txt", models.FileSideNew), exists: false},
		{name: "branch old", req: fileReq(models.DiffTypeBranch, "a.txt", models.FileSideOld), want: "before\n", exists: true},
		{name: "branch new", req: fileReq(models.DiffTypeBranch, "a.txt", models.FileSideNew), want: "commit after\n", exists: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetFileContent(repo, tt.req)
			if err != nil {
				t.Fatal(err)
			}
			if got.Content != tt.want || got.Exists != tt.exists || got.IsBinary {
				t.Fatalf("got %#v", got)
			}
		})
	}
}

func TestGetFileContentEmptyAndBinary(t *testing.T) {
	repo := newContentTestRepo(t)
	empty, err := GetFileContent(repo, fileReq(models.DiffTypeCommit, "empty.txt", models.FileSideNew))
	if err != nil {
		t.Fatal(err)
	}
	if !empty.Exists || empty.Content != "" || empty.IsBinary {
		t.Fatalf("unexpected empty file: %#v", empty)
	}
	binary, err := GetFileContent(repo, fileReq(models.DiffTypeCommit, "binary.bin", models.FileSideNew))
	if err != nil {
		t.Fatal(err)
	}
	if !binary.Exists || !binary.IsBinary || binary.Content != "" {
		t.Fatalf("unexpected binary file: %#v", binary)
	}
}

func TestGetFileContentRejectsUnsafeOrUnrelatedPaths(t *testing.T) {
	repo := newContentTestRepo(t)
	for _, p := range []string{"", "../outside.txt", "/etc/passwd", "unmodified.txt", ".git/config", "a/../a.txt"} {
		if _, err := GetFileContent(repo, fileReq(models.DiffTypeWorking, p, models.FileSideNew)); err == nil {
			t.Errorf("expected %q to be rejected", p)
		}
	}
}

func TestSymlinksReturnLinkTextWithoutReadingTargets(t *testing.T) {
	repo := newContentTestRepo(t)
	target, err := os.Readlink(filepath.Join(repo, "linked.txt"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := GetFileContent(repo, fileReq(models.DiffTypeWorking, "linked.txt", models.FileSideNew))
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != target {
		t.Fatalf("expected link text %q, got %q", target, got.Content)
	}

	diff, err := GetDiff(repo, models.DiffRequest{Type: models.DiffTypeWorking})
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range diff.Files {
		if file.NewFile == "linked.txt" && file.Hunks[0].Lines[0].Content != target {
			t.Fatalf("diff followed external symlink: %#v", file.Hunks[0].Lines[0])
		}
	}

	internal, err := GetFileContent(repo, fileReq(models.DiffTypeWorking, "internal-link", models.FileSideNew))
	if err != nil {
		t.Fatal(err)
	}
	if internal.Content != ".git/config" {
		t.Fatalf("expected internal link text, got %q", internal.Content)
	}
}

func TestGetFileContentRejectsInvalidRequests(t *testing.T) {
	repo := newContentTestRepo(t)
	requests := []models.FileContentRequest{
		fileReq(models.DiffTypeWorking, "a.txt", models.FileSide("invalid")),
		fileReq(models.DiffTypeCommit, "old-name.txt", models.FileSideNew),
		fileReq(models.DiffType("invalid"), "a.txt", models.FileSideNew),
	}
	unknownCommit := fileReq(models.DiffTypeCommit, "a.txt", models.FileSideNew)
	unknownCommit.Commit = "does-not-exist"
	requests = append(requests, unknownCommit)
	unknownBase := fileReq(models.DiffTypeBranch, "a.txt", models.FileSideNew)
	unknownBase.Base = "does-not-exist"
	requests = append(requests, unknownBase)
	for _, req := range requests {
		if _, err := GetFileContent(repo, req); err == nil {
			t.Errorf("expected request %#v to fail", req)
		}
	}
}

func newContentTestRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGitForTest(t, repo, "init", "-b", "main")
	runGitForTest(t, repo, "config", "user.email", "test@test.com")
	runGitForTest(t, repo, "config", "user.name", "Test User")
	writeFileForTest(t, repo, "a.txt", "before\n")
	writeFileForTest(t, repo, "old-name.txt", "rename me\n")
	writeFileForTest(t, repo, "deleted.txt", "delete me\n")
	writeFileForTest(t, repo, "empty.txt", "not empty\n")
	writeFileForTest(t, repo, "unmodified.txt", "same\n")
	if err := os.WriteFile(filepath.Join(repo, "binary.bin"), []byte{0, 1}, 0644); err != nil {
		t.Fatal(err)
	}
	runGitForTest(t, repo, "add", ".")
	runGitForTest(t, repo, "commit", "-m", "base")
	runGitForTest(t, repo, "checkout", "-b", "feature")
	writeFileForTest(t, repo, "a.txt", "commit after\n")
	writeFileForTest(t, repo, "commit-added.txt", "added\n")
	writeFileForTest(t, repo, "empty.txt", "")
	if err := os.WriteFile(filepath.Join(repo, "binary.bin"), []byte{0, 2}, 0644); err != nil {
		t.Fatal(err)
	}
	runGitForTest(t, repo, "mv", "old-name.txt", "new-name.txt")
	if err := os.Remove(filepath.Join(repo, "deleted.txt")); err != nil {
		t.Fatal(err)
	}
	runGitForTest(t, repo, "add", ".")
	runGitForTest(t, repo, "commit", "-m", "feature changes")
	runGitForTest(t, repo, "checkout", "main")
	writeFileForTest(t, repo, "a.txt", "main latest\n")
	writeFileForTest(t, repo, "main-only.txt", "main\n")
	runGitForTest(t, repo, "add", ".")
	runGitForTest(t, repo, "commit", "-m", "main only")
	runGitForTest(t, repo, "checkout", "feature")
	writeFileForTest(t, repo, "a.txt", "after\n")
	writeFileForTest(t, repo, "new.txt", "untracked\n")
	outsideDir := t.TempDir()
	outsidePath := filepath.Join(outsideDir, "outside.txt")
	if err := os.WriteFile(outsidePath, []byte("outside secret\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsidePath, filepath.Join(repo, "linked.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(".git/config", filepath.Join(repo, "internal-link")); err != nil {
		t.Fatal(err)
	}
	return repo
}

func fileReq(kind models.DiffType, filePath string, side models.FileSide) models.FileContentRequest {
	req := models.FileContentRequest{Type: kind, Path: filePath, Side: side}
	switch kind {
	case models.DiffTypeCommit:
		req.Commit = "HEAD"
	case models.DiffTypeBranch:
		req.Base = "main"
	}
	return req
}

func TestGetFileContentTextBinaryTransitions(t *testing.T) {
	repo := newEdgeCaseRepo(t)
	tests := []struct {
		path, side, want string
		binary           bool
	}{
		{"mixed.txt", "old", "READABLE_OLD_TEXT\n", false},
		{"mixed.txt", "new", "", true},
		{"reverse.bin", "old", "", true},
		{"reverse.bin", "new", "READABLE_NEW_TEXT\n", false},
	}
	for _, tt := range tests {
		got, err := GetFileContent(repo, fileReq(models.DiffTypeCommit, tt.path, models.FileSide(tt.side)))
		if err != nil {
			t.Fatalf("%s %s: %v", tt.path, tt.side, err)
		}
		if got.Content != tt.want || got.IsBinary != tt.binary {
			t.Errorf("%s %s: got %#v", tt.path, tt.side, got)
		}
	}
}

func TestGetFileContentSpaceAndUnicodePaths(t *testing.T) {
	repo := newEdgeCaseRepo(t)
	if _, err := GetFileContent(repo, fileReq(models.DiffTypeCommit, "secret.txt", models.FileSideNew)); err == nil {
		t.Fatal("space path confusion allowed an unchanged secret file")
	}
	for _, filePath := range []string{"secret.txt harmless.txt", "src/文档.js"} {
		got, err := GetFileContent(repo, fileReq(models.DiffTypeCommit, filePath, models.FileSideNew))
		if err != nil {
			t.Fatalf("%s: %v", filePath, err)
		}
		if got.Content != "after\n" {
			t.Errorf("%s: got %q", filePath, got.Content)
		}
	}
}

func TestGetFileContentBinaryAndWhitespaceAliases(t *testing.T) {
	repo := newEdgeCaseRepo(t)
	for _, filePath := range []string{"x and b/secret.txt", "quote\".bin", "src/文档.bin"} {
		got, err := GetFileContent(repo, fileReq(models.DiffTypeCommit, filePath, models.FileSideNew))
		if err != nil {
			t.Fatalf("%s: %v", filePath, err)
		}
		if !got.IsBinary || got.Content != "" {
			t.Errorf("%s: got %#v", filePath, got)
		}
	}
	got, err := GetFileContent(repo, fileReq(models.DiffTypeCommit, "trailing ", models.FileSideNew))
	if err != nil || got.Content != "after\n" {
		t.Fatalf("trailing-space path failed: %#v, %v", got, err)
	}
	if _, err := GetFileContent(repo, fileReq(models.DiffTypeCommit, "trailing", models.FileSideNew)); err == nil {
		t.Fatal("trailing-space alias allowed an unchanged file")
	}
}

func TestResolveCommitValidatesAndNormalizesHash(t *testing.T) {
	repo := newEdgeCaseRepo(t)
	if _, err := ResolveCommit(repo, strings.Repeat("0", 40)); err == nil {
		t.Fatal("expected nonexistent 40-character hash to be rejected")
	}
	fullHash := runGitForTest(t, repo, "rev-parse", "HEAD")
	got, err := ResolveCommit(repo, strings.ToUpper(fullHash))
	if err != nil {
		t.Fatal(err)
	}
	if got != fullHash {
		t.Fatalf("expected normalized hash %q, got %q", fullHash, got)
	}
}

func TestRawPathsAndAliasesAcrossSources(t *testing.T) {
	repo := newEdgeCaseRepo(t)
	writeFileForTest(t, repo, "trailing ", "working after\n")
	if err := os.WriteFile(filepath.Join(repo, "x and b/secret.txt"), []byte{0, 3}, 0644); err != nil {
		t.Fatal(err)
	}
	for _, source := range []models.DiffType{models.DiffTypeWorking, models.DiffTypeCommit, models.DiffTypeBranch} {
		req := fileReq(source, "trailing ", models.FileSideNew)
		diff, err := GetDiff(repo, models.DiffRequest{Type: req.Type, Commit: req.Commit, Base: req.Base})
		if err != nil {
			t.Fatalf("%s: %v", source, err)
		}
		found := false
		for _, file := range diff.Files {
			if file.NewFile == "x and b/secret.txt" {
				found = true
			}
			if file.NewFile == "secret.txt" || file.NewFile == "trailing" {
				t.Fatalf("%s exposed an unchanged alias: %#v", source, file)
			}
		}
		if !found {
			t.Errorf("%s lost binary path with spaces", source)
		}
		if _, err := GetFileContent(repo, req); err != nil {
			t.Errorf("%s trailing-space path failed: %v", source, err)
		}
		alias := fileReq(source, "trailing", models.FileSideNew)
		if _, err := GetFileContent(repo, alias); err == nil {
			t.Errorf("%s allowed unchanged alias", source)
		}
	}
}

func TestGetFileContentWorkingDeletedAndRecreatedPath(t *testing.T) {
	repo := t.TempDir()
	runGitForTest(t, repo, "init", "-b", "main")
	runGitForTest(t, repo, "config", "user.email", "test@test.com")
	runGitForTest(t, repo, "config", "user.name", "Test User")
	writeFileForTest(t, repo, "a.txt", "old\n")
	runGitForTest(t, repo, "add", ".")
	runGitForTest(t, repo, "commit", "-m", "base")
	runGitForTest(t, repo, "rm", "--cached", "a.txt")
	writeFileForTest(t, repo, "a.txt", "new\n")
	oldContent, err := GetFileContent(repo, fileReq(models.DiffTypeWorking, "a.txt", models.FileSideOld))
	if err != nil || oldContent.Content != "old\n" {
		t.Fatalf("expected deleted old content, got %#v, %v", oldContent, err)
	}
	newContent, err := GetFileContent(repo, fileReq(models.DiffTypeWorking, "a.txt", models.FileSideNew))
	if err != nil || !newContent.Exists || newContent.Content != "new\n" {
		t.Fatalf("expected recreated new content, got %#v, %v", newContent, err)
	}
}

func newEdgeCaseRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGitForTest(t, repo, "init", "-b", "main")
	runGitForTest(t, repo, "config", "user.email", "test@test.com")
	runGitForTest(t, repo, "config", "user.name", "Test User")
	writeFileForTest(t, repo, "secret.txt", "UNMODIFIED_SECRET\n")
	writeFileForTest(t, repo, "trailing", "UNMODIFIED_TRAILING_ALIAS_SECRET\n")
	writeFileForTest(t, repo, "trailing ", "before\n")
	writeFileForTest(t, repo, "secret.txt harmless.txt", "before\n")
	writeFileForTest(t, repo, "src/文档.js", "before\n")
	writeFileForTest(t, repo, "mixed.txt", "READABLE_OLD_TEXT\n")
	for _, filePath := range []string{"reverse.bin", "x and b/secret.txt", "quote\".bin", "src/文档.bin"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(repo, filePath)), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, filePath), []byte{0, 1}, 0644); err != nil {
			t.Fatal(err)
		}
	}
	runGitForTest(t, repo, "add", ".")
	runGitForTest(t, repo, "commit", "-m", "edge base")
	runGitForTest(t, repo, "checkout", "-b", "feature")
	writeFileForTest(t, repo, "trailing ", "after\n")
	writeFileForTest(t, repo, "secret.txt harmless.txt", "after\n")
	writeFileForTest(t, repo, "src/文档.js", "after\n")
	if err := os.WriteFile(filepath.Join(repo, "mixed.txt"), []byte{0, 2}, 0644); err != nil {
		t.Fatal(err)
	}
	writeFileForTest(t, repo, "reverse.bin", "READABLE_NEW_TEXT\n")
	for _, filePath := range []string{"x and b/secret.txt", "quote\".bin", "src/文档.bin"} {
		if err := os.WriteFile(filepath.Join(repo, filePath), []byte{0, 2}, 0644); err != nil {
			t.Fatal(err)
		}
	}
	runGitForTest(t, repo, "add", ".")
	runGitForTest(t, repo, "commit", "-m", "edge changes")
	return repo
}
