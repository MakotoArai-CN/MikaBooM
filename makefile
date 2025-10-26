# MikaBooM Makefile
# 支持多平台、多架构编译

BINARY_NAME := MikaBooM
VERSION := 1.0.0
BUILD_DATE := $(shell date +"%Y-%m-%d %H:%M:%S")
EXPIRE_TIME := $(shell \
	YEAR=$$(date +%Y); \
	MONTH=$$(date +%m); \
	DAY=$$(date +%d); \
	TIME=$$(date +%H:%M:%S); \
	EXPIRE_YEAR=$$((YEAR + 2)); \
	echo "$$EXPIRE_YEAR-$$MONTH-$$DAY $$TIME")
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
OUTPUT_DIR := dist

# 路径
SRC_ICON_DIR := src
DEST_ICON_DIR := internal/tray/assets

# 检测当前系统
CURRENT_OS := $(shell go env GOOS)
CURRENT_ARCH := $(shell go env GOARCH)

# Go 参数
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# LDFLAGS
LDFLAGS := -s -w \
	-X 'MikaBooM/internal/version.Version=$(VERSION)' \
	-X 'MikaBooM/internal/version.BuildDate=$(BUILD_DATE)' \
	-X 'MikaBooM/internal/version.ExpireDate=$(EXPIRE_TIME)' \
	-X 'MikaBooM/internal/version.CommitHash=$(COMMIT_HASH)'

# 默认目标：编译当前系统
.DEFAULT_GOAL := build-current

# 编译当前系统
.PHONY: build-current
build-current: sync-icons
	@echo "Building for current system ($(CURRENT_OS)/$(CURRENT_ARCH))..."
	@mkdir -p $(OUTPUT_DIR)/$(CURRENT_OS)
