package git

import (
	"strings"
	"testing"

	"github.com/wtf2pr/wtf2pr/pkg/models"
)

func TestParseDiffEmpty(t *testing.T) {
	files, err := ParseDiff("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected 0 files, got %d", len(files))
	}
}

func TestParseDiffNewFile(t *testing.T) {
	raw := `diff --git a/hello.txt b/hello.txt
new file mode 100644
index 0000000..e965047
--- /dev/null
+++ b/hello.txt
@@ -0,0 +1 @@
+Hello world
`
	files, err := ParseDiff(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	f := files[0]
	if !f.IsNew {
		t.Errorf("expected IsNew=true")
	}
	if f.NewFile != "hello.txt" {
		t.Errorf("expected NewFile=hello.txt, got %s", f.NewFile)
	}
	if len(f.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(f.Hunks))
	}
	h := f.Hunks[0]
	if h.NewStart != 1 || h.NewLines != 1 {
		t.Errorf("unexpected hunk range: %d,%d", h.NewStart, h.NewLines)
	}
	if len(h.Lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(h.Lines))
	}
	line := h.Lines[0]
	if line.Type != models.LineTypeAddition {
		t.Errorf("expected addition, got %s", line.Type)
	}
	if line.NewLineNo != 1 {
		t.Errorf("expected NewLineNo=1, got %d", line.NewLineNo)
	}
	if line.Content != "Hello world" {
		t.Errorf("unexpected content: %s", line.Content)
	}
}

