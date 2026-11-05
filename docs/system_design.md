# 系统设计文档 (System Design)

## 项目概述

**WTF2PR** 是一个本地 Git 代码 Review 工具。它能读取本地 Git 仓库中未提交的变更（Working Tree，包含已修改、已暂存、未跟踪文件）、某个具体 Commit 的 Diff，或当前分支相对目标本地分支的三点 Diff。用户可在 Web 页面浏览变更、查看完整文件并对指定 Commit 添加行级或文件级评论，最后将 Review 结果导出为 Markdown / JSON / XML / 自定义 Template，交给 AI Coding 工具修复。

## 技术栈

- **后端**：Go + Gin 框架 + Cobra（CLI 入口）
- **前端**：Vue 3 (Vite) + Tailwind CSS v4
- **打包**：Go 官方 `embed` 将前端页面打包到单一可执行二进制文件
- **依赖**：`github.com/google/uuid` 用于生成唯一 Review ID；`@vue/server-renderer` 仅作为前端测试的直接开发依赖
- **字体**：IBM Plex Sans / Mono 自托管，附 OFL 许可证，无需外部字体服务

### 提供设定

1. 后端通过 `github.com/spf13/cobra` 提供命令行入口。
2. 后端通过 Go 官方 `embed` 将前端页面打包到一个可执行二进制文件。
3. 通过 `wtf2pr web --port={port} --host={host} --workdir={工作目录}` 指令启动服务。
4. Review 默认持久化到 `~/.wtf2pr/review.json`；指定 `--review-id` 时使用 `~/.wtf2pr/review_<id>.json`，指定 `--review-file` 时使用该完整路径。
5. 安全历史 Review ID 允许字母、数字、点、下划线和连字符，兼容 `feature_x` 等旧自定义 ID；路径穿越、斜杠和控制字符被拒绝。

## 系统架构

### 1. 前端

- **统一审查工作区**：使用「代码审查 / 历史 Review / 导出模板」主导航，来源选择器支持未提交、指定 Commit、当前分支到目标本地分支。
- **工作目录切换**：输入本机 Git 仓库绝对路径或子目录，成功后回到未提交视图并重新加载来源选择、分支与 diff；保留目录树/平铺/排序能力及 Review 文件。初始化配置、目标切换和导出均有世代保护，用户未提交草稿不被旧配置覆盖，成功响应统一显示规范化仓库根目录。
- **Diff 展示**：全宽双栏布局，左侧文件栏、右侧代码 Diff，支持新增/删除/上下文行、双行号、Commit 与 merge-base 元数据。
- **文件浏览器**：支持目录树/平铺、目录名选中筛选、目录展开折叠、连续滚动，不再按文件分页。目录名与箭头分别处理筛选和折叠，平铺使用同一过滤结果。内置路径、类型、测试优先、文档优先和反向排序，视图及排序偏好保存在浏览器本地。
- **完整文件弹窗**：原生 `<dialog>` 展示 old/new 完整文件，支持行号、复制及失败降级、Esc 和遮罩关闭；重命名按侧选择真实路径，新增/删除缺失侧有明确状态。
- **Review 交互**：仅指定 Commit 可编辑行级和文件级评论。评论加载采用水合状态，完成前只读并禁止保存/导出；失败可重试。保存、导出、初始化和模板请求均使用 serial/revision 防止旧响应覆盖新状态。
- **Review 状态**：顶部显示当前 Review 标识与保存状态，提供新建、切换、导出和保存控制；保存、新建和切换共用 mutation 锁。
- **视觉与可访问性**：Framed sheet 风格、全宽桌面/窄屏比例布局、语义色变量、focus-visible、加载/空/错误状态和中文页面元数据。
- **开发模式**：Vite 开发服务器通过 Proxy 将 `/api/*` 请求转发到后端。

### 2. 后端

- **Git 模块**：
  - 执行 `git diff -M HEAD` 获取已跟踪文件的未提交变更，并通过 `git ls-files --others --exclude-standard -z` 补充未跟踪文件。
  - 执行 `git show <resolved-commit>` 获取指定 Commit 的 Diff 与元信息，所有 revision 先经 `rev-parse --verify --end-of-options` 解析为完整哈希。
  - 执行 `git merge-base refs/heads/<base> HEAD` 和 `git diff -M <base>...HEAD` 获取本地分支三点 Diff，main 优先、master 回退选择默认目标分支。
  - 使用 `git diff/diff-tree --raw -z -r -M` 的 NUL 元数据建立真实文件身份，覆盖 working/commit/branch，避免空格、引号、Unicode 和路径别名造成错误授权。merge commit 保留 Git 原生合并语义。
  - 工作区普通文件通过 `Lstat`、`OpenRoot`、打开后 `Stat/SameFile` 校验读取；符号链接按 Git blob 语义返回链接文本，不读取链接目标。
  - 内置 Unified Diff 解析器，完整路径同时参考 `---/+++` 与 Git 引号解码；raw 元数据是全文读取授权的最终依据。
