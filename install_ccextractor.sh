#!/bin/bash

# CCExtractor installation script for EIA-608 caption extraction

echo "========================================="
echo "Installing CCExtractor from source"
echo "========================================="

# Install dependencies
echo "[1/6] Installing build dependencies..."
sudo apt install -y git cmake gcc g++ pkg-config libglew-dev libglfw3-dev tesseract-ocr libtesseract-dev libleptonica-dev curl libclang-dev

if [ $? -ne 0 ]; then
    echo "Failed to install dependencies"
    exit 1
fi

# Install Rust/Cargo (required by CCExtractor)
echo ""
echo "[2/6] Installing Rust/Cargo..."
if ! command -v cargo &> /dev/null; then
    curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
fi

# Source cargo environment
if [ -f "$HOME/.cargo/env" ]; then
    source "$HOME/.cargo/env"
fi
export PATH="$HOME/.cargo/bin:$PATH"

# Update Rust to ensure it meets CCExtractor's MSRV (1.87.0+)
echo "Updating Rust to latest stable version..."
if command -v rustup &> /dev/null; then
    rustup update stable
else
    echo "Warning: rustup not found, skipping update"
fi

# Verify cargo is available
if ! command -v cargo &> /dev/null; then
    echo "Failed to install Rust/Cargo"
    exit 1
fi

# Clone CCExtractor repository
echo ""
echo "[3/6] Cloning CCExtractor repository..."
cd /tmp
rm -rf ccextractor
git clone https://github.com/CCExtractor/ccextractor.git
cd ccextractor

if [ $? -ne 0 ]; then
    echo "Failed to clone repository"
    exit 1
fi

# Build CCExtractor
echo ""
echo "[4/6] Building CCExtractor..."
cd linux
# Make sure cargo is in PATH
export PATH="$HOME/.cargo/bin:$PATH"
./build

if [ $? -ne 0 ]; then
    echo "Failed to build CCExtractor"
    exit 1
fi

# Install CCExtractor
echo ""
echo "[5/6] Installing CCExtractor..."
sudo cp ccextractor /usr/local/bin/
sudo chmod +x /usr/local/bin/ccextractor

if [ $? -ne 0 ]; then
    echo "Failed to install CCExtractor"
    exit 1
fi

# Verify installation
echo ""
echo "[6/6] Verifying installation..."
ccextractor --version

if [ $? -eq 0 ]; then
    echo ""
    echo "========================================="
    echo "✓ CCExtractor installed successfully!"
    echo "========================================="
    echo ""
    echo "Location: /usr/local/bin/ccextractor"
    echo ""
    echo "Next steps:"
    echo "1. Rebuild apple-music-downloader: go build -o apple-music-downloader main.go"
    echo "2. Test extraction: ./apple-music-downloader <music-video-url>"

    # Clean up
    cd ~
    rm -rf /tmp/ccextractor

    exit 0
else
    echo "Installation verification failed"
    exit 1
fi