func TestParseDiffDeletedFile(t *testing.T) {
	raw := `diff --git a/hello.txt b/hello.txt
deleted file mode 100644
index e965047..0000000
--- a/hello.txt
+++ /dev/null
@@ -1 +0,0 @@
-Hello world
`
	files, err := ParseDiff(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	f := files[0]
	if !f.IsDeleted {
		t.Errorf("expected IsDeleted=true")
	}
	if f.OldFile != "hello.txt" {
		t.Errorf("expected OldFile=hello.txt, got %s", f.OldFile)
	}
	if len(f.Hunks) != 1 || len(f.Hunks[0].Lines) != 1 {
		t.Fatalf("expected 1 hunk with 1 line")
	}
	line := f.Hunks[0].Lines[0]
	if line.Type != models.LineTypeDeletion {
		t.Errorf("expected deletion, got %s", line.Type)
	}
	if line.OldLineNo != 1 {
		t.Errorf("expected OldLineNo=1, got %d", line.OldLineNo)
	}
}

func TestParseDiffModifiedFile(t *testing.T) {
	raw := `diff --git a/hello.txt b/hello.txt
index e965047..d0e6b9c 100644
--- a/hello.txt
+++ b/hello.txt
@@ -1,2 +1,2 @@
 Hello world
-Old line
+New line
`
	files, err := ParseDiff(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	f := files[0]
	if f.IsNew || f.IsDeleted || f.IsBinary {
		t.Errorf("expected normal modified file")
	}
	if len(f.Hunks) != 1 || len(f.Hunks[0].Lines) != 3 {
		t.Fatalf("expected 1 hunk with 3 lines, got %d", len(f.Hunks[0].Lines))
	}
	lines := f.Hunks[0].Lines
	if lines[0].Type != models.LineTypeContext || lines[0].OldLineNo != 1 || lines[0].NewLineNo != 1 {
		t.Errorf("unexpected first line: %+v", lines[0])
	}
	if lines[1].Type != models.LineTypeDeletion || lines[1].OldLineNo != 2 {
		t.Errorf("unexpected second line: %+v", lines[1])
	}
	if lines[2].Type != models.LineTypeAddition || lines[2].NewLineNo != 2 {
		t.Errorf("unexpected third line: %+v", lines[2])
	}
}

func TestParseDiffBinaryFile(t *testing.T) {
	raw := `diff --git a/image.png b/image.png
Binary files differ
`
	files, err := ParseDiff(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if !files[0].IsBinary {
		t.Errorf("expected IsBinary=true")
	}
	if len(files[0].Hunks) != 0 {
		t.Errorf("expected no hunks for binary file")
	}
}

func TestParseDiffMultipleFiles(t *testing.T) {
	raw := `diff --git a/a.txt b/a.txt
index e965047..d0e6b9c 100644
--- a/a.txt
+++ b/a.txt
@@ -1 +1 @@
-old
+new

diff --git a/b.txt b/b.txt
new file mode 100644
index 0000000..e965047
--- /dev/null
+++ b/b.txt
@@ -0,0 +1 @@
+content
`
	files, err := ParseDiff(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0].NewFile != "a.txt" {
		t.Errorf("expected first file a.txt, got %s", files[0].NewFile)
	}
	if files[1].NewFile != "b.txt" {
		t.Errorf("expected second file b.txt, got %s", files[1].NewFile)
	}
}

func TestParseDiffMultipleHunks(t *testing.T) {
	raw := `diff --git a/file.txt b/file.txt
index abc..def 100644
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,3 @@
 line1
-old2
+new2
 line3
@@ -10,3 +10,3 @@
 line10
-old11
+new11
 line12
`
	files, err := ParseDiff(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 || len(files[0].Hunks) != 2 {
		t.Fatalf("expected 1 file with 2 hunks")
	}
}

func TestParseHunkHeaderInvalid(t *testing.T) {
	_, err := parseHunkHeader("@@ invalid @@")
	if err == nil {
		t.Fatalf("expected error for invalid hunk header")
	}
}

func TestParseRange(t *testing.T) {
	start, lines, err := parseRange("5,10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if start != 5 || lines != 10 {
		t.Errorf("expected 5,10 got %d,%d", start, lines)
	}

	start, lines, err = parseRange("7")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if start != 7 || lines != 1 {
		t.Errorf("expected 7,1 got %d,%d", start, lines)
	}
}

func TestParseDiffLine(t *testing.T) {
	oldNo, newNo := 1, 1
	dl := parseDiffLine("+added", &oldNo, &newNo)
	if dl.Type != models.LineTypeAddition || dl.NewLineNo != 1 || dl.Content != "added" {
		t.Errorf("unexpected addition line: %+v", dl)
	}
	if oldNo != 1 || newNo != 2 {
		t.Errorf("unexpected line numbers after addition: %d %d", oldNo, newNo)
	}

	dl = parseDiffLine("-removed", &oldNo, &newNo)
	if dl.Type != models.LineTypeDeletion || dl.OldLineNo != 1 || dl.Content != "removed" {
		t.Errorf("unexpected deletion line: %+v", dl)
	}

	dl = parseDiffLine(" context", &oldNo, &newNo)
	if dl.Type != models.LineTypeContext || dl.OldLineNo != 2 || dl.NewLineNo != 2 || dl.Content != "context" {
		t.Errorf("unexpected context line: %+v", dl)
	}

	oldNo, newNo = 5, 5
	dl = parseDiffLine("\\ No newline at end of file", &oldNo, &newNo)
	if dl.Type != models.LineTypeContext || dl.Content != "\\ No newline at end of file" {
		t.Errorf("unexpected no-newline line: %+v", dl)
	}
	if oldNo != 5 || newNo != 5 {
		t.Errorf("no-newline should not change line numbers: %d %d", oldNo, newNo)
	}
}

func TestParseDiffWithLongContent(t *testing.T) {
	content := strings.Repeat("a", 2000)
	raw := "diff --git a/f.txt b/f.txt\nindex abc..def 100644\n--- a/f.txt\n+++ b/f.txt\n@@ -1 +1 @@\n-" + content + "\n+" + content + "b\n"
	files, err := ParseDiff(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 || len(files[0].Hunks[0].Lines) != 2 {
		t.Fatalf("expected 2 lines")
	}
	if files[0].Hunks[0].Lines[0].Content != content {
		t.Errorf("deletion content mismatch")
	}
	if files[0].Hunks[0].Lines[1].Content != content+"b" {
		t.Errorf("addition content mismatch")
	}
}
