#!/bin/bash
set -e

# ── 0. FFmpeg ─────────────────────────────────────────────────────────────────
sudo apt install -y ffmpeg

# ── 1. GPAC ───────────────────────────────────────────────────────────────────
sudo apt install -y git build-essential pkg-config cmake \
  libavcodec-dev libavformat-dev libavutil-dev libavdevice-dev \
  libswscale-dev libfreetype6-dev libjpeg-dev libpng-dev \
  libgl1-mesa-dev libglu1-mesa-dev zlib1g-dev

git clone https://github.com/gpac/gpac.git
cd gpac
./configure
make -j$(nproc)
sudo make install
cd ..

# ── 2. CCExtractor ────────────────────────────────────────────────────────────
sudo apt-get install -y libclang-dev clang libtesseract-dev

curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
source "$HOME/.cargo/env"
cargo --version

git clone https://github.com/CCExtractor/ccextractor
cd ccextractor/linux
./build
cd ../..

# ── 3. Bento4 (mp4decrypt) ────────────────────────────────────────────────────
sudo apt install -y unzip wget

wget https://www.bok.net/Bento4/binaries/Bento4-SDK-1-6-0-641.x86_64-unknown-linux.zip
unzip Bento4-SDK-1-6-0-641.x86_64-unknown-linux.zip
sudo cp Bento4-SDK-1-6-0-641.x86_64-unknown-linux/bin/mp4decrypt /usr/local/bin/

echo "✅ All done!"
