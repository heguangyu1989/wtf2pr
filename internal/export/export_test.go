package export

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/wtf2pr/wtf2pr/pkg/models"
)

func makeDiff() *models.DiffResponse {
	return &models.DiffResponse{
		Type: models.DiffTypeCommit,
		CommitInfo: &models.CommitInfo{
			Hash:    "abc123",
			Author:  "tester",
			Date:    "Mon Jan 1 00:00:00 2024 +0000",
			Message: "test commit",
		},
		Files: []models.FileDiff{
			{
				NewFile: "main.go",
				Hunks: []models.Hunk{
					{
						OldStart: 1, OldLines: 1, NewStart: 1, NewLines: 2,
						Lines: []models.DiffLine{
							{Type: models.LineTypeContext, OldLineNo: 1, NewLineNo: 1, Content: "package main"},
							{Type: models.LineTypeAddition, NewLineNo: 2, Content: "import \"fmt\""},
						},
					},
				},
			},
			{
				NewFile: "readme.md",
				IsNew:   true,
				Hunks: []models.Hunk{
					{
						OldStart: 0, OldLines: 0, NewStart: 1, NewLines: 1,
						Lines: []models.DiffLine{
							{Type: models.LineTypeAddition, NewLineNo: 1, Content: "# Hello"},
						},
					},
				},
			},
		},
	}
}

func TestExportMarkdown_NoComments(t *testing.T) {
	diff := makeDiff()
	out, err := Export(diff, []models.Comment{}, models.ExportFormatMarkdown)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "# Review Report") {
		t.Errorf("expected markdown header")
	}
	if !strings.Contains(out, "*No review comments.*") {
		t.Errorf("expected no-comments note")
	}
}

func TestExportMarkdown_WithComments(t *testing.T) {
	diff := makeDiff()
	comments := []models.Comment{
		{ID: "1", FilePath: "main.go", LineKey: "new:2", Content: "Add fmt import"},
		{ID: "2", FilePath: "main.go", Content: "Good file"},
	}
	out, err := Export(diff, comments, models.ExportFormatMarkdown)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "## main.go") {
		t.Errorf("expected main.go section")
	}
	if !strings.Contains(out, "Add fmt import") {
		t.Errorf("expected line comment")
	}
	if !strings.Contains(out, "Good file") {
		t.Errorf("expected file comment")
	}
	if !strings.Contains(out, "`import \"fmt\"`") {
		t.Errorf("expected code line")
	}
	// Files without comments should be omitted
	if strings.Contains(out, "readme.md") {
		t.Errorf("expected readme.md to be omitted since it has no comments")
	}
}

func TestExportJSON(t *testing.T) {
	diff := makeDiff()
	comments := []models.Comment{
		{ID: "1", FilePath: "readme.md", LineKey: "new:1", Content: "Title looks good"},
	}
	out, err := Export(diff, comments, models.ExportFormatJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(out), &data); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if data["type"] != "commit" {
		t.Errorf("unexpected type")
	}
	files, ok := data["files"].([]interface{})
	if !ok || len(files) != 1 {
		t.Fatalf("expected 1 file in json")
	}
}

func TestExportXML(t *testing.T) {
	diff := makeDiff()
	comments := []models.Comment{
		{ID: "1", FilePath: "main.go", LineKey: "new:2", Content: "Nice"},
	}
	out, err := Export(diff, comments, models.ExportFormatXML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(out, "<?xml") {
		t.Errorf("expected xml header")
	}
	if !strings.Contains(out, "<Content>Nice</Content>") {
		t.Errorf("expected comment content in xml")
	}
}

func TestExportUnsupportedFormat(t *testing.T) {
	diff := makeDiff()
	_, err := Export(diff, []models.Comment{}, models.ExportFormat("yaml"))
	if err == nil {
		t.Fatalf("expected error for unsupported format")
	}
}

func TestFindCodeContext_NewLine(t *testing.T) {
	file := models.FileDiff{
		NewFile: "a.go",
		Hunks: []models.Hunk{
			{
				OldStart: 1, OldLines: 1, NewStart: 1, NewLines: 2,
				Lines: []models.DiffLine{
					{Type: models.LineTypeContext, OldLineNo: 1, NewLineNo: 1, Content: "ctx"},
					{Type: models.LineTypeAddition, NewLineNo: 2, Content: "added"},
				},
			},
		},
	}
	code, hunk := findCodeContext(file, "new:2")
	if code != "added" {
		t.Errorf("expected code 'added', got %s", code)
	}
	if hunk != "@@ -1,1 +1,2 @@" {
		t.Errorf("unexpected hunk header: %s", hunk)
	}
}

func TestFindCodeContext_OldLine(t *testing.T) {
	file := models.FileDiff{
		OldFile: "a.go",
		Hunks: []models.Hunk{
			{
				OldStart: 1, OldLines: 2, NewStart: 1, NewLines: 1,
				Lines: []models.DiffLine{
					{Type: models.LineTypeDeletion, OldLineNo: 1, Content: "deleted"},
					{Type: models.LineTypeContext, OldLineNo: 2, NewLineNo: 1, Content: "ctx"},
				},
			},
		},
	}
	code, _ := findCodeContext(file, "old:1")
	if code != "deleted" {
		t.Errorf("expected code 'deleted', got %s", code)
	}
}

func TestFindCodeContext_NotFound(t *testing.T) {
	file := models.FileDiff{NewFile: "a.go", Hunks: []models.Hunk{}}
	code, _ := findCodeContext(file, "new:99")
	if code != "" {
		t.Errorf("expected empty when not found")
	}
}

func TestFindCodeContext_InvalidLineKey(t *testing.T) {
	file := models.FileDiff{NewFile: "a.go", Hunks: []models.Hunk{}}
	code, _ := findCodeContext(file, "invalid")
	if code != "" {
		t.Errorf("expected empty for invalid line key")
	}
}

func TestLineKeyToDisplay(t *testing.T) {
	if lineKeyToDisplay("") != "file" {
		t.Errorf("expected 'file' for empty key")
	}
	if lineKeyToDisplay("new:5") != "new:5" {
		t.Errorf("expected 'new:5'")
	}
}

func TestRenderTemplate(t *testing.T) {
	diff := makeDiff()
	comments := []models.Comment{
		{ID: "1", FilePath: "main.go", LineKey: "new:2", Content: "Add fmt import"},
	}
	tpl := "Type: {{.Type}}\n{{range .Files}}{{.Path}}: {{len .Comments}} comments\n{{end}}"
	out, err := RenderTemplate(diff, comments, tpl)
	t.Log(out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Type: commit") {
		t.Errorf("expected type in template output")
	}
	if !strings.Contains(out, "main.go: 1 comments") {
		t.Errorf("expected file comment count in template output")
	}
	if strings.Contains(out, "readme.md") {
		t.Errorf("expected readme.md to be omitted")
	}
}
