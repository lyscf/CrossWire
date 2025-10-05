# CrossWire Makefile

.PHONY: help
help: ## 显示帮助信息
	@echo "CrossWire - CTF Team Communication System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

.PHONY: deps
deps: ## 安装依赖
	@echo "Installing Go dependencies..."
	cd backend && go mod download
	@echo "TODO: Install frontend dependencies"
	# cd frontend && npm install

.PHONY: tidy
tidy: ## 整理依赖
	cd backend && go mod tidy

.PHONY: build-backend
build-backend: ## 编译后端
	@echo "Building backend..."
	cd backend && go build -o ../build/bin/crosswire ./cmd/crosswire

.PHONY: build
build: ## 构建完整应用（TODO: 需要前端）
	@echo "TODO: Build full application with Wails"
	# wails build

.PHONY: dev
dev: ## 启动开发服务器（TODO: 需要前端）
	@echo "TODO: Start development server"
	# wails dev

.PHONY: test
test: ## 运行测试
	@echo "Running tests..."
	cd backend && go test -v ./...

.PHONY: lint
lint: ## 运行代码检查
	@echo "Running linter..."
	cd backend && go vet ./...
	cd backend && go fmt ./...

.PHONY: clean
clean: ## 清理构建产物
	@echo "Cleaning..."
	rm -rf build/bin/
	rm -rf frontend/dist/
	rm -rf .crosswire/
	rm -rf *.db *.db-wal *.db-shm

.PHONY: init-db
init-db: ## 初始化数据库（用于测试）
	@echo "TODO: Initialize database for testing"

.DEFAULT_GOAL := help
