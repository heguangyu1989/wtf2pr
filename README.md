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
# 使用默认参数启动（端口 8080，当前目录作为 Git 工作目录）
./wtf2pr web

# 指定端口与工作目录
./wtf2pr web --port=8080 --workdir=/path/to/your/repo

# 指定 Review 存储文件（默认：~/.wtf2pr/review.json）
./wtf2pr web --review-file=/path/to/review.json

# 使用 Review ID 自动生成文件：~/.wtf2pr/review_feature_x.json
./wtf2pr web --review-id=feature_x
```

### 3. 打开浏览器

访问 `http://localhost:8080`，在左侧选择文件，在右侧 Diff 中点击代码行即可添加 Review。

---

## CLI 参数说明

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--port` | `8080` | 服务监听端口 |
| `--workdir` | 当前目录 | Git 工作目录 |
| `--review-file` | `""` | Review 数据存储的 JSON 文件完整路径 |
| `--review-id` | `""` | Review ID，用于自动生成默认存储文件名 |

> 当 `--review-file` 未指定时，默认路径逻辑为：
> - 若指定了 `--review-id`，则使用 `~/.wtf2pr/review_{id}.json`
> - 否则使用 `~/.wtf2pr/review.json`

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

- `GET /api/diff?type=working` — 获取工作区未提交的 Diff
- `GET /api/diff?type=commit&commit=<hash>` — 获取指定 Commit 的 Diff
- `GET /api/review` — 获取当前 Review 数据
- `POST /api/review` — 保存 Review 数据
- `POST /api/export` — 导出 Review 报告

---

## License

MIT
