# 单元测试清单

本文档列出了 `wtf2pr` 项目中所有已实现的 Go 后端单元测试，按包分类说明。

---

## 执行方式

```bash
# 运行全部测试（含竞态检测）
go test -race -count=1 ./internal/...

# 运行指定包的测试并输出详情
go test ./internal/config/ -v
go test ./internal/git/ -v
go test ./internal/review/ -v
go test ./internal/export/ -v
go test ./internal/server/ -v
```

---

## `internal/config`

测试文件：`internal/config/config_test.go`

| 测试函数 | 说明 |
|----------|------|
| `TestLoadMissingFile` | 配置文件不存在时返回 `nil, nil`（由调用方决定是否报错） |
| `TestLoadValidConfig` | 验证 YAML 解析与 `allowed_dirs` 的 `~` 展开、绝对化 |
| `TestLoadInvalidYAML` | 非法 YAML 返回错误 |
| `TestIsAllowed` | 验证目录边界匹配（含 `root+"2"` 前缀误判用例） |
| `TestIsAllowedEmptyList` | 允许列表为空时全部放行 |
| `TestNormalizePathBoundaries` | 验证 `NormalizePath` 边界：空串报错、裸 `~`、相对路径、尾部斜杠、不存在目标的回退 |
| `TestIsAllowedRootDir` | 允许列表包含根目录 `/` 时全部放行 |

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
| `TestReadUntrackedFileAsDiff_SymlinkSkipped` | 验证指向仓库外的符号链接被拒绝读取（防越界任意文件读） |
| `TestReadUntrackedFileAsDiff_Oversize` | 验证超过大小上限的未跟踪文件按二进制占位处理（不全量读入内存） |
| `TestGetUntrackedFiles` | 验证未跟踪文件出现在列表、已跟踪文件不出现 |
| `TestGetUntrackedFilesReal` | 验证含空格与非 ASCII 文件名的未跟踪文件正确列出（`-z` NUL 分割） |
| `TestUntrackedFileCountCap` | 验证未跟踪文件数量超限截断并标记 `Truncated` |
| `TestTextconvNotExecuted` | 验证恶意仓库配置的 textconv 钩子不会被执行（working 与 commit diff 双向） |
| `TestParseDiffSpacedFilename` | 验证含空格文件名以 `--- `/`+++ ` 行为准正确解析 |
| `TestParseDiffHunkContentWithDashDashPrefix` | 验证 hunk 内 `-- `/`++ ` 前缀的内容行不被误判为文件头且行数据完整 |
| `TestValidateCommitRef` | 验证 commit 引用白名单：合法 rev（HEAD~3、分支、hash）放行，选项注入（`--output=` 等）拒绝 |
| `TestGetCommitDiff_RejectsOptionInjection` | 验证选项注入不产生任意文件写 |
| `TestCommitsExistRejectsInvalidRef` | 验证非法引用返回 false |
| `TestCommitsExistBatch` | 验证批量 commit 存在性检查（单次 `git cat-file --batch-check`），含不存在与非法引用 |
| `TestRunGitCappedTruncation` | 验证 git 输出超限截断到最后完整行且仍可解析（>64KB 管道缓冲，含不排空管道的死锁回归），恢复上限后完整输出 |
| `TestGetWorkingDiffTruncatedFlag` | 验证巨型 working diff 时 `DiffResponse.Truncated=true` |

---

## `internal/review`

测试文件：`internal/review/store_test.go`

| 测试函数 | 说明 |
|----------|------|
| `TestNewStore_MemoryOnly` | 验证无文件路径时创建纯内存 Store，初始为空 |
| `TestStoreSaveAndGet` | 验证 `Save` 后 `Get` 返回正确数据，且返回的是深拷贝（修改结果不影响 Store） |
| `TestStoreFilePersistence` | 验证带文件路径的 Store 能将评论持久化到 JSON，并从 JSON 重新加载 |
| `TestStoreClear` | 验证 `Clear` 清空内存与磁盘文件 |
| `TestStoreSwitchFile` | 验证 `SwitchFile` 切换存储文件后自动加载该文件已有评论，且旧文件数据不受影响 |
| `TestStoreEmptyNotWritten` | 验证空评论保存时删除文件而非写入空文件 |
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
| `TestRenderTemplate` | 验证 Go `text/template` 自定义模板导出 |
| `TestValidateTemplateRejectsRecursion` | 验证递归模板（`{{define}}/{{template}}` 及其空白变体）被拒绝 |

---

## `internal/server`

测试文件：`internal/server/server_test.go`

**基础 API**

