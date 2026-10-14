# 系统设计文档 (System Design)

## 项目概述

**WTF2PR** 是一个本地 Git 代码 Review 工具。它能读取本地 Git 仓库中未提交的变更（Working Tree，包含已修改、已暂存、未跟踪文件）或某个具体 Commit 的 Diff，在 Web 页面上进行可视化展示。用户可对文件整体或具体代码行添加 Review 意见，最终将 Review 结果以结构化格式（Markdown / JSON / XML / 自定义 Template）一次性导出，交给 AI Coding 工具进行问题修复。

## 技术栈

- **后端**：Go + Gin 框架 + Cobra（CLI 入口）
- **前端**：Vue 3 (Vite) + Tailwind CSS v4
- **打包**：Go 官方 `embed` 将前端产物打包为单一可执行二进制文件
- **依赖**：`github.com/google/uuid` 用于生成唯一 Review ID；`gopkg.in/yaml.v3` 用于解析 YAML 启动配置

### 提供设定

1. 后端通过 `github.com/spf13/cobra` 提供命令行入口。
2. 后端通过 Go 官方 `embed` 将前端页面打包到一个可执行二进制文件。
3. 通过 `wtf2pr web [--config={配置文件}] [--port={端口}] [--workdir={工作目录}]` 指令启动服务；服务配置集中在 YAML 配置文件（默认 `~/.wtf2pr/config.yaml`），支持 `port` / `host` / `workdir` / `token` / `allowed_dirs`；`port` 与 `workdir` 支持 CLI 覆盖（优先级：CLI 显式指定 > 配置文件 > 默认值），其余配置项以配置文件为准。
4. Review 数据按工作目录隔离持久化到 `~/.wtf2pr/workspaces/<目录哈希>/review_{uuid}.json`，支持历史 Review 的列表查看与切换；启动后可在页面上切换工作目录（目标必须是 git 仓库），旧版扁平存储数据在对应工作区数据区首次创建时自动迁移。
5. 访问令牌为空时每次启动随机生成（可在配置文件中指定固定值），Web 端所有 API 需携带令牌访问（`Authorization: Bearer`、`X-Auth-Token` 头或 `?token=` 查询参数）。
6. 配置文件 `allowed_dirs` 限制 Web 端可访问的文件路径范围；切换工作目录与目录浏览越界时返回 403。

## 系统架构

### 1. 前端

- **Diff 展示**：左侧文件列表 + 右侧代码 Diff 的双栏布局，支持新增/删除/上下文行的颜色区分。
- **Commit 选择器**：切换为 Commit 模式时，通过分页下拉框（每页 10 条）快速选择历史 Commit。
- **Review 交互**：
  - 点击代码行可在该行**下方直接内联添加**行级评论。
  - 点击文件标题栏"添加文件评论"可对文件整体发表意见。
  - 评论支持删除。
- **Review 状态**：顶部显示当前 Review ID、保存状态（已保存/未保存），并提供"新建 Review"、"切换 Review"、"导出"、"保存"控制。
- **开发模式**：Vite 开发服务器通过 Proxy 将 `/api/*` 请求转发到后端，实现前后端分离开发。

### 2. 后端

- **Config 模块**：
  - 加载 YAML 启动配置（`port` / `host` / `workdir` / `token` / `allowed_dirs`），文件缺失时全部使用默认值。
  - 路径规范化（`~` 展开、绝对化、符号链接解析）与目录边界匹配，为 `allowed_dirs` 访问控制提供判定。
- **Git 模块**：
  - 执行 `git diff HEAD` 获取已跟踪文件的未提交变更。
  - 执行 `git ls-files --others --exclude-standard -z` 补充未跟踪（untracked）文件（NUL 分割支持空格/非 ASCII 文件名；仅普通文件，单文件超过 1MB 按二进制占位，数量超过 500 截断）。
  - 执行 `git show {commit}` 获取指定 Commit 的 Diff 与 Commit 元信息。
  - 执行 `git log` + `git rev-list --count` 实现分页 Commit 列表查询（`page_size` 上限 100）。
  - 内置 Unified Diff 解析器，将 diff 文本解析为结构化数据（文件 → Hunk → 行；文件名以 `--- `/`+++ ` 行为准，支持含空格文件名）。
  - diff 类输出（`git diff HEAD` / `git show -p`）经 `runGitCapped` 限制在 20MB（超限排空管道后截断并标记 `DiffResponse.Truncated`）。
  - 安全基线：commit 引用白名单校验 + `--end-of-options` 防选项注入；`--no-ext-diff --no-textconv` + 禁用 fsmonitor hook + `-c log.showSignature=false` + 枚举置空 `filter.<driver>.clean/smudge/process` 防恶意仓库配置执行命令（textconv / ext-diff / gpg.program / filter 驱动）；`GIT_NO_LAZY_FETCH=1` 防部分克隆懒加载触发 `core.sshCommand`/`credential.helper`；`core.quotePath=false` 防文件名转义错位；git 子进程环境变量白名单过滤（去除 `GIT_DIR`/`GIT_WORK_TREE`/`GIT_CONFIG*` 等劫持向量）；单个子进程 60s 超时（`gitCmdTimeout`）；批量 commit 存在性检查走单次 `git cat-file --batch-check`。
  - merge commit 经 `-m --first-parent` 展示与第一父的差异；空仓库（无 HEAD）working diff 仅返回 untracked、commits 返回空列表。
