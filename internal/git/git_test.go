package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

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
	run("init", "-b", "master")
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
	dir := initGitRepo(t)
	// 新建未跟踪文件应出现在列表中；已跟踪文件不应出现
	if err := os.WriteFile(filepath.Join(dir, "untracked.txt"), []byte("x\n"), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	files, _, err := getUntrackedFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, f := range files {
		if f == "untracked.txt" {
			found = true
		}
		if f == "a.txt" {
			t.Errorf("tracked file a.txt should not be listed")
		}
	}
	if !found {
		t.Errorf("expected untracked.txt in list, got %v", files)
	}
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

func TestCommitsExistRejectsInvalidRef(t *testing.T) {
	dir := initGitRepo(t)
	if CommitsExist(dir, []string{"--output=/tmp/x"})["--output=/tmp/x"] {
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
	// 构造超过 64KB 管道缓冲的未提交变更（截断后 git 会阻塞在 write，可捕获不排空管道的死锁回归）
	var sb strings.Builder
	for i := 0; i < 100000; i++ {
		sb.WriteString("abcdefghij")
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte(sb.String()+"\n"), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	old := maxGitOutputBytes
	maxGitOutputBytes = 512
	t.Cleanup(func() { maxGitOutputBytes = old })

	type result struct {
		out       []byte
		truncated bool
		err       error
	}
	done := make(chan result, 1)
	go func() {
		out, truncated, err := runGitCapped(dir, "diff", "HEAD")
		done <- result{out, truncated, err}
	}()
	var res result
	select {
	case res = <-done:
	case <-time.After(30 * time.Second):
		t.Fatalf("runGitCapped deadlocked on oversized diff")
	}
	if res.err != nil {
		t.Fatalf("unexpected error: %v", res.err)
	}
	out, truncated := res.out, res.truncated
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

func TestGetUntrackedFilesReal(t *testing.T) {
	dir := initGitRepo(t)
	// 新建未跟踪文件应出现在列表中（含空格与非 ASCII 文件名）
	for _, name := range []string{"hello world.txt", "你好.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x\n"), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}
	files, _, err := getUntrackedFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	set := map[string]bool{}
	for _, f := range files {
		set[f] = true
	}
	if !set["hello world.txt"] {
		t.Errorf("expected spaced filename in untracked list, got %v", files)
	}
	if !set["你好.txt"] {
		t.Errorf("expected non-ASCII filename in untracked list, got %v", files)
	}
}

func TestUntrackedFileCountCap(t *testing.T) {
	dir := initGitRepo(t)
	for i := 0; i < 5; i++ {
		if err := os.WriteFile(filepath.Join(dir, "f"+strconv.Itoa(i)+".txt"), []byte("x\n"), 0644); err != nil {
			t.Fatalf("failed to write: %v", err)
		}
	}
	old := maxUntrackedFiles
	maxUntrackedFiles = 2
	t.Cleanup(func() { maxUntrackedFiles = old })

	resp, err := getWorkingDiff(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Truncated {
		t.Errorf("expected Truncated=true when untracked files exceed cap")
	}
}

func TestTextconvNotExecuted(t *testing.T) {
	dir := initGitRepo(t)
	// 恶意仓库配置：textconv 驱动会在渲染 diff 时执行任意命令
	marker := filepath.Join(t.TempDir(), "pwned.txt")
	cfg := fmt.Sprintf("[diff \"pwn\"]\n\ttextconv = touch %s\n", marker)
	f, err := os.OpenFile(filepath.Join(dir, ".git", "config"), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("failed to open git config: %v", err)
	}
	if _, err := f.WriteString(cfg); err != nil {
		t.Fatalf("failed to write git config: %v", err)
	}
	f.Close()
	if err := os.WriteFile(filepath.Join(dir, ".gitattributes"), []byte("a.txt diff=pwn\n"), 0644); err != nil {
		t.Fatalf("failed to write gitattributes: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("changed\n"), 0644); err != nil {
		t.Fatalf("failed to write a.txt: %v", err)
	}

	if _, err := getWorkingDiff(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Errorf("textconv hook should not be executed")
	}
	if _, err := getCommitDiff(dir, "HEAD"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Errorf("textconv hook should not be executed on commit diff")
	}
}

func TestParseDiffSpacedFilename(t *testing.T) {
	raw := `diff --git a/hello world.txt b/hello world.txt
index 257cc56..5716ca5 100644
--- a/hello world.txt
+++ b/hello world.txt
@@ -1 +1 @@
-foo
+bar
`
	files, err := ParseDiff(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].NewFile != "hello world.txt" || files[0].OldFile != "hello world.txt" {
		t.Errorf("spaced filename parsed wrong: old=%q new=%q", files[0].OldFile, files[0].NewFile)
	}
}

func TestGpgProgramNotExecuted(t *testing.T) {
	dir := initGitRepo(t)
	marker := filepath.Join(t.TempDir(), "gpg-pwned.txt")
	// 恶意仓库配置：log.showSignature 触发 gpg.program 执行；用脚本形式（touch 形式会被 git 追加的参数打断）
	prog := filepath.Join(dir, "evil-gpg.sh")
	if err := os.WriteFile(prog, []byte("#!/bin/sh\ntouch "+marker+"\n"), 0755); err != nil {
		t.Fatalf("failed to write evil program: %v", err)
	}
	cfg := "[log]\n\tshowSignature = true\n[gpg]\n\tprogram = " + prog + "\n"
	f, err := os.OpenFile(filepath.Join(dir, ".git", "config"), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("failed to open git config: %v", err)
	}
	if _, err := f.WriteString(cfg); err != nil {
		t.Fatalf("failed to write git config: %v", err)
	}
	f.Close()

	// 伪造一个带 gpgsig 头的 commit 对象并指向 HEAD（未签名 commit 从不触发 gpg 调用）；
	// 直接取现有 commit 原文注入签名头，避免手工拼装 author 格式
	raw := string(mustGit(t, dir, "cat-file", "commit", "HEAD"))
	sigBlock := "gpgsig -----BEGIN PGP SIGNATURE-----\n fake\n -----END PGP SIGNATURE-----\n"
	idx := strings.Index(raw, "\n\n")
	if idx < 0 {
		t.Fatalf("unexpected commit object format")
	}
	commitContent := raw[:idx+1] + sigBlock + raw[idx+1:]
	signedHash := strings.TrimSpace(string(mustGitStdin(t, dir, commitContent, "hash-object", "-t", "commit", "-w", "--stdin")))
	mustGit(t, dir, "update-ref", "refs/heads/master", signedHash)

	// 阳性对照：不带防护参数时恶意配置确实会触发（证明测试配置是活的）
	direct := exec.Command("git", "-c", "log.showSignature=true", "log", "-1")
	direct.Dir = dir
	direct.Env = gitEnv()
	_ = direct.Run()
	if _, err := os.Stat(marker); err != nil {
		t.Skipf("positive control failed: malicious config did not trigger marker: %v", err)
	}
	_ = os.Remove(marker)

	// 经过本工具的调用必须不触发
	if _, err := GetCommits(dir, models.CommitListRequest{Page: 1, PageSize: 10}); err != nil {
		t.Fatalf("GetCommits failed: %v", err)
	}
	if _, err := getCommitDiff(dir, "HEAD"); err != nil {
		t.Fatalf("getCommitDiff failed: %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Errorf("gpg.program hook should not be executed")
	}
}

// mustGit 执行 git 命令并返回 stdout
func mustGit(t *testing.T, dir string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
	return out
}

// mustGitStdin 带 stdin 执行 git 命令并返回 stdout
func mustGitStdin(t *testing.T, dir, input string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
	return out
}

func TestParseDiffRenameSpacedFilename(t *testing.T) {
	// 纯重命名无 --- / +++ 行，文件名应从 rename from/to 行获取
	raw := `diff --git a/hello world.txt b/bye world.txt
similarity index 100%
rename from hello world.txt
rename to bye world.txt
`
	files, err := ParseDiff(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].OldFile != "hello world.txt" || files[0].NewFile != "bye world.txt" {
		t.Errorf("rename with spaces parsed wrong: old=%q new=%q", files[0].OldFile, files[0].NewFile)
	}
}

func TestMergeCommitDiff(t *testing.T) {
	dir := initGitRepo(t)
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	// 构造一个 merge commit
	run("checkout", "-b", "feature")
	if err := os.WriteFile(filepath.Join(dir, "feat.txt"), []byte("feat\n"), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	run("add", ".")
	run("commit", "-m", "feature commit")
	run("checkout", "master")
	run("merge", "--no-ff", "-m", "merge feature", "feature")

	resp, err := getCommitDiff(dir, "HEAD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Files) == 0 {
		t.Errorf("merge commit diff should not be empty (expected -m --first-parent diff)")
	}
}

func TestEmptyRepo(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	run("init", "-b", "master")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test User")
	// 空仓库（零 commit）写入未跟踪文件
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// working diff 应返回 untracked 部分而非 500
	resp, err := getWorkingDiff(dir)
	if err != nil {
		t.Fatalf("expected working diff to work on empty repo, got %v", err)
	}
	if len(resp.Files) != 1 || resp.Files[0].NewFile != "new.txt" {
		t.Errorf("expected untracked file in empty repo diff, got %+v", resp.Files)
	}

	// commits 列表应返回空而非 500
	list, err := GetCommits(dir, models.CommitListRequest{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("expected GetCommits to work on empty repo, got %v", err)
	}
	if list.Total != 0 || len(list.List) != 0 {
		t.Errorf("expected empty commit list, got %+v", list)
	}
}
