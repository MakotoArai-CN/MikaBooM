#!/bin/bash

# MikaBooM 完整交叉编译脚本
# 支持多平台、多架构的CLI版本

set -e

# 版本信息
VERSION="1.0.0"
BUILD_DATE=$(date +"%Y-%m-%d %H:%M:%S")
COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
OUTPUT_DIR="dist"
BINARY_NAME="MikaBooM"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

# 打印带颜色的信息
print_info() {
    echo -e "${CYAN}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

# 显示标题
show_banner() {
    echo -e "${MAGENTA}"
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║           MikaBooM Cross-Platform Build Script            ║"
    echo "║                      CLI Edition                          ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    echo -e "${CYAN}Version:${NC} ${VERSION}"
    echo -e "${CYAN}Date:${NC} ${BUILD_DATE}"
    echo -e "${CYAN}Commit:${NC} ${COMMIT_HASH}"
    echo ""
}

# 检查依赖
check_dependencies() {
    print_info "检查编译依赖..."
    
    if ! command -v go &> /dev/null; then
        print_error "未找到 Go 编译器"
        echo "请安装 Go: https://golang.org/dl/"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}')
    print_success "Go 版本: ${GO_VERSION}"
    
    # 检查 git（可选）
    if command -v git &> /dev/null; then
        GIT_VERSION=$(git --version | awk '{print $3}')
        print_success "Git 版本: ${GIT_VERSION}"
    else
        print_warning "未找到 Git，将无法获取提交信息"
    fi
    
    echo ""
}

# 创建输出目录
setup_directories() {
    print_info "创建输出目录..."
    mkdir -p ${OUTPUT_DIR}
    mkdir -p ${OUTPUT_DIR}/windows
    mkdir -p ${OUTPUT_DIR}/linux
    mkdir -p ${OUTPUT_DIR}/darwin
    mkdir -p ${OUTPUT_DIR}/freebsd
    mkdir -p ${OUTPUT_DIR}/openbsd
    mkdir -p ${OUTPUT_DIR}/android
    print_success "目录创建完成"
    echo ""
}

# 同步图标文件
sync_icons() {
    print_info "同步图标文件..."
    
    local SRC_DIR="src"
    local DEST_DIR="internal/tray/assets"
    
    # 创建目标目录
    mkdir -p "${DEST_DIR}"
    
    # 检查源图标是否存在
    if [ ! -f "${SRC_DIR}/icon.ico" ] && [ ! -f "${SRC_DIR}/icon.png" ]; then
        print_warning "未找到源图标文件 (src/icon.ico 或 src/icon.png)"
        print_warning "将使用默认图标或现有图标"
        echo ""
        return 0
    fi
    
    # 复制并重命名图标文件
    local copied=false
    
    if [ -f "${SRC_DIR}/icon.ico" ]; then
        cp "${SRC_DIR}/icon.ico" "${DEST_DIR}/icon_windows.ico"
        print_success "已复制: icon_windows.ico"
        copied=true
    fi
    
    if [ -f "${SRC_DIR}/icon.png" ]; then
        cp "${SRC_DIR}/icon.png" "${DEST_DIR}/icon_linux.png"
        cp "${SRC_DIR}/icon.png" "${DEST_DIR}/icon_macos.png"
        print_success "已复制: icon_linux.png"
        print_success "已复制: icon_macos.png"
        copied=true
    fi
    
    # 如果只有 ico 文件，尝试创建 png（需要 ImageMagick）
    if [ -f "${SRC_DIR}/icon.ico" ] && [ ! -f "${SRC_DIR}/icon.png" ]; then
        if command -v convert &> /dev/null; then
            print_info "检测到 ImageMagick，转换 ICO 到 PNG..."
            if convert "${SRC_DIR}/icon.ico[0]" -resize 256x256 "${DEST_DIR}/icon_linux.png" 2>/dev/null; then
                cp "${DEST_DIR}/icon_linux.png" "${DEST_DIR}/icon_macos.png"
                print_success "已转换并复制 PNG 图标"
            else
                print_warning "ICO 转换失败，Linux 和 macOS 可能无法显示图标"
            fi
        else
            print_warning "未安装 ImageMagick，无法转换 ICO 到 PNG"
            print_warning "Linux 和 macOS 可能无法显示图标"
            print_info "建议: 安装 ImageMagick (apt/yum/brew install imagemagick)"
        fi
    fi
    
    if [ "$copied" = true ]; then
        print_success "图标同步完成"
    fi
    
    echo ""
}

