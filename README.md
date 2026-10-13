# WTF2PR

> “别让一声 WTF 成为结局，让它成为修复的开始。”

**WTF2PR** 是一个本地代码 Review 工具。它能把 Git 仓库中未提交的变更（包括已修改、已暂存、未跟踪的文件）或某个具体 Commit 的 Diff，在 Web 页面上直观展示出来。你可以逐行添加 Review 意见，最后将 Review 结果以结构化格式（Markdown / JSON / XML）一次性导出，交给 AI Coding 工具进行修复。

---

## 功能特性

- **Working Tree Review**：查看当前工作目录中所有未提交的变更，包括：
  - 已修改但未 `git add` 的文件
  - 已 `git add` 但未 `git commit` 的文件
  - 尚未被 Git 跟踪的新文件（untracked）
- **Commit Review**：输入任意 Commit Hash，查看该 Commit 的完整 Diff 与 Commit 信息。
- **工作目录切换**：启动后可在页面上随时切换工作目录（支持目录浏览与最近使用列表），无需重启服务。
- **按目录隔离存储**：每个工作目录的 Review 数据独立存储在 `~/.wtf2pr/workspaces/<目录哈希>/` 下，互不干扰；旧版扁平存储的数据会在首次启动时自动迁移。
- **访问令牌鉴权**：服务启动时自动生成随机 Token（也可用 `--token` 指定），Web 端所有 API 均需携带令牌访问。
- **行级评论**：可对文件整体或具体代码行添加 Review 意见。
- **结构化导出**：支持导出为 **Markdown**、**JSON**、**XML**，仅包含被评论的文件与关键代码行，避免冗余 Token 消耗。
- **Review 持久化**：Review 数据自动保存到本地 JSON 文件，关闭服务后仍可恢复。
- **单二进制部署**：前端通过 Go `embed` 直接打包进后端可执行文件，无需额外部署。

---

## 技术栈

- **后端**：Go + Gin + Cobra
- **前端**：Vue 3 (Vite) + Tailwind CSS v4
- **打包**：Go `embed` 将前端产物打包为单一二进制

---

## 快速开始

### 1. 构建

确保已安装 **Go** 与 **Node.js / npm**。

```bash
cd web
npm install
npm run build
cd ..
rm -rf cmd/wtf2pr/dist
cp -r web/dist cmd/wtf2pr/dist
go build -o wtf2pr ./cmd/wtf2pr
```

### 2. 启动服务

```bash
# 使用默认参数启动（端口 8322，当前目录作为 Git 工作目录）
./wtf2pr web

# 指定端口与工作目录
./wtf2pr web --port=8322 --workdir=/path/to/your/repo

# 手动指定访问令牌（默认每次启动随机生成）
./wtf2pr web --token=my-fixed-token

# 指定 Review 存储文件（指定后不启用按工作目录隔离存储）
./wtf2pr web --review-file=/path/to/review.json

# 使用 Review ID 命名当前工作目录数据区内的文件：review_{id}.json
./wtf2pr web --review-id=feature_x
```

启动后终端会输出访问令牌与带令牌的链接，例如：

```
Access token: 7a3982a2e90860bd551af76b6c2ae586
Open in browser: http://localhost:8322/?token=7a3982a2e90860bd551af76b6c2ae586
```

打开链接即可进入；也可以直接访问 `http://localhost:8322` 并手动输入令牌。

### 3. 开发模式运行（前后端分离）

```bash
# 终端 1：启动后端
make dev-server

# 终端 2：启动前端（默认监听 127.0.0.1:5173）
make dev-web

# 自定义前端监听地址和端口
make dev-web DEV_HOST=0.0.0.0 DEV_PORT=3000
```

前端开发服务器会通过 Vite Proxy 将 `/api/*` 请求转发到 `http://localhost:8322`。

### 4. 打开浏览器

访问启动日志中输出的带 Token 链接，在左侧选择文件，在右侧 Diff 中点击代码行即可添加 Review。点击顶部目录名可随时切换工作目录，页面数据会按新目录重新加载。

---

## CLI 参数说明

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--port` | `8322` | 服务监听端口 |
| `--host` | `0.0.0.0` | 服务监听地址（设为 `127.0.0.1` 仅本地访问） |
| `--workdir` | 当前目录 | Git 工作目录（启动后可在页面切换） |
| `--token` | `""` | Web 访问令牌，为空时每次启动随机生成 |
| `--review-file` | `""` | Review 数据存储的 JSON 文件完整路径（兼容旧模式） |
| `--review-id` | `""` | Review ID，用于命名当前工作目录数据区内的存储文件 |

> 数据存储规则（未指定 `--review-file` 时）：
> - 每个工作目录对应独立数据区：`~/.wtf2pr/workspaces/<目录哈希>/`
> - 当前 Review 默认存储为 `review.json`，新建的 Review 存储为 `review_{uuid}.json`，导出模板为 `export.tpl`
> - 首次启动时会将旧版直接放在 `~/.wtf2pr/` 下的数据自动迁移到对应工作目录的数据区

---

## 项目结构

```
.
├── cmd/wtf2pr/          # CLI 入口（含 embed 的静态资源）
├── internal/
│   ├── git/             # Git 命令执行与 Unified Diff 解析
│   ├── server/          # Gin HTTP API 与路由
│   ├── review/          # Review 数据存储（内存 + JSON 持久化）
│   └── export/          # Markdown / JSON / XML 格式化导出
├── pkg/models/          # 前后端共享的结构体定义
├── web/                 # Vue 前端项目
└── docs/
    └── system_design.md # 系统设计文档
```

---

## API 简介

> 除静态页面外，所有 API 均需鉴权：`Authorization: Bearer <token>`（或 `?token=<token>` 查询参数）。

- `GET /api/auth/check` — 校验访问令牌是否有效
- `GET /api/diff?type=working` — 获取工作区未提交的 Diff
- `GET /api/diff?type=commit&commit=<hash>` — 获取指定 Commit 的 Diff
- `GET /api/workdir` — 获取当前工作目录与已知工作区列表
- `POST /api/workdir` — 切换工作目录（`{"path": "/path/to/repo"}`，数据区随之切换）
- `GET /api/fs/browse?path=<dir>` — 浏览服务器目录（用于工作目录选择）
- `GET /api/review` — 获取当前 Review 数据
- `POST /api/review` — 保存 Review 数据
- `POST /api/export` — 导出 Review 报告

---

## License

MIT
