# 单元测试清单

本文档列出了 `wtf2pr` 项目中所有已实现的 Go 后端单元测试，按包分类说明。

---

## 执行方式

```bash
# 运行全部测试
go test ./...

# 运行指定包的测试并输出详情
go test ./internal/git/ -v
go test ./internal/review/ -v
go test ./internal/export/ -v
go test ./internal/server/ -v
```

---

## `internal/git`

测试文件：`internal/git/parser_test.go`、`internal/git/git_test.go`

| 测试函数 | 说明 |
|----------|------|
| `TestParseDiffEmpty` | 验证空 diff 字符串返回空文件列表 |
| `TestParseDiffNewFile` | 验证新增文件（`new file mode`）的 diff 解析，包含 hunk 与 addition 行号 |
| `TestParseDiffDeletedFile` | 验证删除文件（`deleted file mode`）的 diff 解析，包含 deletion 行号 |
| `TestParseDiffModifiedFile` | 验证修改文件的 diff 解析，包含 context / deletion / addition 混合行 |
| `TestParseDiffBinaryFile` | 验证二进制文件标记（`Binary files differ`）正确识别，无 hunks |
| `TestParseDiffMultipleFiles` | 验证同一 diff 文本中包含多个文件时分别解析 |
| `TestParseDiffMultipleHunks` | 验证单个文件包含多个 hunk 时全部解析 |
| `TestParseHunkHeaderInvalid` | 验证非法 hunk header 返回错误 |
| `TestParseRange` | 验证 hunk range 解析，含 `start,lines` 与仅 `start` 两种情况 |
| `TestParseDiffLine` | 验证单行 diff 前缀解析（`+`、`-`、空格、`\`）及行号递增逻辑 |
| `TestParseDiffWithLongContent` | 验证超长内容行（2000 字符）解析不丢失数据 |
| `TestReadUntrackedFileAsDiff_Text` | 验证未跟踪文本文件被构造为完整 `FileDiff`（新增模式） |
| `TestReadUntrackedFileAsDiff_Binary` | 验证未跟踪二进制文件识别为 `IsBinary=true` |
| `TestReadUntrackedFileAsDiff_NotFound` | 验证读取不存在的未跟踪文件返回错误 |
| `TestGetUntrackedFiles` | 调用 `git ls-files` 接口，确保无 panic |

---

## `internal/review`

测试文件：`internal/review/store_test.go`

| 测试函数 | 说明 |
|----------|------|
| `TestNewStore_MemoryOnly` | 验证无文件路径时创建纯内存 Store，初始为空 |
| `TestStoreSaveAndGet` | 验证 `Save` 后 `Get` 返回正确数据，且返回的是深拷贝（修改结果不影响 Store） |
| `TestStoreFilePersistence` | 验证带文件路径的 Store 能将评论持久化到 JSON，并从 JSON 重新加载 |
| `TestStoreClear` | 验证 `Clear` 清空内存与磁盘文件 |
| `TestStoreConcurrency` | 高并发读写测试（100 协程），验证 `sync.RWMutex` 无线程安全问题 |

---

## `internal/export`

测试文件：`internal/export/export_test.go`

| 测试函数 | 说明 |
|----------|------|
| `TestExportMarkdown_NoComments` | 无评论时导出 Markdown 仅含 Commit 元信息与 "No review comments" 提示 |
| `TestExportMarkdown_WithComments` | 有评论时仅导出含评论的文件，包含行号、对应代码、hunk header |
| `TestExportJSON` | 验证 JSON 导出格式合法，且仅包含被 review 的文件 |
| `TestExportXML` | 验证 XML 导出格式合法，包含正确标签与内容 |
| `TestExportUnsupportedFormat` | 验证传入不支持的格式（如 `yaml`）返回错误 |
| `TestFindCodeContext_NewLine` | 验证通过 `new:{lineNo}` 在 diff 中定位到对应新增代码行 |
| `TestFindCodeContext_OldLine` | 验证通过 `old:{lineNo}` 在 diff 中定位到对应删除代码行 |
| `TestFindCodeContext_NotFound` | 验证不存在的行号返回空字符串 |
| `TestFindCodeContext_InvalidLineKey` | 验证非法 `lineKey` 格式返回空字符串 |
| `TestLineKeyToDisplay` | 验证 `lineKey` 空值显示为 `file`，非空保持原样 |

---

## `internal/server`

测试文件：`internal/server/server_test.go`

| 测试函数 | 说明 |
|----------|------|
| `TestHandleGetConfig` | 验证 `GET /api/config` 返回当前 review 文件路径配置 |
| `TestHandleGetReview` | 验证 `GET /api/review` 初始返回空数组 |
| `TestHandleSaveReview` | 验证 `POST /api/review` 保存后，再次 GET 能正确取回 |
| `TestHandleGetDiff_Working` | 在真实临时 Git 仓库中修改文件，验证 `GET /api/diff?type=working` 返回 diff 数据 |
| `TestHandleGetCommits` | 在真实临时 Git 仓库中验证 `GET /api/commits` 返回分页 Commit 列表 |
| `TestHandleExport` | 验证 `POST /api/export` 返回指定格式的导出内容 |

> **说明**：`server` 测试使用真实 `git init` 创建的临时仓库，确保 API 与 Git 命令的集成链路完整可用。

---

## 测试统计

- **总测试包数**：4 个
- **总测试函数**：约 30 个
- **全部通过状态**：✅ `PASS`
