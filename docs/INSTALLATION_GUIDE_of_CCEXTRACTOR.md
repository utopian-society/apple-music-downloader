# Installation and Testing Instructions

## Step 1: Install CCExtractor

CCExtractor is required because FFmpeg cannot properly extract EIA-608 closed captions from Apple Music videos.

Run the installation script:
```bash
cd ~/am
chmod +x install_ccextractor.sh
./install_ccextractor.sh
```

This will:
1. Install required dependencies (git, cmake, gcc, etc.)
2. Clone and build CCExtractor from source
3. Install it to `/usr/local/bin/ccextractor`

**Expected output:**
```
✓ CCExtractor installed successfully!
CCExtractor 0.XX, ...
```

## Step 2: Rebuild the Application

```bash
cd ~/am
go build -o apple-music-downloader main.go
```

## Step 3: Test with the Problematic Video

```bash
./apple-music-downloader https://music.apple.com/us/music-video/i-love-you-im-sorry/1773474489
```

## Expected Result

```
Queue 1 of 1: Music Video I Love You, I'm Sorry
Video: 1280x720-SDR
Downloaded. Decrypted.
Audio: audio-stereo-256
Downloaded. Decrypted.
✓ Closed captions extracted successfully
✓ EIA-608 captions removed successfully
✓ MV Remuxed with subtitles
```

## What Changed

### The Problem
- FFmpeg can detect EIA-608 streams but cannot extract the actual caption data
- FFmpeg extracts only an empty WebVTT header (7 bytes)
- The temp file was created but had no actual subtitles

### The Solution
1. **CCExtractor Priority**: For EIA-608 streams, try CCExtractor FIRST before FFmpeg
2. **Better Validation**: Require files to be > 100 bytes (not just > 0) to avoid empty headers
3. **Proper Tool**: CCExtractor is specifically designed for EIA-608/CEA-608 extraction

### Code Changes
- `ExtractClosedCaptionsFromMP4()`: Detects EIA-608 codec and tries CCExtractor first
- `extractWithCCExtractor()`: Properly configured with `-out=srt` and `-charset=utf8`
- File size validation changed from `> 0` to `> 100` bytes

## Troubleshooting

### If CCExtractor installation fails:
```bash
# Install dependencies manually
sudo apt install -y git cmake gcc g++ pkg-config

# Clone and build
cd /tmp
git clone https://github.com/CCExtractor/ccextractor.git
cd ccextractor/linux
./build
sudo cp ccextractor /usr/local/bin/
```

### If extraction still fails:
```bash
# Test CCExtractor directly
ccextractor "AM-DL downloads/I Love You, I'm Sorry (1773474489).mp4" -out=srt -o test_output.srt

# Check the output
cat test_output.srt
```

### If CCExtractor works but the app doesn't:
```bash
# Rebuild with verbose output
cd ~/am
go build -v -o apple-music-downloader main.go

# Test again
./apple-music-downloader https://music.apple.com/us/music-video/i-love-you-im-sorry/1773474489
```

## Why CCExtractor is Required

**FFmpeg Limitation:**
- FFmpeg can DETECT EIA-608 streams (shows codec: eia_608, tag: c608)
- But FFmpeg cannot DECODE the actual caption data from these streams
- FFmpeg outputs only the format header without content

**CCExtractor Solution:**
- CCExtractor is specifically designed for closed caption extraction
- It can properly decode EIA-608/CEA-608 data from video frames
- It outputs properly formatted SRT files with actual subtitle content

## Dependencies

### Already Installed
- ffmpeg - for video processing
- ffprobe - for stream detection
- MP4Box - for muxing

### Newly Required
- **ccextractor** - for EIA-608 caption extraction

## Summary

The fix requires CCExtractor because:
1. EIA-608 is a specific closed caption format embedded in video frames
2. FFmpeg cannot decode this format (only detects it)
3. CCExtractor is the standard tool for extracting EIA-608 captions
4. Apple Music videos use EIA-608 format for their closed captions

After installing CCExtractor and rebuilding, the subtitle extraction will work correctly!

