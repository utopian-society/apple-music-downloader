# Lyrics Only Mode

Download only word-synced lyrics files (LRC or TTML) without downloading audio files.

## Usage

### Via Command Line Flag

```bash
./main --lyrics <url>
```

Example:
```bash
./main --lyrics https://music.apple.com/jp/album/red/1402161820
```

### Via Config File

Set `lyrics-only: true` in `config.yaml`:

```yaml
lyrics-only: true
```

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `lyrics-only` | bool | `false` | Download only lyrics files (no audio) |
| `lrc-type` | string | `syllable-lyrics` | Type of lyrics: `lyrics` or `syllable-lyrics` (word-synced) |
| `lrc-format` | string | `lrc` | Output format: `lrc` or `ttml` |
| `save-lrc-file` | bool | `true` | Save lyrics as separate file |

## Requirements

- Valid `media-user-token` in config.yaml (required for lyrics access)
- Tracks must have lyrics available on Apple Music

## Output

Lyrics files are saved in the same folder structure as audio downloads:
- Format determined by `lrc-format` setting (`.lrc` or `.ttml`)
- Filename follows `song-file-format` pattern with lyrics extension

## Notes

- The `--lyrics` flag overrides the `lyrics-only` config setting
- If a track has no lyrics available, it will be marked as "Unavailable"
- Word-synced lyrics (`syllable-lyrics`) provide per-word timing for karaoke-style display
- Regular lyrics (`lyrics`) provide per-line timing