ifeq ($(CURRENT_OS),windows)
	@$(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/$(CURRENT_OS)/$(BINARY_NAME)-$(CURRENT_OS)-$(CURRENT_ARCH).exe
	@echo "Built: $(OUTPUT_DIR)/$(CURRENT_OS)/$(BINARY_NAME)-$(CURRENT_OS)-$(CURRENT_ARCH).exe"
else
	@$(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/$(CURRENT_OS)/$(BINARY_NAME)-$(CURRENT_OS)-$(CURRENT_ARCH)
	@echo "Built: $(OUTPUT_DIR)/$(CURRENT_OS)/$(BINARY_NAME)-$(CURRENT_OS)-$(CURRENT_ARCH)"
endif
	@echo ""
	@echo "Tip: Use 'make build-all' to build for all platforms"

# 编译所有平台
.PHONY: all
all: clean deps sync-icons test build-all

# 清理
.PHONY: clean
clean:
	@echo "Cleaning..."
	@$(GOCLEAN)
	@rm -rf $(OUTPUT_DIR)
	@rm -f $(BINARY_NAME) $(BINARY_NAME).exe

# 下载依赖
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	@$(GOMOD) download
	@$(GOMOD) tidy

# 同步图标文件
.PHONY: sync-icons
sync-icons:
	@echo "Syncing icon files..."
	@mkdir -p $(DEST_ICON_DIR)
	@if [ -f "$(SRC_ICON_DIR)/icon.ico" ]; then \
		cp $(SRC_ICON_DIR)/icon.ico $(DEST_ICON_DIR)/icon_windows.ico && \
		echo "  ✓ Copied: icon_windows.ico"; \
	else \
		echo "  ⚠ Warning: $(SRC_ICON_DIR)/icon.ico not found"; \
	fi
	@if [ -f "$(SRC_ICON_DIR)/icon.png" ]; then \
		cp $(SRC_ICON_DIR)/icon.png $(DEST_ICON_DIR)/icon_linux.png && \
		cp $(SRC_ICON_DIR)/icon.png $(DEST_ICON_DIR)/icon_macos.png && \
		echo "  ✓ Copied: icon_linux.png" && \
		echo "  ✓ Copied: icon_macos.png"; \
	else \
		echo "  ⚠ Warning: $(SRC_ICON_DIR)/icon.png not found"; \
	fi

# 测试
.PHONY: test
test:
	@echo "Running tests..."
	@$(GOTEST) -v ./...

# 创建输出目录
.PHONY: prepare
prepare:
	@mkdir -p $(OUTPUT_DIR)/windows
	@mkdir -p $(OUTPUT_DIR)/linux
	@mkdir -p $(OUTPUT_DIR)/darwin
	@mkdir -p $(OUTPUT_DIR)/freebsd
	@mkdir -p $(OUTPUT_DIR)/android

# 编译所有平台
.PHONY: build-all
build-all: prepare sync-icons build-windows build-linux build-darwin build-bsd build-android
	@echo ""
	@echo "All platforms built successfully!"

# Windows 编译
.PHONY: build-windows
build-windows:
	@echo "🪟 Building Windows..."
	@GOOS=windows GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/windows/$(BINARY_NAME)-windows-amd64.exe
	@GOOS=windows GOARCH=386 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/windows/$(BINARY_NAME)-windows-386.exe
	@GOOS=windows GOARCH=arm64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/windows/$(BINARY_NAME)-windows-arm64.exe
	@GOOS=windows GOARCH=arm GOARM=7 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/windows/$(BINARY_NAME)-windows-armv7.exe

# Linux 编译
.PHONY: build-linux
build-linux:
	@echo "Building Linux..."
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/linux/$(BINARY_NAME)-linux-amd64
	@GOOS=linux GOARCH=386 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/linux/$(BINARY_NAME)-linux-386
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/linux/$(BINARY_NAME)-linux-arm64
	@GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/linux/$(BINARY_NAME)-linux-armv7
	@GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/linux/$(BINARY_NAME)-linux-armv6
	@GOOS=linux GOARCH=arm GOARM=5 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/linux/$(BINARY_NAME)-linux-armv5
	@GOOS=linux GOARCH=mips CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/linux/$(BINARY_NAME)-linux-mips
	@GOOS=linux GOARCH=mipsle CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/linux/$(BINARY_NAME)-linux-mipsle
	@GOOS=linux GOARCH=mips64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/linux/$(BINARY_NAME)-linux-mips64
	@GOOS=linux GOARCH=mips64le CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/linux/$(BINARY_NAME)-linux-mips64le
	@GOOS=linux GOARCH=ppc64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/linux/$(BINARY_NAME)-linux-ppc64
	@GOOS=linux GOARCH=ppc64le CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/linux/$(BINARY_NAME)-linux-ppc64le
	@GOOS=linux GOARCH=riscv64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/linux/$(BINARY_NAME)-linux-riscv64
	@GOOS=linux GOARCH=s390x CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/linux/$(BINARY_NAME)-linux-s390x

# macOS 编译
.PHONY: build-darwin
build-darwin:
	@echo "Building macOS..."
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/darwin/$(BINARY_NAME)-darwin-amd64
	@GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/darwin/$(BINARY_NAME)-darwin-arm64

# BSD 编译
.PHONY: build-bsd
build-bsd:
	@echo "Building BSD..."
	@GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/freebsd/$(BINARY_NAME)-freebsd-amd64
	@GOOS=freebsd GOARCH=386 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/freebsd/$(BINARY_NAME)-freebsd-386
	@GOOS=freebsd GOARCH=arm64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/freebsd/$(BINARY_NAME)-freebsd-arm64

# Android 编译
.PHONY: build-android
build-android:
	@echo "Building Android..."
	@GOOS=android GOARCH=arm64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/android/$(BINARY_NAME)-android-arm64
	@GOOS=android GOARCH=arm GOARM=7 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/android/$(BINARY_NAME)-android-armv7

# 快速编译（当前目录，不输出到dist）
.PHONY: build
build: sync-icons
	@echo "Quick build for current system..."
ifeq ($(CURRENT_OS),windows)
	@$(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME).exe
	@echo "Built: $(BINARY_NAME).exe"
else
	@$(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME)
	@echo "Built: $(BINARY_NAME)"
endif

# 运行
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	@echo ""
ifeq ($(CURRENT_OS),windows)
	@./$(BINARY_NAME).exe
else
	@./$(BINARY_NAME)
endif

# 安装到 GOPATH/bin
.PHONY: install
install: sync-icons
	@echo "Installing to $(GOPATH)/bin/$(BINARY_NAME)..."
	@$(GOBUILD) -ldflags="$(LDFLAGS)" -o $(GOPATH)/bin/$(BINARY_NAME)
	@echo "Installed successfully!"
	@echo ""
	@echo "You can now run: $(BINARY_NAME)"

# 显示当前系统信息
.PHONY: info
info:
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "Build Information"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "Binary Name:     $(BINARY_NAME)"
	@echo "Version:         $(VERSION)"
	@echo "Build Date:      $(BUILD_DATE)"
	@echo "Commit Hash:     $(COMMIT_HASH)"
	@echo ""
	@echo "Current System:  $(CURRENT_OS)/$(CURRENT_ARCH)"
	@echo "Go Version:      $(shell go version)"
	@echo "Output Dir:      $(OUTPUT_DIR)"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# 帮助
.PHONY: help
help:
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "MikaBooM Makefile - Help"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@echo "Building Commands:"
	@echo "  make                - Build for current system only (default)"
	@echo "  make build          - Quick build in current directory"
	@echo "  make build-current  - Build current system to dist/"
	@echo "  make build-all      - Build for all platforms"
	@echo "  make all            - Clean, test and build all platforms"
	@echo ""
	@echo "Platform-Specific Builds:"
	@echo "  make build-windows  - Build for Windows (all architectures)"
	@echo "  make build-linux    - Build for Linux (all architectures)"
	@echo "  make build-darwin   - Build for macOS (all architectures)"
	@echo "  make build-bsd      - Build for BSD (all architectures)"
	@echo "  make build-android  - Build for Android (all architectures)"
	@echo ""
	@echo "Utility Commands:"
	@echo "  make clean          - Clean build files"
	@echo "  make deps           - Download dependencies"
	@echo "  make sync-icons     - Sync icon files from src/ to internal/tray/assets/"
	@echo "  make test           - Run tests"
	@echo "  make run            - Build and run"
	@echo "  make install        - Install to GOPATH/bin"
	@echo "  make info           - Show build information"
	@echo "  make help           - Show this help"
	@echo ""
	@echo "Examples:"
	@echo "  make                # Build for current system ($(CURRENT_OS)/$(CURRENT_ARCH))"
	@echo "  make build          # Quick build for testing"
	@echo "  make run            # Build and run immediately"
	@echo "  make build-all      # Build for all platforms"
	@echo "  make clean build    # Clean and rebuild current system"
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"