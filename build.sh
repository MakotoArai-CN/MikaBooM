#!/bin/bash
# MikaBooM 完整交叉编译脚本
# 支持多平台、多架构的CLI版本

# 注释掉 set -e，因为我们要手动处理错误
# set -e

# 版本信息
VERSION="1.0.0"
BUILD_DATE=$(date +"%Y-%m-%d %H:%M:%S")
COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
OUTPUT_DIR="dist"
BINARY_NAME="MikaBooM"

# 编译统计
BUILD_SUCCESS=0
BUILD_FAILED=0
FAILED_TARGETS=()

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
    
    # 检查系统特定依赖
    case "$(uname -s)" in
        Linux*)
            print_info "检测到 Linux 系统"
            # 检查是否安装了必要的开发库
            if ! ldconfig -p 2>/dev/null | grep -q libappindicator; then
                print_warning "未检测到 libappindicator，系统托盘可能无法工作"
                print_info "建议安装: sudo apt-get install libappindicator3-dev libgtk-3-dev"
            fi
            ;;
        Darwin*)
            print_info "检测到 macOS 系统"
            ;;
        MINGW*|MSYS*|CYGWIN*)
            print_info "检测到 Windows 系统"
            ;;
    esac
    
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
    mkdir -p ${OUTPUT_DIR}/netbsd
    mkdir -p ${OUTPUT_DIR}/android
    mkdir -p ${OUTPUT_DIR}/solaris
    mkdir -p ${OUTPUT_DIR}/aix
    mkdir -p ${OUTPUT_DIR}/plan9
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
    
    # 使用纯 bash 计算过期时间（2年后）
    local YEAR=$(date +%Y)
    local MONTH=$(date +%m)
    local DAY=$(date +%d)
    local TIME=$(date +%H:%M:%S)
    local EXPIRE_YEAR=$((YEAR + 2))
    local EXPIRE_TIME="${EXPIRE_YEAR}-${MONTH}-${DAY} ${TIME}"
    
    echo "-s -w -X 'MikaBooM/internal/version.Version=${VERSION}' -X 'MikaBooM/internal/version.BuildDate=${BUILD_TIME}' -X 'MikaBooM/internal/version.ExpireDate=${EXPIRE_TIME}'"
}

# 编译函数（带错误处理）
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
    
    # 根据平台设置 CGO
    if [ "${GOOS}" = "windows" ]; then
        export CGO_ENABLED=0
    elif [ "${GOOS}" = "linux" ] || [ "${GOOS}" = "darwin" ]; then
        # 检查是否在目标平台上编译
        CURRENT_OS=$(uname -s | tr '[:upper:]' '[:lower:]')
        if [[ "${CURRENT_OS}" == *"${GOOS}"* ]]; then
            export CGO_ENABLED=1
        else
            # 跨平台编译，禁用 CGO
            export CGO_ENABLED=0
        fi
    else
        export CGO_ENABLED=0
    fi
    
    if [ -n "${ARM_VERSION}" ]; then
        export GOARM=${ARM_VERSION}
    else
        unset GOARM
    fi
    
    # 构建目标名称用于显示
    local TARGET_NAME="${GOOS}/${GOARCH}"
    if [ -n "${ARM_VERSION}" ]; then
        TARGET_NAME="${TARGET_NAME}v${ARM_VERSION}"
    fi
    if [ -n "${EXTRA_NAME}" ]; then
        TARGET_NAME="${TARGET_NAME} (${EXTRA_NAME})"
    fi
    
    # 编译（显示详细错误）
    print_info "编译 ${TARGET_NAME} [CGO=${CGO_ENABLED}]..."
    
    # 临时文件存储错误信息
    local ERROR_LOG=$(mktemp)
    
    # 执行编译，捕获错误
    if go build -ldflags="$(get_ldflags)" -o "${OUTPUT_PATH}" 2>"${ERROR_LOG}"; then
        # 编译成功，检查文件是否存在
        if [ -f "${OUTPUT_PATH}" ]; then
            SIZE=$(du -h "${OUTPUT_PATH}" | cut -f1)
            print_success "${OUTPUT_NAME} (${SIZE})"
            rm -f "${ERROR_LOG}"
            BUILD_SUCCESS=$((BUILD_SUCCESS + 1))
            return 0
        else
            print_error "编译失败: ${TARGET_NAME} - 输出文件不存在"
            BUILD_FAILED=$((BUILD_FAILED + 1))
            FAILED_TARGETS+=("${TARGET_NAME}")
            rm -f "${ERROR_LOG}"
            return 1
        fi
    else
        # 编译失败
        print_error "编译失败: ${TARGET_NAME}"
        
        # 显示错误详情（只显示前20行，避免刷屏）
        if [ -s "${ERROR_LOG}" ]; then
            echo -e "${RED}错误详情:${NC}"
            head -20 "${ERROR_LOG}" | sed 's/^/  /'
            local LINE_COUNT=$(wc -l < "${ERROR_LOG}")
            if [ "${LINE_COUNT}" -gt 20 ]; then
                echo -e "${YELLOW}  ... (还有 $((LINE_COUNT - 20)) 行错误信息被省略)${NC}"
            fi
        fi
        echo ""
        
        rm -f "${ERROR_LOG}"
        BUILD_FAILED=$((BUILD_FAILED + 1))
        FAILED_TARGETS+=("${TARGET_NAME}")
        return 1
    fi
}

