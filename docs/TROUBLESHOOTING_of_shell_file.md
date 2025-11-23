# Troubleshooting Guide

This document contains solutions to common issues you may encounter when using the Apple Music Downloader.

## Script Execution Errors

### Error: "cannot execute: required file not found"

If you encounter this error when trying to run `build.sh` or other shell scripts:

```bash
./build.sh
-bash: ./build.sh: cannot execute: required file not found
```


**Cause:** This error occurs when shell scripts have Windows-style line endings (CRLF) instead of Unix-style line endings (LF). This commonly happens when files are created or edited on Windows and then transferred to a Linux/WSL environment.

**Solution:** Convert the line endings using the `sed` command:

```bash
sed -i 's/\r$//' *.sh
```

This command will fix all `.sh` files in the current directory by removing the carriage return character (`\r`) from the end of each line.

**For a single file:**
```bash
sed -i 's/\r$//' build.sh
```

**For multiple specific files:**
```bash
sed -i 's/\r$//' build.sh install.sh deploy.sh
```

**Alternative solution using dos2unix:**
```bash
# Install dos2unix if not already installed
sudo apt-get install dos2unix

# Convert line endings
dos2unix *.sh
```

After fixing the line endings, make sure the script is executable:
```bash
chmod +x build.sh
./build.sh
```

---

## Build Issues

### Missing Dependencies

If compilation fails, ensure all required dependencies are installed. Refer to the installation guides in the `docs/` folder.

---

## Download Issues

### Missing media-user-token

If you're unable to download certain content types (AAC-LC, MV, lyrics), make sure you have properly configured your `media-user-token` in `config.yaml`.

See the main [README.md](../README.md) for instructions on how to obtain this token.

---

## Other Issues

If you encounter issues not covered in this guide, please:
1. Check the [README.md](../README.md) for general usage instructions
2. Review the relevant installation guides in the `docs/` folder
3. Check if there are any error messages in the console output
4. Ensure all prerequisites (MP4Box, mp4decrypt, wrapper) are properly installed and in your PATH
