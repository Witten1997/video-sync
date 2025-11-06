.PHONY: all build run test clean install-deps docker-build docker-run help

# 变量定义
APP_NAME := bili-download
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
MAIN_PATH := ./cmd/server
BUILD_DIR := ./build
BINARY := $(BUILD_DIR)/$(APP_NAME)

# Go 编译参数
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# 默认目标
all: clean build

# 帮助信息
help:
	@echo "可用的 make 目标:"
	@echo "  make build          - 编译项目"
	@echo "  make run            - 运行项目"
	@echo "  make test           - 运行测试"
	@echo "  make clean          - 清理构建产物"
	@echo "  make install-deps   - 安装依赖"
	@echo "  make docker-build   - 构建 Docker 镜像"
	@echo "  make docker-run     - 运行 Docker 容器"
	@echo "  make fmt            - 格式化代码"
	@echo "  make lint           - 代码检查"
	@echo "  make dev            - 开发模式运行"

# 安装依赖
install-deps:
	@echo "安装 Go 依赖..."
	go mod download
	go mod tidy

# 格式化代码
fmt:
	@echo "格式化代码..."
	go fmt ./...

# 代码检查
lint:
	@echo "代码检查..."
	go vet ./...

# 编译项目
build: install-deps
	@echo "编译项目..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BINARY) $(MAIN_PATH)
	@echo "编译完成: $(BINARY)"

# 编译多平台版本
build-all: clean
	@echo "编译多平台版本..."
	@mkdir -p $(BUILD_DIR)
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(MAIN_PATH)
	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 $(MAIN_PATH)
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe $(MAIN_PATH)
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 $(MAIN_PATH)
	# macOS ARM64
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 $(MAIN_PATH)
	@echo "多平台编译完成"

# 运行项目
run: build
	@echo "运行项目..."
	$(BINARY)

# 开发模式运行（使用 air 热重载）
dev:
	@if ! command -v air > /dev/null; then \
		echo "安装 air..."; \
		go install github.com/air-verse/air@latest; \
	fi
	air

# 运行测试
test:
	@echo "运行测试..."
	go test -v -race -cover ./...

# 运行测试并生成覆盖率报告
test-coverage:
	@echo "运行测试并生成覆盖率报告..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

# 清理构建产物
clean:
	@echo "清理构建产物..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "清理完成"

# 构建 Docker 镜像
docker-build:
	@echo "构建 Docker 镜像..."
	docker build -t $(APP_NAME):$(VERSION) .
	docker tag $(APP_NAME):$(VERSION) $(APP_NAME):latest
	@echo "Docker 镜像构建完成"

# 运行 Docker 容器
docker-run:
	@echo "运行 Docker 容器..."
	docker run -d \
		--name $(APP_NAME) \
		-p 8080:8080 \
		-v $(PWD)/configs:/app/configs \
		-v $(PWD)/downloads:/downloads \
		$(APP_NAME):latest

# 停止 Docker 容器
docker-stop:
	@echo "停止 Docker 容器..."
	docker stop $(APP_NAME)
	docker rm $(APP_NAME)

# 数据库迁移
db-migrate:
	@echo "执行数据库迁移..."
	# TODO: 实现数据库迁移命令

# 生成 API 文档
docs:
	@echo "生成 API 文档..."
	# TODO: 实现 API 文档生成

# 版本信息
version:
	@echo "$(APP_NAME) version: $(VERSION)"
	@echo "Build time: $(BUILD_TIME)"