# 设置 LDFLAGS
get_ldflags() {
    local BUILD_TIME=$(date +"%Y-%m-%d %H:%M:%S")
    
    # 使用纯 bash 计算
    local YEAR=$(date +%Y)
    local MONTH=$(date +%m)
    local DAY=$(date +%d)
    local TIME=$(date +%H:%M:%S)
    
    local EXPIRE_YEAR=$((YEAR + 2))
    local EXPIRE_TIME="${EXPIRE_YEAR}-${MONTH}-${DAY} ${TIME}"
    
    echo "-s -w -X 'MikaBooM/internal/version.Version=${VERSION}' -X 'MikaBooM/internal/version.BuildDate=${BUILD_TIME}' -X 'MikaBooM/internal/version.ExpireDate=${EXPIRE_TIME}'"
}

# 编译函数
build_target() {
    local GOOS=$1
    local GOARCH=$2
    local ARM_VERSION=$3
    local EXTRA_NAME=$4
    
    # 构建输出文件名
    local OUTPUT_NAME="${BINARY_NAME}-${GOOS}-${GOARCH}"
    if [ -n "${ARM_VERSION}" ]; then
        OUTPUT_NAME="${OUTPUT_NAME}v${ARM_VERSION}"
    fi
    if [ -n "${EXTRA_NAME}" ]; then
        OUTPUT_NAME="${OUTPUT_NAME}-${EXTRA_NAME}"
    fi
    
    # Windows 需要 .exe 后缀
    if [ "${GOOS}" = "windows" ]; then
        OUTPUT_NAME="${OUTPUT_NAME}.exe"
    fi
    
    local OUTPUT_PATH="${OUTPUT_DIR}/${GOOS}/${OUTPUT_NAME}"
    
    # 设置环境变量
    export GOOS=${GOOS}
    export GOARCH=${GOARCH}
    export CGO_ENABLED=0
    
    if [ -n "${ARM_VERSION}" ]; then
        export GOARM=${ARM_VERSION}
    fi
    
    # 编译
    print_info "编译 ${GOOS}/${GOARCH}${ARM_VERSION:+v$ARM_VERSION}${EXTRA_NAME:+ ($EXTRA_NAME)}..."
    
    if go build -ldflags="$(get_ldflags)" -o "${OUTPUT_PATH}" 2>/dev/null; then
        # 获取文件大小
        if [ -f "${OUTPUT_PATH}" ]; then
            SIZE=$(du -h "${OUTPUT_PATH}" | cut -f1)
            print_success "${OUTPUT_NAME} (${SIZE})"
            return 0
        fi
    fi
    
    print_error "编译失败: ${GOOS}/${GOARCH}${ARM_VERSION:+v$ARM_VERSION}"
    return 1
}

# Windows 平台编译
build_windows() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}🪟 Windows 平台${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    build_target "windows" "amd64" "" ""           # 64位 x86-64
    build_target "windows" "386" "" ""             # 32位 x86
    build_target "windows" "arm64" "" ""           # ARM64
    build_target "windows" "arm" "7" ""            # ARM32 v7
    
    echo ""
}

