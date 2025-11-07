#!/bin/bash

# Apple Music Downloader Build Script
# Purpose: Compile and generate binary file
# Output: apple-music-downloader (project root directory)

set -e  # Exit immediately on error

# Embedded permission fix - Ensure script has execute permission
# This code attempts to fix permissions even when restricted
chmod +x "$0" 2>/dev/null || true

# Force permission fix - Fallback method for WSL or permission issues
(umask 0077; chmod +x "$0" 2>/dev/null || true) &
wait

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Project information
PROJECT_NAME="apple-music-downloader"
BUILD_OUTPUT="${PROJECT_NAME}"

echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     Apple Music Downloader - Build Script                     ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Get current directory (project root directory)
PROJECT_ROOT=$(pwd)
echo -e "${BLUE}📂 Project Root:${NC} ${PROJECT_ROOT}"
echo ""

# Clean old binary files
echo -e "${YELLOW}🧹 Cleaning old version...${NC}"
if [ -f "${BUILD_OUTPUT}" ]; then
    rm -f "${BUILD_OUTPUT}"
    echo -e "${GREEN}✅ Deleted old binary file${NC}"
else
    echo -e "${BLUE}ℹ️  Old version file not found${NC}"
fi
echo ""

# Get version information
echo -e "${YELLOW}📝 Collecting version information...${NC}"
GIT_TAG=$(git describe --tags 2>/dev/null || git tag --sort=-v:refname | head -n 1 2>/dev/null || echo "v0.0.0")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date '+%Y-%m-%d %H:%M:%S %Z')
GO_VERSION=$(go version | awk '{print $3}' || echo "unknown")

echo -e "   Version Tag: ${GREEN}${GIT_TAG}${NC}"
echo -e "   Commit Hash: ${GREEN}${GIT_COMMIT}${NC}"
echo -e "   Build Time:  ${GREEN}${BUILD_TIME}${NC}"
echo -e "   Go Version:  ${GREEN}${GO_VERSION}${NC}"
echo ""

# Build binary file
echo -e "${YELLOW}🔨 Starting compilation...${NC}"
echo -e "   Target Platform: ${BLUE}$(go env GOOS)/$(go env GOARCH)${NC}"
echo -e "   Output File:     ${BLUE}${BUILD_OUTPUT}${NC}"
echo ""

# Build parameters
LDFLAGS="-s -w"
LDFLAGS="${LDFLAGS} -X 'main.Version=${GIT_TAG}'"
LDFLAGS="${LDFLAGS} -X 'main.CommitHash=${GIT_COMMIT}'"
LDFLAGS="${LDFLAGS} -X 'main.BuildTime=${BUILD_TIME}'"

# Check if upx is installed
if command -v upx >/dev/null 2>&1; then
    echo -e "${YELLOW}📦 Using UPX to compress binary...${NC}"
    COMPRESS_FLAG="-buildmode=pie"
else
    COMPRESS_FLAG=""
fi

# Execute build
if go build ${COMPRESS_FLAG} -trimpath -ldflags="${LDFLAGS}" -o "${BUILD_OUTPUT}" .; then
    if [ ! -z "$COMPRESS_FLAG" ]; then
        upx --best --lzma "${BUILD_OUTPUT}" || true
    fi
    echo -e "${GREEN}✅ Compilation successful!${NC}"
    echo ""
else
    echo -e "${RED}❌ Compilation failed!${NC}"
    exit 1
fi

# Display build results
echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     Build Complete                                             ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# File information
FILE_SIZE=$(du -h "${BUILD_OUTPUT}" | cut -f1)
FILE_PATH=$(realpath "${BUILD_OUTPUT}")

echo -e "${GREEN}📦 Binary File Information:${NC}"
echo -e "   File Name:   ${BLUE}${BUILD_OUTPUT}${NC}"
echo -e "   File Size:   ${BLUE}${FILE_SIZE}${NC}"
echo -e "   Full Path:   ${BLUE}${FILE_PATH}${NC}"
echo -e "   Executable:  ${GREEN}✓${NC}"
echo ""

# Set execute permission
chmod +x "${BUILD_OUTPUT}"

# Verify executable
echo -e "${YELLOW}🧪 Verifying build...${NC}"
if "${BUILD_OUTPUT}" --help 2>&1 | head -10 | grep -iq "Music"; then
    echo -e "${GREEN}✅ Verification successful, program runs correctly${NC}"
else
    echo -e "${RED}⚠️  Warning: Program may not run correctly${NC}"
fi
echo ""

# Usage instructions
echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     Usage Instructions                                         ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "  Run program:"
echo -e "    ${GREEN}./${BUILD_OUTPUT}${NC}"
echo ""
echo -e "  View help:"
echo -e "    ${GREEN}./${BUILD_OUTPUT} --help${NC}"
echo ""
echo -e "${GREEN}🎉 Build complete!${NC}"