# Windows 平台编译
build_windows() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}🪟 Windows 平台${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    build_target "windows" "amd64" "" "" || true
    build_target "windows" "386" "" "" || true
    build_target "windows" "arm64" "" "" || true
    build_target "windows" "arm" "7" "" || true
    
    echo ""
}

# Linux 平台编译
build_linux() {
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}🐧 Linux 平台${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    # 检查是否在 Linux 上编译
    CURRENT_OS=$(uname -s)
    if [ "${CURRENT_OS}" != "Linux" ]; then
        print_warning "非 Linux 系统编译 Linux 版本，系统托盘功能将不可用"
        print_warning "建议在 Linux 系统上编译或使用 Docker"
    fi
    
    # x86/x64 架构
    build_target "linux" "amd64" "" "" || true
    build_target "linux" "386" "" "" || true
    
    # ARM 架构
    build_target "linux" "arm64" "" "" || true
    build_target "linux" "arm" "7" "" || true
    build_target "linux" "arm" "6" "" || true
    build_target "linux" "arm" "5" "" || true
    
    # MIPS 架构（路由器）
    build_target "linux" "mips" "" "" || true
    build_target "linux" "mipsle" "" "" || true
    build_target "linux" "mips64" "" "" || true
    build_target "linux" "mips64le" "" "" || true
    
    # PowerPC 架构
    build_target "linux" "ppc64" "" "" || true
    build_target "linux" "ppc64le" "" "" || true
    
    # RISC-V 架构
    build_target "linux" "riscv64" "" "" || true
    
    # S390X 架构（IBM大型机）
    build_target "linux" "s390x" "" "" || true
    
    # LoongArch 架构（龙芯）
    build_target "linux" "loong64" "" "" || true
    
    echo ""
}

# macOS 平台编译
build_darwin() {
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}🍎 macOS 平台${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    # 检查是否在 macOS 上编译
    CURRENT_OS=$(uname -s)
    if [ "${CURRENT_OS}" != "Darwin" ]; then
        print_warning "非 macOS 系统编译 macOS 版本，系统托盘功能将不可用"
        print_warning "建议在 macOS 系统上编译"
    fi
    
    build_target "darwin" "amd64" "" "" || true
    build_target "darwin" "arm64" "" "" || true
    
    echo ""
}

