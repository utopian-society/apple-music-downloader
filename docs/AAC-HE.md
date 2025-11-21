Guide of AAC He and more

This document explains HE‑AAC (AAC‑HE / HE‑AACv2) support and the new `aac-max` configuration.

What is HE‑AAC?
- HE‑AAC is a family of AAC codecs optimized for low bitrate audio using Spectral Band Replication (SBR) and Parametric Stereo (PS).
- It appears in Apple Music manifests as audio types like `audio-he` or `audio-he-stereo` and uses MPEG‑4 Audio object types (e.g., `mp4a.40.5`) in streams.

Compatibility notes
- `aac-lc` (low complexity) is a different AAC profile and cannot decode HE‑AAC streams. The program's `aac-lc` selection will not match HE‑AAC variants—use `aac` or `aac-he` when you need HE‑AAC content.
- The code recognizes HE‑AAC via codec and manifest labels. If a track uses HE‑AAC, choose `--aac` with `aac-type` set to `aac` or `aac-he`.

New configuration: `aac-max`
- Purpose: limit the maximum AAC bitrate selected for downloads.
- Location: `config.yaml` (key: `aac-max`) and CLI flag `--aac-max`.
- Behavior: When `--aac` mode is used, the downloader inspects available AAC/HE‑AAC variants and selects the highest variant whose bitrate is less than or equal to `aac-max`.

Recommended values
- 64, 96, 128, 192, 256 (kbps) — pick according to quality and file size tradeoffs.

Examples
- config.yaml:
  aac-type: aac
  aac-max: 128

- CLI override:
  go run main.go --aac --aac-max 256 <album-url>

Notes & troubleshooting
- If you set `aac-type: aac-lc` but the manifest only contains HE‑AAC variants, the program may fall back or report the stream as unavailable. Switch to `aac` or `aac-he`.
- Debugging: use `--debug` to print detected AAC variants and their bitrates. This helps confirm which variant will be chosen based on `aac-max`.
- Decryption requirement: AAC-HE (and many protected AAC streams) require the decryption `wrapper` to be running so the downloader can decrypt segments while downloading. Make sure the `wrapper` decryption service is running (see `README.md` for setup) before attempting AAC-HE downloads.

Changelog
- Added `aac-max` configuration and CLI flag.
- Documented HE‑AAC behaviour and clarified `aac-lc` limitations.

"More"
- The downloader already supports AAC variants such as binaural and downmix. Use `aac-type` to pick the desired AAC flavor.

End of guide.