| 测试函数 | 说明 |
|----------|------|
| `TestHandleGetConfig` | 验证 `GET /api/config` 返回 reviewFile / reviewID / workDir |
| `TestHandleGetReview` | 验证 `GET /api/review` 初始返回空评论列表 |
| `TestHandleSaveReview` | 验证 `POST /api/review` 保存后，再次 GET 能正确取回 |
| `TestHandleGetDiff_Working` | 在真实临时 Git 仓库中修改文件，验证 `GET /api/diff?type=working` 返回 diff 数据 |
| `TestHandleGetCommits` | 在真实临时 Git 仓库中验证 `GET /api/commits` 返回分页 Commit 列表 |
| `TestHandleGetCommitsPageSizeCap` | 验证 `page_size` 超过上限时被收敛到 100 |
| `TestHandleGetReviews` | 验证 `GET /api/reviews` 返回有评论的历史 review 列表 |
| `TestHandleGetReviewsFiltersEmpty` | 向存储目录预置零评论与有评论文件，真实验证零评论被过滤 |
| `TestHandleGetReviewsCommitExists` | 验证 commit 类型 review 列表的 `commitExists` 真值/假值路径与 detail 校验 |
| `TestDefaultReviewListedAndSwitchable` | 验证默认 review.json 以 `default` 标识入列、可切换切回、detail 可读 |
| `TestHandleNewReview` | 验证 `POST /api/review/new` 生成 UUID reviewID 并切换当前 review（文件在首次非空保存时才落盘） |
| `TestHandleSwitchReview` | 验证 `POST /api/review/switch` 能切换到历史 review 并加载其评论数据；不存在时返回 400 |
| `TestHandleGetReviewDetail` | 验证 review 详情读取与不存在的 ID 返回 404 |
| `TestHandleExport` | 验证 `POST /api/export` 返回指定格式的导出内容 |
| `TestHandleExportMissingReviewID` | 验证显式指定不存在的 reviewID 返回 404（不静默回退当前 review） |
| `TestHandleGetTemplate` / `TestHandleSaveTemplate` | 验证导出模板读取与保存 |
| `TestHandleSaveTemplateRejectsRecursion` | 验证递归模板保存被 400 拒绝 |

**鉴权与路径安全**

| 测试函数 | 说明 |
|----------|------|
| `TestAuthMiddleware` | 验证无 token / 错误 token 返回 401，Bearer 与 `?token=` 正确放行 |
| `TestXAuthTokenHeader` | 验证 `X-Auth-Token` 头鉴权路径 |
| `TestReviewIDTraversalRejected` | 4 种路径遍历 ID × detail/switch/export 三入口均返回 400 |
| `TestSymlinkEscapeRejected` | 允许目录内指向外部的符号链接，浏览与切换均 403 |
| `TestBrowseDotDotAndTrailingSlash` | `..` 归一化后在范围内放行、越界拒绝 |

**工作目录与存储隔离**

| 测试函数 | 说明 |
|----------|------|
| `TestHandleSwitchWorkdir` | 验证切换 git 仓库成功、非 git 目录与不存在目录返回 400、存储目录按 workspace 隔离 |
| `TestHandleGetWorkdir` | 验证返回当前 workDir 与 workspace 列表（含 current 标记） |
| `TestWorkspaceDataIsolation` | 验证两个工作目录的评论数据互不可见 |
| `TestHandleBrowse` | 验证目录浏览（隐藏目录与文件被过滤、parent 计算、无效路径 400） |
| `TestHandleBrowseTruncated` | 验证超过 500 个目录时流式读取截断并标记 `truncated=true` |
| `TestHandleBrowseExactBoundary` | 验证恰好 500 个目录不误报截断、501 个正确截断 |
| `TestRequestBodyLimit` | 验证超过 10MB 的请求体被拒绝（400） |
| `TestRestrictedSwitchWorkdir` / `TestRestrictedBrowse` / `TestRestrictedWorkdirInfo` | 验证 allowed_dirs 受限模式下的放行/拒绝与信息标记 |
| `TestWorkspaceIDStable` | 验证相对路径/符号链接与绝对路径产生相同 workspace ID |

**数据迁移与并发**

| 测试函数 | 说明 |
|----------|------|
| `TestLegacyDataMigration` | 验证旧版扁平 review.json 自动迁移到 workspace 并加载 |
| `TestLegacyExportTemplateMigration` | 验证旧版 export.tpl 自动迁移到 workspace |
| `TestMigrationSkipsNonToolFiles` | 验证非本工具命名的 `review*.json`（如 reviewbackup.json）不被迁移 |
| `TestConcurrentSwitchAndSave` | `-race` 下并发切换 workdir 与保存，断言 review 文件不逃逸当前存储目录 |
| `TestConcurrentSwitchReviewAndSave` | `-race` 下并发切换/新建 review 与保存，同上不变量 |

> **说明**：`server` 测试使用真实 `git init` 创建的临时仓库，确保 API 与 Git 命令的集成链路完整可用；临时数据根目录通过包内注入与真实 `~/.wtf2pr` 隔离。

---

## 测试统计

- **总测试包数**：5 个
- **总测试函数**：93 个
- **全部通过状态**：✅ `PASS`（`go test -race -count=1 ./internal/...`）
