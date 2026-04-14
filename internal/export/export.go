package export

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"github.com/wtf2pr/wtf2pr/pkg/models"
)

// Export 将 diff 和 review 导出为指定格式（仅导出含 review 的核心信息，避免冗余 diff 原文）
func Export(diff *models.DiffResponse, comments []models.Comment, format models.ExportFormat) (string, error) {
	data := buildReviewExport(diff, comments)
	switch format {
	case models.ExportFormatMarkdown:
		return exportMarkdown(data), nil
	case models.ExportFormatJSON:
		return exportJSON(data)
	case models.ExportFormatXML:
		return exportXML(data)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// reviewExport 精简的 review 导出结构
type reviewExport struct {
	Type       string             `json:"type" xml:"Type"`
	CommitInfo *models.CommitInfo `json:"commitInfo,omitempty" xml:"CommitInfo,omitempty"`
	Files      []fileReviewExport `json:"files" xml:"Files>File"`
}

type fileReviewExport struct {
	Path      string          `json:"path" xml:"Path"`
	IsNew     bool            `json:"isNew" xml:"IsNew"`
	IsDeleted bool            `json:"isDeleted" xml:"IsDeleted"`
	Comments  []commentExport `json:"comments" xml:"Comments>Comment"`
}

type commentExport struct {
	LineKey    string `json:"lineKey" xml:"LineKey"`
	LineNo     string `json:"lineNo" xml:"LineNo"`
	Content    string `json:"content" xml:"Content"`
	CodeLine   string `json:"codeLine,omitempty" xml:"CodeLine,omitempty"`
	HunkHeader string `json:"hunkHeader,omitempty" xml:"HunkHeader,omitempty"`
}

// buildReviewExport 仅构造包含 review comment 的文件信息，并提取对应代码行与 hunk 上下文
func buildReviewExport(diff *models.DiffResponse, comments []models.Comment) *reviewExport {
	result := &reviewExport{
		Type:       string(diff.Type),
		CommitInfo: diff.CommitInfo,
		Files:      []fileReviewExport{},
	}
	if len(comments) == 0 {
		return result
	}

	// 按文件路径索引 comments
	fileComments := make(map[string][]models.Comment)
	for _, c := range comments {
		fileComments[c.FilePath] = append(fileComments[c.FilePath], c)
	}

	// 遍历 diff 文件，只保留有 comment 的
	for _, file := range diff.Files {
		path := file.NewFile
		if path == "" {
			path = file.OldFile
		}
		cmts, ok := fileComments[path]
		if !ok || len(cmts) == 0 {
			continue
		}

		fre := fileReviewExport{
			Path:      path,
			IsNew:     file.IsNew,
			IsDeleted: file.IsDeleted,
			Comments:  make([]commentExport, 0, len(cmts)),
		}
		for _, c := range cmts {
			codeLine, hunkHeader := findCodeContext(file, c.LineKey)
			fre.Comments = append(fre.Comments, commentExport{
				LineKey:    c.LineKey,
				LineNo:     lineKeyToDisplay(c.LineKey),
				Content:    c.Content,
				CodeLine:   codeLine,
				HunkHeader: hunkHeader,
			})
		}
		result.Files = append(result.Files, fre)
	}

	return result
}

// findCodeContext 根据 lineKey 在 fileDiff 中定位对应代码行与 hunk header
func findCodeContext(file models.FileDiff, lineKey string) (codeLine, hunkHeader string) {
	if lineKey == "" {
		return "", ""
	}
	parts := strings.Split(lineKey, ":")
	if len(parts) != 2 {
		return "", ""
	}
	targetNo, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", ""
	}
	isOld := parts[0] == "old"

	for _, hunk := range file.Hunks {
		for _, line := range hunk.Lines {
			var match bool
			if isOld {
				match = line.OldLineNo == targetNo
			} else {
				match = line.NewLineNo == targetNo
			}
			if match {
				return line.Content, fmt.Sprintf("@@ -%d,%d +%d,%d @@", hunk.OldStart, hunk.OldLines, hunk.NewStart, hunk.NewLines)
			}
		}
	}
	return "", ""
}

func lineKeyToDisplay(lineKey string) string {
	if lineKey == "" {
		return "file"
	}
	return lineKey
}

func exportMarkdown(data *reviewExport) string {
	var b strings.Builder
	b.WriteString("# Review Report\n\n")
	b.WriteString(fmt.Sprintf("**Type:** %s\n\n", data.Type))

	if data.Type == string(models.DiffTypeCommit) && data.CommitInfo != nil {
		b.WriteString(fmt.Sprintf("- **Commit:** %s\n", data.CommitInfo.Hash))
		b.WriteString(fmt.Sprintf("- **Author:** %s\n", data.CommitInfo.Author))
		b.WriteString(fmt.Sprintf("- **Date:** %s\n", data.CommitInfo.Date))
		b.WriteString(fmt.Sprintf("- **Message:** %s\n\n", data.CommitInfo.Message))
	}

	if len(data.Files) == 0 {
		b.WriteString("*No review comments.*\n")
		return b.String()
	}

	for _, f := range data.Files {
		b.WriteString(fmt.Sprintf("## %s\n\n", f.Path))
		if f.IsNew {
			b.WriteString("*New file*\n\n")
		}
		if f.IsDeleted {
			b.WriteString("*Deleted file*\n\n")
		}

		for _, c := range f.Comments {
			if c.LineKey != "" {
				b.WriteString(fmt.Sprintf("### Line %s", c.LineNo))
				if c.HunkHeader != "" {
					b.WriteString(fmt.Sprintf(" (%s)", c.HunkHeader))
				}
				b.WriteString("\n")
			} else {
				b.WriteString("### File Comment\n")
			}
			b.WriteString(fmt.Sprintf("- **Review:** %s\n", c.Content))
			if c.CodeLine != "" {
				b.WriteString(fmt.Sprintf("- **Code:** `%s`\n", c.CodeLine))
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

func exportJSON(data *reviewExport) (string, error) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func exportXML(data *reviewExport) (string, error) {
	b, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return xml.Header + string(b), nil
}
