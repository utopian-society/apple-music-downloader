#!/bin/bash

# Simple CCExtractor installation using pre-built binary

echo "========================================="
echo "Installing CCExtractor (pre-built binary)"
echo "========================================="

# Download pre-built binary from GitHub releases
echo "[1/3] Downloading CCExtractor pre-built binary..."
cd /tmp
wget https://github.com/CCExtractor/ccextractor/releases/download/v0.94/ccextractor.binaries.v0.94.zip

if [ $? -ne 0 ]; then
    echo "Failed to download CCExtractor"
    echo "Trying alternative download method..."
    curl -L -o ccextractor.binaries.v0.94.zip https://github.com/CCExtractor/ccextractor/releases/download/v0.94/ccextractor.binaries.v0.94.zip
    if [ $? -ne 0 ]; then
        echo "Download failed. Please check your internet connection."
        exit 1
    fi
fi

# Extract the binary
echo ""
echo "[2/3] Extracting binary..."
sudo apt install -y unzip
unzip -q ccextractor.binaries.v0.94.zip

# Find and install the Linux binary
if [ -f "ccextractor" ]; then
    BINARY="ccextractor"
elif [ -f "ccextractor.linux" ]; then
    BINARY="ccextractor.linux"
elif [ -f "ccextractor.binaries.v0.94/linux/ccextractor" ]; then
    BINARY="ccextractor.binaries.v0.94/linux/ccextractor"
else
    echo "Could not find ccextractor binary in archive"
    ls -la
    exit 1
fi

echo ""
echo "[3/3] Installing to /usr/local/bin..."
sudo cp "$BINARY" /usr/local/bin/ccextractor
sudo chmod +x /usr/local/bin/ccextractor

# Clean up
cd ~
rm -rf /tmp/ccextractor*

# Verify installation
echo ""
echo "========================================="
if command -v ccextractor &> /dev/null; then
    echo "✓ CCExtractor installed successfully!"
    echo "========================================="
    ccextractor --version 2>&1 | head -5
    echo ""
    echo "Next steps:"
    echo "1. Rebuild apple-music-downloader: go build -o apple-music-downloader main.go"
    echo "2. Test extraction: ./apple-music-downloader <music-video-url>"
    exit 0
else
    echo "✗ Installation failed"
    echo "========================================="
    exit 1
fi