# BSD 平台编译
build_bsd() {
    echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${MAGENTA}👹 BSD 平台${NC}"
    echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    # FreeBSD
    build_target "freebsd" "amd64" "" "" || true
    build_target "freebsd" "386" "" "" || true
    build_target "freebsd" "arm64" "" "" || true
    build_target "freebsd" "arm" "7" "" || true
    
    # OpenBSD
    build_target "openbsd" "amd64" "" "" || true
    build_target "openbsd" "386" "" "" || true
    build_target "openbsd" "arm64" "" "" || true
    build_target "openbsd" "arm" "7" "" || true
    
    # NetBSD
    build_target "netbsd" "amd64" "" "" || true
    build_target "netbsd" "386" "" "" || true
    build_target "netbsd" "arm64" "" "" || true
    
    echo ""
}

# Android 平台编译
build_android() {
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}🤖 Android 平台${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    build_target "android" "arm64" "" "" || true
    build_target "android" "arm" "7" "" || true
    build_target "android" "amd64" "" "" || true
    build_target "android" "386" "" "" || true
    
    echo ""
}

# 其他平台编译
build_others() {
    echo -e "${WHITE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${WHITE}🌐 其他平台${NC}"
    echo -e "${WHITE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    # Solaris
    build_target "solaris" "amd64" "" "" || true
    
    # AIX
    build_target "aix" "ppc64" "" "" || true
    
    # Plan 9
    build_target "plan9" "amd64" "" "" || true
    build_target "plan9" "386" "" "" || true
    build_target "plan9" "arm" "" "" || true
    
    echo ""
}

# 创建压缩包
create_archives() {
    print_info "创建压缩包..."
    
    cd ${OUTPUT_DIR}
    
    local archive_count=0
    
    # 为每个平台创建 tar.gz 压缩包
    for platform in windows linux darwin freebsd openbsd netbsd android solaris aix plan9; do
        if [ -d "${platform}" ] && [ "$(ls -A ${platform} 2>/dev/null)" ]; then
            if tar -czf "${BINARY_NAME}-${VERSION}-${platform}.tar.gz" ${platform}/ 2>/dev/null; then
                print_success "创建 ${BINARY_NAME}-${VERSION}-${platform}.tar.gz"
                archive_count=$((archive_count + 1))
            else
                print_warning "创建 ${platform} 压缩包失败"
            fi
        fi
    done
    
    # 创建一个包含所有平台的总压缩包
    if [ ${archive_count} -gt 0 ]; then
        if tar -czf "${BINARY_NAME}-${VERSION}-all.tar.gz" */ 2>/dev/null; then
            print_success "创建 ${BINARY_NAME}-${VERSION}-all.tar.gz"
        fi
    else
        print_warning "没有成功编译的文件，跳过创建总压缩包"
    fi
    
    cd ..
    echo ""
}

# 生成 SHA256 校验和
generate_checksums() {
    print_info "生成 SHA256 校验和..."
    
    cd ${OUTPUT_DIR}
    
    # 查找所有可执行文件
    local file_count=$(find . -type f \( -name "*.exe" -o -name "${BINARY_NAME}-*" \) ! -name "*.tar.gz" ! -name "SHA256SUMS.txt" 2>/dev/null | wc -l)
    
    if [ ${file_count} -gt 0 ]; then
        # 生成每个文件的校验和
        find . -type f \( -name "*.exe" -o -name "${BINARY_NAME}-*" \) ! -name "*.tar.gz" ! -name "SHA256SUMS.txt" -exec sha256sum {} \; > SHA256SUMS.txt 2>/dev/null
        print_success "校验和已保存到 SHA256SUMS.txt (${file_count} 个文件)"
    else
        print_warning "没有可执行文件，跳过生成校验和"
    fi
    
    cd ..
    echo ""
}

