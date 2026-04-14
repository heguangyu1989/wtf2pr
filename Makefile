.PHONY: dev-web dev-server ensure-dist embed build clean lint lint-go lint-web

WEB_DIR := web
DIST_DIR := cmd/wtf2pr/dist
BINARY := wtf2pr
BUILD_DIR := build

## dev-web: 以开发模式运行前端（通过 Vite proxy 访问后端 API）
##   可覆盖环境变量：DEV_HOST, DEV_PORT
##
dev-web:
	@echo "Starting frontend dev server on $$(DEV_HOST:=0.0.0.0):$$(DEV_PORT:=8323)..."
	@cd $(WEB_DIR) && DEV_HOST=$(DEV_HOST) DEV_PORT=$(DEV_PORT) npm run dev

## ensure-dist: 保证 embed 目录非空，避免 go:embed 编译失败
##
ensure-dist:
	@if [ ! -d "$(DIST_DIR)" ] || [ -z "$$(ls -A $(DIST_DIR) 2>/dev/null)" ]; then \
		mkdir -p $(DIST_DIR); \
		echo '<!doctype html><html><body>API Server Only</body></html>' > $(DIST_DIR)/index.html; \
	fi

## dev-server: 开发模式下运行后端（不重新构建前端）
##
dev-server: ensure-dist
	@echo "Starting backend server..."
	@go run ./cmd/wtf2pr web --workdir=. --port=8322 --host=0.0.0.0

## embed: 构建前端并将产物嵌入 Go 二进制
##
embed:
	@echo "Building frontend and embedding into Go binary..."
	@cd $(WEB_DIR) && npm run build
	@rm -rf $(DIST_DIR)
	@cp -r $(WEB_DIR)/dist $(DIST_DIR)

## build: 构建 macOS 版本最终交付物（amd64 + arm64）
##
build: embed
	@echo "Building macOS binaries..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 ./cmd/wtf2pr
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 ./cmd/wtf2pr
	@echo "Packaging deliverables..."
	@tar -czvf $(BUILD_DIR)/$(BINARY)-darwin.tar.gz -C $(BUILD_DIR) $(BINARY)-darwin-amd64 $(BINARY)-darwin-arm64
	@echo "Build complete: $(BUILD_DIR)/$(BINARY)-darwin.tar.gz"

## lint-go: 后端代码检查（优先 golangci-lint，否则 go vet + gofmt）
##
lint-go:
	@echo "==> Linting Go code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found, falling back to go vet"; \
		go vet ./...; \
	fi
	@echo "==> Checking Go format..."
	@test -z "$$(gofmt -l .)" || (echo "Please run gofmt on:" && gofmt -l . && exit 1)

## lint-web: 前端代码检查
##
lint-web:
	@echo "==> Linting frontend code..."
	@cd $(WEB_DIR) && npm run lint

## lint: 同时检查前后端代码
##
lint: lint-go lint-web

## clean: 清理构建产物
##
clean:
	@rm -rf $(DIST_DIR) $(BUILD_DIR)
