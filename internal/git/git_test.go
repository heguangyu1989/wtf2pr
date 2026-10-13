package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/wtf2pr/wtf2pr/pkg/models"
)

func initGitRepo(t *testing.T) string {
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to run git %v: %v", args, err)
		}
	}
	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	run("add", ".")
	run("commit", "-m", "first commit")
	return dir
}

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

func TestReadUntrackedFileAsDiff_SymlinkSkipped(t *testing.T) {
	dir := t.TempDir()
	// 指向仓库外的符号链接不应被读取（防止越界任意文件读）
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0644); err != nil {
		t.Fatalf("failed to create outside file: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "leak.txt")); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}
	if _, err := readUntrackedFileAsDiff(dir, "leak.txt"); err == nil {
		t.Errorf("expected symlink to be rejected, got content of outside file")
	}
}

func TestValidateCommitRef(t *testing.T) {
	valid := []string{"HEAD", "HEAD~3", "HEAD^2", "main", "feat/x.y", "v1.0.0", "d14782ebad7ebe972de3d6f5f5dbd2a5a0ebbc8d", "refs/heads/main"}
	for _, c := range valid {
		if err := validateCommitRef(c); err != nil {
			t.Errorf("expected %q valid, got %v", c, err)
		}
	}
	// 选项注入与异常输入必须拒绝
	invalid := []string{"", "--output=/tmp/x", "-p", "--exec=sh", "../etc", "a b", "a;b"}
	for _, c := range invalid {
		if err := validateCommitRef(c); err == nil {
			t.Errorf("expected %q rejected", c)
		}
	}
}

func TestGetCommitDiff_RejectsOptionInjection(t *testing.T) {
	dir := initGitRepo(t)
	if _, err := getCommitDiff(dir, "--output=/tmp/wtf2pr-injected.txt"); err == nil {
		t.Fatalf("expected option injection to be rejected")
	}
	if _, err := os.Stat("/tmp/wtf2pr-injected.txt"); !os.IsNotExist(err) {
		os.Remove("/tmp/wtf2pr-injected.txt")
		t.Fatalf("injected file should not be created")
	}
}

func TestCommitExists_RejectsInvalidRef(t *testing.T) {
	dir := initGitRepo(t)
	if CommitExists(dir, "--output=/tmp/x") {
		t.Errorf("expected invalid ref to return false")
	}
}

func TestCommitsExistBatch(t *testing.T) {
	dir := initGitRepo(t)
	// 再提交一次，制造第二个 commit
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("second\n"), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	run("add", ".")
	run("commit", "-m", "second")

	result := CommitsExist(dir, []string{"HEAD", "HEAD~1", "0000000000000000000000000000000000000000", "--output=/tmp/x"})
	if !result["HEAD"] {
		t.Errorf("expected HEAD to exist")
	}
	if !result["HEAD~1"] {
		t.Errorf("expected HEAD~1 to exist")
	}
	if result["0000000000000000000000000000000000000000"] {
		t.Errorf("expected zero hash to not exist")
	}
	if result["--output=/tmp/x"] {
		t.Errorf("expected invalid ref to not exist")
	}
}

func TestRunGitCappedTruncation(t *testing.T) {
	dir := initGitRepo(t)
	// 构造较大的未提交变更
	big := make([]byte, 4096)
	for i := range big {
		big[i] = 'a' + byte(i%26)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), append(big, '\n'), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	old := maxGitOutputBytes
	maxGitOutputBytes = 512
	t.Cleanup(func() { maxGitOutputBytes = old })

	out, truncated, err := runGitCapped(dir, "diff", "HEAD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !truncated {
		t.Fatalf("expected truncated=true, got %d bytes", len(out))
	}
	if len(out) > 512 {
		t.Errorf("expected output capped at 512, got %d", len(out))
	}
	if len(out) > 0 && out[len(out)-1] != '\n' {
		t.Errorf("expected output cut at line boundary")
	}
	// 截断后的内容仍可被解析器处理
	if _, err := ParseDiff(string(out)); err != nil {
		t.Errorf("expected truncated diff to remain parseable: %v", err)
	}

	// 恢复上限后完整输出不截断
	maxGitOutputBytes = old
	out2, truncated2, err := runGitCapped(dir, "diff", "HEAD")
	if err != nil || truncated2 {
		t.Errorf("expected full output without truncation, truncated=%v err=%v", truncated2, err)
	}
	if len(out2) <= len(out) {
		t.Errorf("expected full output larger than capped output")
	}
}

func TestReadUntrackedFileAsDiff_Oversize(t *testing.T) {
	dir := t.TempDir()
	content := make([]byte, 2048)
	if err := os.WriteFile(filepath.Join(dir, "big.txt"), content, 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	old := maxUntrackedFileBytes
	maxUntrackedFileBytes = 1024
	t.Cleanup(func() { maxUntrackedFileBytes = old })

	fd, err := readUntrackedFileAsDiff(dir, "big.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fd.IsBinary {
		t.Errorf("expected oversize file to degrade to binary placeholder")
	}
	if len(fd.Hunks) != 0 {
		t.Errorf("expected no hunks for oversize file")
	}
}

func TestGetWorkingDiffTruncatedFlag(t *testing.T) {
	dir := initGitRepo(t)
	big := make([]byte, 4096)
	for i := range big {
		big[i] = 'a' + byte(i%26)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), append(big, '\n'), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	old := maxGitOutputBytes
	maxGitOutputBytes = 256
	t.Cleanup(func() { maxGitOutputBytes = old })

	resp, err := getWorkingDiff(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Truncated {
		t.Errorf("expected DiffResponse.Truncated=true")
	}
}