# Linux 平台编译
build_linux() {
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}🐧 Linux 平台${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    # x86/x64 架构
    build_target "linux" "amd64" "" ""             # 64位 x86-64
    build_target "linux" "386" "" ""               # 32位 x86
    
    # ARM 架构
    build_target "linux" "arm64" "" ""             # ARM64 (ARMv8)
    build_target "linux" "arm" "7" ""              # ARM32 v7 (树莓派2/3)
    build_target "linux" "arm" "6" ""              # ARM32 v6 (树莓派1)
    build_target "linux" "arm" "5" ""              # ARM32 v5 (旧设备)
    
    # MIPS 架构（路由器）
    build_target "linux" "mips" "" ""              # MIPS 大端
    build_target "linux" "mipsle" "" ""            # MIPS 小端
    build_target "linux" "mips64" "" ""            # MIPS64 大端
    build_target "linux" "mips64le" "" ""          # MIPS64 小端
    
    # PowerPC 架构
    build_target "linux" "ppc64" "" ""             # PowerPC 64位 大端
    build_target "linux" "ppc64le" "" ""           # PowerPC 64位 小端
    
    # RISC-V 架构
    build_target "linux" "riscv64" "" ""           # RISC-V 64位
    
    # S390X 架构（IBM大型机）
    build_target "linux" "s390x" "" ""             # IBM S390X
    
    # LoongArch 架构（龙芯）
    build_target "linux" "loong64" "" ""           # 龙芯 64位
    
    echo ""
}

# macOS 平台编译
build_darwin() {
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}🍎 macOS 平台${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    build_target "darwin" "amd64" "" ""            # Intel Mac
    build_target "darwin" "arm64" "" ""            # Apple Silicon (M1/M2/M3)
    
    echo ""
}

# BSD 平台编译
build_bsd() {
    echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${MAGENTA}👹 BSD 平台${NC}"
    echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    # FreeBSD
    build_target "freebsd" "amd64" "" ""           # FreeBSD 64位
    build_target "freebsd" "386" "" ""             # FreeBSD 32位
    build_target "freebsd" "arm64" "" ""           # FreeBSD ARM64
    build_target "freebsd" "arm" "7" ""            # FreeBSD ARM32
    
    # OpenBSD
    build_target "openbsd" "amd64" "" ""           # OpenBSD 64位
    build_target "openbsd" "386" "" ""             # OpenBSD 32位
    build_target "openbsd" "arm64" "" ""           # OpenBSD ARM64
    build_target "openbsd" "arm" "7" ""            # OpenBSD ARM32
    
    # NetBSD
    build_target "netbsd" "amd64" "" ""            # NetBSD 64位
    build_target "netbsd" "386" "" ""              # NetBSD 32位
    build_target "netbsd" "arm64" "" ""            # NetBSD ARM64
    
    echo ""
}

# Android 平台编译
build_android() {
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}🤖 Android 平台${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    build_target "android" "arm64" "" ""           # Android ARM64
    build_target "android" "arm" "7" ""            # Android ARM32
    build_target "android" "amd64" "" ""           # Android x86-64 (模拟器)
    build_target "android" "386" "" ""             # Android x86 (模拟器)
    
    echo ""
}

# 其他平台编译
build_others() {
    echo -e "${WHITE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${WHITE}🌐 其他平台${NC}"
    echo -e "${WHITE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    # Solaris
    build_target "solaris" "amd64" "" ""           # Solaris x86-64
    
    # AIX
    build_target "aix" "ppc64" "" ""               # IBM AIX PowerPC
    
    # Plan 9
    build_target "plan9" "amd64" "" ""             # Plan 9 x86-64
    build_target "plan9" "386" "" ""               # Plan 9 x86
    build_target "plan9" "arm" "" ""               # Plan 9 ARM
    
    echo ""
}

# 创建压缩包
create_archives() {
    print_info "创建压缩包..."
    
    cd ${OUTPUT_DIR}
    
    # 为每个平台创建 tar.gz 压缩包
    for platform in windows linux darwin freebsd openbsd android; do
        if [ -d "${platform}" ] && [ "$(ls -A ${platform})" ]; then
            tar -czf "${BINARY_NAME}-${VERSION}-${platform}.tar.gz" ${platform}/
            print_success "创建 ${BINARY_NAME}-${VERSION}-${platform}.tar.gz"
        fi
    done
    
    # 创建一个包含所有平台的总压缩包
    if [ "$(ls -A .)" ]; then
        tar -czf "${BINARY_NAME}-${VERSION}-all.tar.gz" */
        print_success "创建 ${BINARY_NAME}-${VERSION}-all.tar.gz"
    fi
    
    cd ..
    echo ""
}

