English / [简体中文](docs/README-CN.md)


### ！！Must be installed first [MP4Box](https://gpac.io/downloads/gpac-nightly-builds/)，And confirm [MP4Box](https://gpac.io/downloads/gpac-nightly-builds/) Correctly added to environment variables

### Add features

1. Supports inline covers and LRC lyrics（Demand`media-user-token`，See the instructions at the end for how to get it）
2. Added support for getting word-by-word and out-of-sync lyrics
3. Support downloading singers `go run main.go https://music.apple.com/us/artist/taylor-swift/159260351` `--all-album` Automatically select all albums of the artist
4. The download decryption part is replaced with Sendy McSenderson to decrypt while downloading, and solve the lack of memory when decrypting large files
5. MV Download, installation required[mp4decrypt](https://www.bento4.com/downloads/)
6. Add interactive search with arrow-key navigation `go run main.go --search [song/album/artist] "search_term"`
7. **Batch download support** - Download multiple albums/playlists from text file(s) `go run main.go --batch urls.txt` or multiple files `go run main.go 1.txt 2.txt`
8. **Disc folder separation** - Automatically organize multi-disc albums into separate disc folders (configurable via `separate-disc-folders` in config.yaml)
9. **Music Video toggle** - Enable or disable music video downloads via `download-music-video` config option or `--dl-mv` flag

### Special thanks to `chocomint` for creating `agent-arm64.js`

For acquisition`aac-lc` `MV` `lyrics` You must fill in the information with a subscription`media-user-token`

- `alac (audio-alac-stereo)`
- `ec3 (audio-atmos / audio-ec3)`
- `aac (audio-stereo)`
- `aac-lc (audio-stereo)`
- `aac-he (audio-he)`
- `aac-binaural (audio-stereo-binaural)`
- `aac-downmix (audio-stereo-downmix)`
- Note: `aac-lc` does not work with HE-AAC streams — it only works with AAC variants such as binaural, downmix, and stereo.

# Apple Music ALAC / Dolby Atmos Downloader

Original script by Sorrow. Modified by me to include some fixes and improvements.

## How to use
1. Make sure the decryption program [wrapper](https://github.com/zhaarey/wrapper) is running
2. Start downloading some albums: `go run main.go https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511`.
3. Start downloading single song: `go run main.go --song https://music.apple.com/us/album/never-gonna-give-you-up-2022-remaster/1624945511?i=1624945512` or `go run main.go https://music.apple.com/us/song/you-move-me-2022-remaster/1624945520`.
4. Start downloading select: `go run main.go --select https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511` input numbers separated by spaces.
5. Start downloading some playlists: `go run main.go https://music.apple.com/us/playlist/taylor-swift-essentials/pl.3950454ced8c45a3b0cc693c2a7db97b` or `go run main.go https://music.apple.com/us/playlist/hi-res-lossless-24-bit-192khz/pl.u-MDAWvpjt38370N`.
6. For dolby atmos: `go run main.go --atmos https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538`.
7. For aac: `go run main.go --aac https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538`.
8. For see quality: `go run main.go --debug https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538`.
9. **For batch download from file(s)**: 
   - Single file: `go run main.go --batch urls.txt`
   - Multiple files: `go run main.go 1.txt 2.txt 3.txt`
   - Or: `go run main.go --batch file1.txt --batch file2.txt`
   
   (Create text file(s) with one Apple Music URL per line. See `batch_example.txt` for format).

[Chinese tutorial - see Method 3 for details](https://telegra.ph/Apple-Music-Alac高解析度无損音樂下載教程-04-02-2)

## Downloading lyrics

1. Open [Apple Music](https://music.apple.com) and log in
2. Open the Developer tools, Click `Application -> Storage -> Cookies -> https://music.apple.com`
3. Find the cookie named `media-user-token` and copy its value
4. Paste the cookie value obtained in step 3 into the setting called "media-user-token" in config.yaml and save it
5. Start the script as usual

## Get translation and pronunciation lyrics (Beta)

1. Open [Apple Music](https://beta.music.apple.com) and log in.
2. Open the Developer tools, click `Network` tab.
3. Search a song which is available for translation and pronunciation lyrics (recommend K-Pop songs).
4. Press Ctrl+R and let Developer tools sniff network data.
5. Play a song and then click lyric button, sniff will show a data called `syllable-lyrics`.
6. Stop sniff (small red circles button on top left), then click `Fetch/XHR` tabs.
7. Click `syllable-lyrics` data, see requested URL.
8. Find this line `.../syllable-lyrics?l=<copy all the language value from here>&extend=ttmlLocalizations`.
9. Paste the language value obtained in step 8 into the config.yaml and save it.
10. If don't need pronunciation, do this `...%5D=<remove this value>&extend...` on config.yaml and save it.
11. Start the script as usual.

Noted: These features are only in beta version right now.

## Configuration Options

### Batch Download from Multiple Files

The downloader supports processing multiple batch files at once:

- **Single file**: `go run main.go --batch urls.txt`
- **Multiple files (auto-detected)**: `go run main.go 1.txt 2.txt 3.txt`
- **Multiple files (explicit)**: `go run main.go --batch file1.txt --batch file2.txt`

When you pass `.txt` files as arguments, they are automatically treated as batch files. All URLs from all files are combined and processed sequentially.

**Note:** All downloads are processed sequentially (one at a time) to ensure stability and proper file handling.

### Multi-Disc Album Organization

The downloader now supports organizing multi-disc albums into separate disc folders. This is controlled by the `separate-disc-folders` option in `config.yaml`:

- **`separate-disc-folders: true`** - Creates a "Disc 1", "Disc 2", etc. subfolder for each disc in multi-disc albums
- **`separate-disc-folders: false`** (default) - All tracks are saved directly in the album folder

**Example folder structure with `separate-disc-folders: true`:**
```
Artist Name/
└── Album Name [ALAC]/
    ├── cover.jpg
    ├── Disc 1/
    │   |
    │   ├── 01. Song Title.m4a
    │   └── 02. Song Title.m4a
    └── Disc 2/
        |
        ├── 01. Song Title.m4a
        └── 02. Song Title.m4a
```

**Note:** Single-disc albums are not affected by this setting and will continue to save tracks directly in the album folder.

### Music Video Download Control

You can now control whether music videos are downloaded using either the configuration file or command-line flag:

- **Config file**: Set `download-music-video: true` or `download-music-video: false` in `config.yaml`
- **Command-line**: Use `--dl-mv=true` or `--dl-mv=false` flag when running the program

**Examples:**
```bash
# Disable music video downloads via command line
go run main.go --dl-mv=false https://music.apple.com/us/album/example/123456

# Enable music video downloads via command line (overrides config)
go run main.go --dl-mv=true https://music.apple.com/us/album/example/123456
```

When music video download is disabled:
- Music videos in albums, playlists, and stations will be skipped
- Standalone music video URLs will be skipped
- A message "Music video download is disabled, skipping" will be displayed

**Note:** Music video downloads require:
1. `download-music-video` to be enabled (true)
2. A valid `media-user-token` in config.yaml
3. [mp4decrypt](https://www.bento4.com/downloads/) installed and available in PATH