- **Review 模块**：
  - 内存存储 + JSON 文件持久化（临时文件 + 原子替换，权限 0600），线程安全（`sync.RWMutex`）。
  - 支持 `SwitchFile` 动态切换存储文件，切换时自动加载该文件已有评论。
  - 新建 Review 时生成 UUID 并切换存储路径，文件在首次非空保存时才写入 `review_{uuid}.json`（空评论保存则删除文件）。
  - 提供扫描目录接口，列出有评论的历史 Review 文件（零评论文件被过滤；默认 `review.json` 以 `default` 标识入列，可切换/查看）。
- **Export 模块**：
  - 仅导出**包含 Review 评论的核心信息**，避免完整 diff 原文导致的 Token 冗余。
  - 输出包含：Commit 元信息、被评论文件路径、评论内容、对应代码行、Hunk Header。
  - 支持 Markdown、JSON、XML 与自定义 Template（Go `text/template`）四种格式。
- **Server 模块**：
  - 提供 RESTful API（`/api/auth/check`、`/api/diff`、`/api/commits`、`/api/config`、`/api/review`、`/api/review/detail`、`/api/reviews`、`/api/review/new`、`/api/review/switch`、`/api/export`、`/api/template`、`/api/workdir`、`/api/fs/browse`）。
  - 所有 API 经 token 鉴权中间件（常量时间比较）；review ID 经白名单校验，防止路径遍历越出存储目录。
  - 请求体上限 10MB；HTTP 服务器配置 ReadHeader/Read/Write/Idle 超时。
  - 自定义导出模板经安全校验（禁止 `{{define}}/{{template}}/{{block}}` 递归动作），模板文件与 review 数据落盘均为原子写 + 0600。
  - 切换 workdir/review 与保存互斥（统一 `s.mu → store.mu` 锁序），避免并发交错。
  - 静态文件服务通过 `NoRoute` 兜底，将未匹配路由指向 embed 的前端资源。

### 3. 构建与开发

- `make dev-web`：以开发模式运行前端（Vite dev server + Proxy）。
- `make dev-server`：以开发模式运行后端（不重新构建前端，可用 `WTF2PR_CONFIG` 指定配置文件）。
- `make build`：构建 macOS / Linux 最终交付物（交叉编译 `amd64` + `arm64`，打包为 `tar.gz`）。
- `make lint`：同时执行后端 `golangci-lint / go vet / gofmt` 与前端 `eslint` 检查。
- `make clean`：清理构建产物。

### 4. 测试

- 后端单元测试覆盖 `internal/config`（配置加载与路径访问控制）、`internal/git`（Diff 解析、未跟踪文件读取）、`internal/review`（存储与持久化）、`internal/export`（格式化导出）、`internal/server`（API 集成测试）。
- 测试文档详见 `docs/unit_tests.md`。

## AI 开发守则
> 当 AI 依据本文档进行功能开发时，必须遵守以下规则：

1. 你要严谨认真的完成任务，在对于相关技术由疑问的时候，需要通过网络查询或者mcp工具查询获取更多信息
2. 前后端通信要以定义严格的结构体进行处理，不要出现临时定义临时使用的情况
3. 你要注意代码质量，该复用的代码要进行复用，保证一个功能只有一个函数实现的情况
4. 完成编码之后要进行代码review，自己对照目标找出问题并进行修复
5. 完成功能开发修改之后，需要看看是否需要反向更新system_design文档。必要的时候进行更新，但是你不允许更新`AI 开发守则`下的任何内容。(什么时候必要：较大模块功能更新，模块能力拓展，较大功能更新等)