# 生成 SHA256 校验和
generate_checksums() {
    print_info "生成 SHA256 校验和..."
    
    cd ${OUTPUT_DIR}
    
    # 生成每个文件的校验和
    find . -type f \( -name "*.exe" -o -name "${BINARY_NAME}-*" \) ! -name "*.tar.gz" -exec sha256sum {} \; > SHA256SUMS.txt
    
    print_success "校验和已保存到 SHA256SUMS.txt"
    
    cd ..
    echo ""
}

# 显示编译统计
show_statistics() {
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}📊 编译统计${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    # 统计各平台文件数量
    for platform in windows linux darwin freebsd openbsd netbsd android; do
        if [ -d "${OUTPUT_DIR}/${platform}" ]; then
            COUNT=$(find ${OUTPUT_DIR}/${platform} -type f | wc -l)
            if [ ${COUNT} -gt 0 ]; then
                SIZE=$(du -sh ${OUTPUT_DIR}/${platform} | cut -f1)
                printf "%-10s: %2d 个文件, 总大小: %s\n" ${platform} ${COUNT} ${SIZE}
            fi
        fi
    done
    
    echo ""
    
    # 总统计
    TOTAL_FILES=$(find ${OUTPUT_DIR} -type f \( -name "*.exe" -o -name "${BINARY_NAME}-*" \) ! -name "*.tar.gz" | wc -l)
    TOTAL_SIZE=$(du -sh ${OUTPUT_DIR} | cut -f1)
    
    echo -e "${GREEN}总计: ${TOTAL_FILES} 个可执行文件${NC}"
    echo -e "${GREEN}总大小: ${TOTAL_SIZE}${NC}"
    echo ""
}

# 清理函数
cleanup() {
    print_info "清理临时文件..."
    rm -f *.syso 2>/dev/null
    print_success "清理完成"
}

# 选择性编译
selective_build() {
    echo "请选择要编译的平台："
    echo "1) Windows"
    echo "2) Linux"
    echo "3) macOS"
    echo "4) BSD系列"
    echo "5) Android"
    echo "6) 其他平台"
    echo "7) 全部平台"
    echo "0) 退出"
    
    read -p "请输入选项 (0-7): " choice
    
    case $choice in
        1) build_windows ;;
        2) build_linux ;;
        3) build_darwin ;;
        4) build_bsd ;;
        5) build_android ;;
        6) build_others ;;
        7) 
            build_windows
            build_linux
            build_darwin
            build_bsd
            build_android
            build_others
            ;;
        0) 
            echo "退出编译"
            exit 0
            ;;
        *)
            print_error "无效选项"
            exit 1
            ;;
    esac
}

# 主函数
main() {
    # 显示标题
    show_banner
    
    # 检查依赖
    check_dependencies
    
    # 创建目录
    setup_directories
    
    # 同步图标文件
    sync_icons
    
    # 检查是否有参数
    if [ $# -eq 0 ]; then
        # 无参数，进入交互模式
        selective_build
    else
        # 有参数，根据参数编译
        case "$1" in
            all)
                build_windows
                build_linux
                build_darwin
                build_bsd
                build_android
                build_others
                ;;
            windows) build_windows ;;
            linux) build_linux ;;
            darwin|macos) build_darwin ;;
            bsd) build_bsd ;;
            android) build_android ;;
            others) build_others ;;
            *)
                print_error "未知参数: $1"
                echo "用法: $0 [all|windows|linux|darwin|bsd|android|others]"
                exit 1
                ;;
        esac
    fi
    
    # 创建压缩包
    create_archives
    
    # 生成校验和
    generate_checksums
    
    # 显示统计
    show_statistics
    
    # 清理
    cleanup
    
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}✅ 编译完成！${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "输出目录: ${OUTPUT_DIR}/"
    echo "压缩包已创建，可用于分发"
    echo ""
}

# 运行主函数
main "$@"