- **完整文件模块**：working 旧侧为 HEAD、新侧为工作区；Commit 旧侧为父提交、新侧为指定 Commit；branch 旧侧为 merge-base、新侧为 HEAD。逐侧判断二进制、存在性和内容，支持同路径删除+未跟踪新增、重命名环和文本/二进制互转。
- **Review 模块**：内存存储 + JSON 文件持久化，线程安全（`sync.RWMutex`）。支持切换存储文件并加载已有评论。新建 Review 只生成 UUID 并切换路径，首次非空保存才创建文件。
- **Review 并发**：服务端 RWMutex 覆盖 active Review 文件与 Store 的读写及复合快照；前端 mutation 锁防止保存期间新建/切换。
- **Export 模块**：仅导出包含评论的核心信息，支持 Markdown、JSON、XML 和 Go `text/template` 自定义模板。当前请求使用当前评论快照；省略 comments 的旧 working/commit 请求仅在 Review 类型和 canonical Commit 一致时回退 Store。当前 working/branch 重新计算 diff，有效历史 working/commit 快照可直接导出。
- **工作目录模块**：`POST /api/config/workdir` 支持本地及远程部署访问，不限制客户端 IP、Origin 或 Host。绝对路径或仓库子目录经符号链接解析和 `git rev-parse --show-toplevel` 归一为仓库根目录。校验失败保持原目录；成功只更新工作目录，不切换 Review 文件。所有 Git 调用使用加锁快照，切换与 Review 保存/导出等复合操作共用 `reviewMu`，避免跨仓库混用。
- **Server 模块**：提供 RESTful API：
  - `GET /api/diff`、`GET /api/commits`、`GET /api/branches`、`GET /api/file/content`
  - `GET /api/config`、`POST /api/config/workdir`
  - `GET /api/review`、`GET /api/reviews`、`GET /api/review/detail`
  - `POST /api/review`、`POST /api/review/new`、`POST /api/review/switch`
  - `POST /api/export`、`GET/POST /api/template`
  - 静态文件服务通过 `NoRoute` 兜底，将未匹配路由指向 embed 的前端资源。

### 3. 构建与开发

- `make dev-web`：以开发模式运行前端（Vite dev server + Proxy）。
- `make dev-server`：以开发模式运行后端（不重新构建前端）。
- `make build`：构建 macOS 与 Linux 交付物（amd64 + arm64，tar.gz）。
- `make lint`：同时执行后端 `golangci-lint / go vet / gofmt` 与前端 `eslint` 检查。
- `make embed`：构建前端并复制到 `cmd/wtf2pr/dist`，供 Go embed 使用。
- `make clean`：清理构建产物。

### 4. 测试

- 后端单元与集成测试覆盖 Diff 解析、raw 路径身份、分支与 merge-base、完整文件新旧侧、路径安全、Review 存储、导出和 Server API。
- 前端 Node 原生测试覆盖文件树/排序、API 参数、Vue SSR 组件契约；真实浏览器验证三种来源、异步水合、评论、导出、历史和模板。
- 当前实测为 **91 个顶层 Go 测试函数 + 33 个 Node 测试**，测试清单详见 `docs/unit_tests.md`。

## AI 开发守则
> 当 AI 依据本文档进行功能开发时，必须遵守以下规则：

1. 你要严谨认真的完成任务，在对于相关技术由疑问的时候，需要通过网络查询或者mcp工具查询获取更多信息
2. 前后端通信要以定义严格的结构体进行处理，不要出现临时定义临时使用的情况
3. 你要注意代码质量，该复用的代码要进行复用，保证一个功能只有一个函数实现的情况
4. 完成编码之后要进行代码review，自己对照目标找出问题并进行修复
5. 完成功能开发修改之后，需要看看是否需要反向更新system_design文档。必要的时候进行更新，但是你不允许更新`AI 开发守则`下的任何内容。(什么时候必要：较大模块功能更新，模块能力拓展，较大功能更新等)
