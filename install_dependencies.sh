#!/bin/bash
set -euo pipefail

# ── 0. FFmpeg + curl + Golang ─────────────────────────────────────────────────
sudo apt install -y ffmpeg curl golang-go

# ── 1. GPAC ───────────────────────────────────────────────────────────────────
sudo apt install -y git build-essential pkg-config cmake \
  libavcodec-dev libavformat-dev libavutil-dev libavdevice-dev \
  libswscale-dev libfreetype6-dev libjpeg-dev libpng-dev \
  libgl1-mesa-dev libglu1-mesa-dev zlib1g-dev

git clone https://github.com/gpac/gpac.git
cd gpac
./configure
make -j"$(nproc)"
sudo make install
cd ..

# ── 2. CCExtractor ────────────────────────────────────────────────────────────
sudo apt-get install -y libclang-dev clang libtesseract-dev

curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
# shellcheck disable=SC1091
source "$HOME/.cargo/env"
cargo --version

git clone https://github.com/CCExtractor/ccextractor
cd ccextractor/linux
./build
sudo cp ccextractor /usr/local/bin/
sudo chmod +x /usr/local/bin/ccextractor
cd ../..

# ── 3. Bento4 (mp4decrypt) ────────────────────────────────────────────────────
sudo apt install -y unzip wget

wget https://www.bok.net/Bento4/binaries/Bento4-SDK-1-6-0-641.x86_64-unknown-linux.zip
unzip Bento4-SDK-1-6-0-641.x86_64-unknown-linux.zip
sudo cp Bento4-SDK-1-6-0-641.x86_64-unknown-linux/bin/mp4decrypt /usr/local/bin/
sudo chmod +x /usr/local/bin/mp4decrypt
rm -rf Bento4-SDK-1-6-0-641.x86_64-unknown-linux Bento4-SDK-1-6-0-641.x86_64-unknown-linux.zip

# ── Cleanup ───────────────────────────────────────────────────────────────────
rm -rf gpac ccextractor
rustup self uninstall -y

echo "✅ All done!"
echo ""
echo "Verifying installations:"
MP4Box -version 2>&1 | head -1
ccextractor --version 2>&1 | head -1
mp4decrypt 2>&1 | head -1
ffmpeg -version 2>&1 | head -1
go version 2>&1 | head -1
