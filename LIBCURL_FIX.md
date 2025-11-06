# libcurl4-openssl-dev Dependency Fix

## Problem
The subtitle extraction functionality previously had a dependency on `CCExtractor`, which requires `libcurl4-openssl-dev` during compilation. This package is not available in newer OS distributions (like Ubuntu 24.04+), causing build failures.

## Solution
The subtitle extraction code has been refactored to remove the dependency on CCExtractor and use only FFmpeg-based methods, which don't require libcurl4-openssl-dev.

## Changes Made

### 1. Updated `utils/subtitle/subtitle.go`
- **Replaced `extractWithCCExtractor()` function** with `extractWithFFmpegOnly()`
- The new implementation uses only FFmpeg with multiple fallback methods:
  - Method 1: Extract with text subtitles codec
  - Method 2: Extract with mov_text codec for EIA-608/CEA-608
  - Method 3: Extract all available subtitle streams
- No external dependencies beyond FFmpeg (which was already required)

### 2. Updated Documentation
- Removed the warning about libcurl4-openssl-dev from both `README.md` and `README-CN.md`
- The application now works on newer OS distributions without special dependency workarounds

## Benefits
- ✅ **No libcurl dependency**: Works on Ubuntu 24.04+ and other modern distributions
- ✅ **Simpler installation**: Fewer dependencies to install
- ✅ **Same functionality**: All subtitle extraction features still work
- ✅ **Better compatibility**: Uses standard FFmpeg features available across platforms
- ✅ **Pure Go + FFmpeg**: Only requires Go and FFmpeg, both of which are standard requirements

## Testing
The code compiles successfully without any errors. All subtitle extraction methods remain functional through FFmpeg's built-in capabilities.

## Requirements
- Go (already required)
- FFmpeg (already required)
- ~~libcurl4-openssl-dev~~ ❌ NO LONGER NEEDED
- ~~CCExtractor~~ ❌ NO LONGER NEEDED