# 显示编译统计
show_statistics() {
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}📊 编译统计${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    # 统计各平台文件数量
    local has_output=false
    for platform in windows linux darwin freebsd openbsd netbsd android solaris aix plan9; do
        if [ -d "${OUTPUT_DIR}/${platform}" ]; then
            COUNT=$(find ${OUTPUT_DIR}/${platform} -type f 2>/dev/null | wc -l)
            if [ ${COUNT} -gt 0 ]; then
                SIZE=$(du -sh ${OUTPUT_DIR}/${platform} 2>/dev/null | cut -f1)
                printf "%-10s: %2d 个文件, 总大小: %s\n" ${platform} ${COUNT} ${SIZE}
                has_output=true
            fi
        fi
    done
    
    if [ "$has_output" = false ]; then
        print_warning "没有成功编译的文件"
    fi
    
    echo ""
    
    # 总统计
    TOTAL_FILES=$(find ${OUTPUT_DIR} -type f \( -name "*.exe" -o -name "${BINARY_NAME}-*" \) ! -name "*.tar.gz" 2>/dev/null | wc -l)
    
    if [ ${TOTAL_FILES} -gt 0 ]; then
        TOTAL_SIZE=$(du -sh ${OUTPUT_DIR} 2>/dev/null | cut -f1)
        echo -e "${GREEN}✅ 成功编译: ${BUILD_SUCCESS} 个目标${NC}"
    else
        echo -e "${YELLOW}⚠️  成功编译: ${BUILD_SUCCESS} 个目标${NC}"
    fi
    
    if [ ${BUILD_FAILED} -gt 0 ]; then
        echo -e "${RED}❌ 编译失败: ${BUILD_FAILED} 个目标${NC}"
    fi
    
    if [ ${TOTAL_FILES} -gt 0 ]; then
        echo -e "${CYAN}📦 总计: ${TOTAL_FILES} 个可执行文件${NC}"
        echo -e "${CYAN}💾 总大小: ${TOTAL_SIZE}${NC}"
    fi
    
    # 显示失败的目标列表
    if [ ${BUILD_FAILED} -gt 0 ]; then
        echo ""
        echo -e "${RED}失败的编译目标:${NC}"
        for target in "${FAILED_TARGETS[@]}"; do
            echo -e "  ${RED}•${NC} ${target}"
        done
    fi
    
    echo ""
}

# 清理函数
cleanup() {
    print_info "清理临时文件..."
    rm -f *.syso 2>/dev/null || true
    # 清理可能存在的临时错误日志文件
    rm -f /tmp/tmp.* 2>/dev/null || true
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
    
    # 只在有成功编译的文件时才创建压缩包和校验和
    if [ ${BUILD_SUCCESS} -gt 0 ]; then
        # 创建压缩包
        create_archives
        
        # 生成校验和
        generate_checksums
    else
        print_warning "没有成功编译的文件，跳过创建压缩包和校验和"
        echo ""
    fi
    
    # 显示统计
    show_statistics
    
    # 清理
    cleanup
    
    # 根据编译结果显示不同的结束信息
    if [ ${BUILD_FAILED} -eq 0 ] && [ ${BUILD_SUCCESS} -gt 0 ]; then
        echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${GREEN}✅ 编译全部成功！${NC}"
        echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    elif [ ${BUILD_SUCCESS} -gt 0 ]; then
        echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${YELLOW}⚠️  编译部分完成 (${BUILD_SUCCESS} 成功, ${BUILD_FAILED} 失败)${NC}"
        echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    else
        echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${RED}❌ 编译全部失败！${NC}"
        echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        exit 1
    fi
    
    echo ""
    
    if [ ${BUILD_SUCCESS} -gt 0 ]; then
        echo "输出目录: ${OUTPUT_DIR}/"
        echo "压缩包已创建，可用于分发"
        echo ""
    fi
    
    echo -e "${YELLOW}提示:${NC}"
    echo "  - Linux/macOS 版本需要在对应平台上编译以支持系统托盘（CGO）"
    echo "  - 跨平台编译将自动禁用 CGO，系统托盘功能将不可用"
    echo "  - Windows 版本使用纯 Go 实现，可在任何平台编译"
    echo "  - 某些架构可能不受当前 Go 版本支持，会被自动跳过"
    echo ""
    
    # 如果有失败的编译，返回非零退出码
    if [ ${BUILD_FAILED} -gt 0 ]; then
        exit 2
    fi
}

# 运行主函数
main "$@"