#!/bin/bash

# Apple Music Downloader 构建脚本
# 用途: 编译生成二进制文件
# 输出: apple-music-downloader (项目根目录)

set -e  # 遇到错误立即退出

# 嵌入式权限修复 - 确保脚本具有执行权限
# 这行代码即使在权限受限时也会尝试修复
chmod +x "$0" 2>/dev/null || true

# 强制修复权限 - 来自WSL或权限问题的备用方法
(umask 0077; chmod +x "$0" 2>/dev/null || true) &
wait

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目信息
PROJECT_NAME="apple-music-downloader"
BUILD_OUTPUT="${PROJECT_NAME}"

echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     Apple Music Downloader - 构建脚本                         ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# 获取当前目录（项目根目录）
PROJECT_ROOT=$(pwd)
echo -e "${BLUE}📂 项目根目录:${NC} ${PROJECT_ROOT}"
echo ""

# 清理旧的二进制文件
echo -e "${YELLOW}🧹 清理旧版本...${NC}"
if [ -f "${BUILD_OUTPUT}" ]; then
    rm -f "${BUILD_OUTPUT}"
    echo -e "${GREEN}✅ 已删除旧版本二进制文件${NC}"
else
    echo -e "${BLUE}ℹ️  未找到旧版本文件${NC}"
fi
echo ""

# 获取版本信息
echo -e "${YELLOW}📝 收集版本信息...${NC}"
GIT_TAG=$(git describe --tags 2>/dev/null || git tag --sort=-v:refname | head -n 1 2>/dev/null || echo "v0.0.0")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date '+%Y-%m-%d %H:%M:%S %Z')
GO_VERSION=$(go version | awk '{print $3}' || echo "unknown")

echo -e "   版本标签: ${GREEN}${GIT_TAG}${NC}"
echo -e "   提交哈希: ${GREEN}${GIT_COMMIT}${NC}"
echo -e "   构建时间: ${GREEN}${BUILD_TIME}${NC}"
echo -e "   Go 版本:  ${GREEN}${GO_VERSION}${NC}"
echo ""

# 构建二进制文件
echo -e "${YELLOW}🔨 开始编译...${NC}"
echo -e "   目标平台: ${BLUE}$(go env GOOS)/$(go env GOARCH)${NC}"
echo -e "   输出文件: ${BLUE}${BUILD_OUTPUT}${NC}"
echo ""

# 构建参数
LDFLAGS="-s -w"
LDFLAGS="${LDFLAGS} -X 'main.Version=${GIT_TAG}'"
LDFLAGS="${LDFLAGS} -X 'main.CommitHash=${GIT_COMMIT}'"
LDFLAGS="${LDFLAGS} -X 'main.BuildTime=${BUILD_TIME}'"

# 检查是否安装了 upx
if command -v upx >/dev/null 2>&1; then
    echo -e "${YELLOW}📦 使用 UPX 压缩二进制文件...${NC}"
    COMPRESS_FLAG="-buildmode=pie"
else
    COMPRESS_FLAG=""
fi

# 执行构建
if go build ${COMPRESS_FLAG} -trimpath -ldflags="${LDFLAGS}" -o "${BUILD_OUTPUT}" .; then
    if [ ! -z "$COMPRESS_FLAG" ]; then
        upx --best --lzma "${BUILD_OUTPUT}" || true
    fi
    echo -e "${GREEN}✅ 编译成功！${NC}"
    echo ""
else
    echo -e "${RED}❌ 编译失败！${NC}"
    exit 1
fi

# 显示构建结果
echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     构建完成                                                   ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# 文件信息
FILE_SIZE=$(du -h "${BUILD_OUTPUT}" | cut -f1)
FILE_PATH=$(realpath "${BUILD_OUTPUT}")

echo -e "${GREEN}📦 二进制文件信息:${NC}"
echo -e "   文件名:   ${BLUE}${BUILD_OUTPUT}${NC}"
echo -e "   文件大小: ${BLUE}${FILE_SIZE}${NC}"
echo -e "   完整路径: ${BLUE}${FILE_PATH}${NC}"
echo -e "   可执行:   ${GREEN}✓${NC}"
echo ""

# 设置执行权限
chmod +x "${BUILD_OUTPUT}"

# 验证可执行
echo -e "${YELLOW}🧪 验证构建...${NC}"
if "${BUILD_OUTPUT}" --help 2>&1 | head -10 | grep -iq "Music"; then
    echo -e "${GREEN}✅ 验证成功，程序可正常运行${NC}"
else
    echo -e "${RED}⚠️  警告: 程序可能无法正常运行${NC}"
fi
echo ""

# 使用提示
echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     使用方法                                                   ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "  运行程序:"
echo -e "    ${GREEN}./${BUILD_OUTPUT}${NC}"
echo ""
echo -e "  查看帮助:"
echo -e "    ${GREEN}./${BUILD_OUTPUT} --help${NC}"
echo ""
echo -e "  查看版本:"
echo -e "    ${GREEN}./${BUILD_OUTPUT} --version${NC}"
echo ""
echo -e "${GREEN}🎉 构建完成！${NC}"
