package git

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/wtf2pr/wtf2pr/pkg/models"
)

// ParseDiff 解析 unified diff 文本
func ParseDiff(raw string) ([]models.FileDiff, error) {
	lines := strings.Split(raw, "\n")
	files := make([]models.FileDiff, 0)
	var current *models.FileDiff
	var currentHunk *models.Hunk
	var oldLineNo, newLineNo int

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "diff --git ") {
			if current != nil {
				if currentHunk != nil {
					current.Hunks = append(current.Hunks, *currentHunk)
				}
				files = append(files, *current)
			}
			current = &models.FileDiff{}
			currentHunk = nil
			parts := strings.SplitN(line, " ", 4)
			if len(parts) >= 4 {
				current.OldFile = strings.TrimPrefix(parts[2], "a/")
				current.NewFile = strings.TrimPrefix(parts[3], "b/")
			}
			continue
		}
		if current == nil {
			continue
		}
		if strings.HasPrefix(line, "new file mode ") {
			current.IsNew = true
			current.OldFile = ""
			continue
		}
		if strings.HasPrefix(line, "deleted file mode ") {
			current.IsDeleted = true
			current.NewFile = ""
			continue
		}
		if strings.HasPrefix(line, "Binary files ") {
			current.IsBinary = true
			continue
		}
		if strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") {
			continue
		}
		if strings.HasPrefix(line, "@@ ") {
			if currentHunk != nil {
				current.Hunks = append(current.Hunks, *currentHunk)
			}
			hunk, err := parseHunkHeader(line)
			if err != nil {
				return nil, err
			}
			currentHunk = hunk
			oldLineNo = currentHunk.OldStart
			newLineNo = currentHunk.NewStart
			continue
		}
		if currentHunk == nil {
			continue
		}
		if len(line) == 0 {
			// unified diff 中的空行以空格开头，如果 raw line 长度为零
			// 可能是 patch 的最后一行之类，跳过
			continue
		}
		dl := parseDiffLine(line, &oldLineNo, &newLineNo)
		currentHunk.Lines = append(currentHunk.Lines, dl)
	}
	if current != nil {
		if currentHunk != nil {
			current.Hunks = append(current.Hunks, *currentHunk)
		}
		files = append(files, *current)
	}
	return files, nil
}

func parseHunkHeader(line string) (*models.Hunk, error) {
	// @@ -oldStart,oldLines +newStart,newLines @@
	parts := strings.Split(line, " ")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid hunk header: %s", line)
	}
	oldPart := strings.TrimPrefix(parts[1], "-")
	newPart := strings.TrimPrefix(parts[2], "+")
	oldStart, oldLines, err := parseRange(oldPart)
	if err != nil {
		return nil, err
	}
	newStart, newLines, err := parseRange(newPart)
	if err != nil {
		return nil, err
	}
	return &models.Hunk{
		OldStart: oldStart,
		OldLines: oldLines,
		NewStart: newStart,
		NewLines: newLines,
		Lines:    []models.DiffLine{},
	}, nil
}

func parseRange(s string) (start, lines int, err error) {
	parts := strings.Split(s, ",")
	start, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	if len(parts) > 1 {
		lines, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, err
		}
	} else {
		lines = 1
	}
	return start, lines, nil
}

func parseDiffLine(line string, oldNo, newNo *int) models.DiffLine {
	if len(line) == 0 {
		*oldNo++
		*newNo++
		return models.DiffLine{Type: models.LineTypeContext, OldLineNo: *oldNo - 1, NewLineNo: *newNo - 1, Content: ""}
	}
	prefix := line[0]
	content := line[1:]
	switch prefix {
	case '+':
		*newNo++
		return models.DiffLine{Type: models.LineTypeAddition, NewLineNo: *newNo - 1, Content: content}
	case '-':
		*oldNo++
		return models.DiffLine{Type: models.LineTypeDeletion, OldLineNo: *oldNo - 1, Content: content}
	case ' ':
		*oldNo++
		*newNo++
		return models.DiffLine{Type: models.LineTypeContext, OldLineNo: *oldNo - 1, NewLineNo: *newNo - 1, Content: content}
	case '\\':
		// "\ No newline at end of file" - 不影响行号
		return models.DiffLine{Type: models.LineTypeContext, Content: line}
	default:
		// 某些 diff 变体
		*oldNo++
		*newNo++
		return models.DiffLine{Type: models.LineTypeContext, OldLineNo: *oldNo - 1, NewLineNo: *newNo - 1, Content: line}
	}
}
