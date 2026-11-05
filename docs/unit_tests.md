# 单元与集成测试清单

本文档按包说明 WTF2PR 当前测试。数量来自实跑 `go test -list '^Test'` 与 `npm test`，不是估计值。

## 执行方式

```bash
# 后端全部测试及竞态检测
go test -race ./...

# 静态检查与构建
go vet ./...
go build ./cmd/wtf2pr

# 指定后端包
go test ./internal/git/ -v
go test ./internal/review/ -v
go test ./internal/export/ -v
go test ./internal/server/ -v

# 前端全部测试、lint 与构建
cd web
npm test
npm run lint
npm run build
```

## 实测统计

| 范围 | 顶层测试数 | 说明 |
|------|------------|------|
| `internal/git` | 40 | Diff 解析、raw 路径身份、分支与完整文件 |
| `internal/review` | 6 | 内存/文件 Store、深拷贝、切换与并发 |
| `internal/export` | 11 | Markdown / JSON / XML、评论代码定位 |
| `internal/server` | 34 | 真实临时 Git 仓库上的 API 集成 |
| Go 合计 | **91** | 不含子测试展开数量 |
| `web/tests/apiClient.test.js` | 4 | Base/path/side、AbortSignal 与工作目录请求/异常响应 |
| `web/tests/fileContentComponent.test.js` | 2 | 原生 dialog 与 diff 查看入口 |
| `web/tests/fileExplorer.test.js` | 13 | 分类、树、排序、路径与节点 key |
| `web/tests/fileExplorerComponent.test.js` | 2 | 连续渲染、无分页、目录选中筛选与恢复 |
| `web/tests/workdirSwitcher.test.js` | 2 | 路径、错误、切换中禁用与可访问状态 |
| `web/tests/workdirBehavior.test.js` | 7 | 真实 Vue 内存渲染、异步乱序与草稿/路径状态 |
| `web/tests/workspaceComponent.test.js` | 3 | 来源空状态、工作目录展示与统一导航 |
| Node 合计 | **33** | 全部通过，未新增测试框架 |

## `internal/git`

### Diff 与分支

- Unified diff 的新增、删除、修改、二进制、多文件、多 hunk、长行与非法 hunk header。
- 空格路径、Git 引号 Unicode、尾随空格及 `and/b` 路径别名。
- `main` / `master` 默认分支优先级、detached HEAD、无可用目标分支、同名 branch/tag 及列表→diff round-trip。
- 三点 diff 排除 main 独有提交和未提交修改，精确核对 merge-base。
- merge commit 保留 Git 原生合并 diff，不与单父 raw diff 强制对齐。

### 完整文件与路径安全

- working/commit/branch 的新旧版本、未跟踪、重命名、重命名环、同路径删除+未跟踪新增。
- 新增旧侧、删除新侧返回 `exists=false`；错侧重命名和未匹配路径被拒绝。
- 文本↔二进制双向转换逐侧判断内容，空文件与 NUL 二进制分别处理。
- `..`、绝对路径、非规范路径、未修改文件、`.git`、非仓库和未知 revision 的状态分类。
- 符号链接返回链接文本，不读取仓库外或根内目标；普通文件使用 `OpenRoot` 与 `SameFile` 防护。
- 40 位不存在哈希拒绝，大写哈希规范化，选项形式 revision 不产生输出文件。
- raw NUL 元数据在三种来源下覆盖真实路径与负向别名。

## `internal/review`

- 纯内存 Store 初始化、保存与深拷贝。
- JSON 持久化、重新加载、切换文件及旧文件不受影响。
- 100 协程并发读写验证 `sync.RWMutex`。
- 空评论使用 Save 的现有清空语义，不再保留仅测试调用的 `Clear` 生产接口。

## `internal/export`

- 无评论、文件级/行级评论、仅含评论文件和关键代码。
- Markdown / JSON / XML 输出结构及不支持格式错误。
- old/new line key 定位、无效 key 与不存在行号。
- 自定义模板渲染和错误传播。

## `internal/server`

- 配置、初始 Review 对象、保存/读取评论与空历史数组。
- working/commit/branch diff、分支列表、Commit 分页和缺失参数错误。
- 完整文件 API、路径穿越、未修改文件、未知来源与仓库故障的 400/500 分类。
- Review 新建、历史列表、详情、切换与空评论过滤。
- 合法 `feature_x` 的详情、切换和导出完整链路；恶意 ReviewID 继续拒绝。
- Markdown / JSON / XML / Template 导出、当前草稿、旧客户端省略 comments 的兼容回退。
- 跨 Commit 评论隔离、stale working/branch 快照重算、缺失 Commit 的有效历史快照导出。
- canonical Commit 保存、自定义模板保存与读取。
- 40 组 `/review/new` 与 `/api/config` 并发请求，供 race detector 检查 active Review 路径。
- 工作目录切换：A/B 仓库 config/diff/commits/branches/file content 来源与 Review 保留；子目录归一；空、相对、普通文件、不存在和非仓库路径不改变旧目录。
- 工作目录远程部署与并发：非 loopback 客户端及部署 Origin/Host 可切换仓库；30 组切换、diff 和分支并发由 race detector 检查。

## `web/tests`

- API：branch Base、完整文件 path/side、URL 编码、AbortSignal、工作目录 POST 与非法成功响应。
- 文件分类：测试、文档、扩展名、自然路径、升降序、输入不变性。
- 树：目录优先、深层目录、空路径段、测试组内排序、file→directory 唯一 key。
- 默认文件：优先首个文本 diff，只有二进制时回退。
- 完整文件：原生 `<dialog>`、old/new 按钮与 editable/readonly 查看入口。
- 工作区：三种来源、单分支空状态、统一代码审查导航；工作目录路径、错误和切换中状态从 App 传递到实际输入组件。
- 工作目录输入：显式标签、错误关联、aria-invalid/busy、切换中状态通知、输入与按钮禁用。
- 内存客户端渲染：使用现有 Vue `createRenderer` 与 Vite `ModuleRunner` 执行真实组件，覆盖初始 Commit+慢 config、正常初始化、草稿保护、失败保持、旧导出、旧分支和旧 diff 乱序响应。

## 真实浏览器补充验证

隔离 main/feature 仓库执行过三种来源、39 文件连续滚动、树/平铺、全部排序、目录折叠、Commit 分页、完整文件新旧侧与关闭/重开、二进制/空/删除状态、文件级和行级评论保存、四种导出、历史 Review 和模板操作。生产 embed 在 1440/768/390 视口下完成视觉检查，结果为 0 FAIL。

工作目录专项使用两个独立临时 Git 仓库，实际从 A 切换到 B 的子目录并归一到根目录。B 的 working、指定 Commit、main...HEAD 分支对比和完整文件均读取正确内容；目录筛选、树/平铺和已保存 Commit 评论正常，Review 标识不变。受控延迟初始配置后，旧响应不能覆盖 B；未提交草稿也不会被初始配置覆盖。非法路径与非 Git 目录有可见错误且保持 B，恶意 Origin 的真实请求返回 403。

浏览器剪贴板权限和 macOS 原生键盘截图通道存在环境限制，未将对应能力标为 PASS；复制失败降级、按钮与遮罩关闭已实点验证。
