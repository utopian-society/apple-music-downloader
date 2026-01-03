# Playlist Search Feature

## Overview
The application supports searching for playlists directly from the command line, allowing you to find and download Apple Music playlists by name.

## Usage

### Search for Playlists
```bash
./main --search playlist <query>
```

### Examples
```bash
./main --search playlist "Today's Hits"
./main --search playlist "Chill Vibes"
./main --search playlist "New Music Daily"
./main --search playlist "Workout"
```

## How It Works

1. **Search API**: The search functionality queries Apple Music's API with the `playlists` type
2. **Interactive Selection**: Search results are displayed with:
   - Playlist name
   - Curator name (e.g., "Apple Music" for official playlists)
3. **Pagination**: Navigate through results using "Previous Page" and "Next Page" options
4. **Quality Selection**: After selecting a playlist, choose from available download qualities:
   - Lossless (ALAC)
   - High-Quality (AAC)
   - Dolby Atmos

## Display Format

Search results are displayed in the following format:
```
Playlist Name - Curator Name
```

For example:
```
Today's Hits - Apple Music
New Music Daily - Apple Music
Chill Vibes - Apple Music
```

If a playlist has no curator name, it defaults to "Apple Music".

## Implementation Details

### Modified Files

1. **utils/ampapi/search.go**
   - Added `Playlists *PlaylistResults` field to `SearchResults` struct
   - Added `PlaylistResults` struct for holding playlist search results
   - Added `PlaylistSearchData` struct for playlist search result items with fields:
     - `CuratorName`: The name of the playlist curator
     - `Name`: The playlist name
     - `URL`: The Apple Music URL for the playlist
     - `Artwork`: Artwork information

2. **main.go**
   - Added `"playlist": true` to `validTypes` map in `handleSearch()`
   - Added `playlist` → `playlists` API type mapping
   - Added case handling for playlist results in the search results switch statement
   - Updated `--search` flag description to include `'playlist'`
   - Updated help text usage line to include `playlist`

## API Changes

### SearchResults Struct
```go
type SearchResults struct {
    Songs       *SongResults       `json:"songs,omitempty"`
    Albums      *AlbumResults      `json:"albums,omitempty"`
    Artists     *ArtistResults     `json:"artists,omitempty"`
    MusicVideos *MusicVideoResults `json:"music-videos,omitempty"`
    Playlists   *PlaylistResults   `json:"playlists,omitempty"` // NEW
}
```

### PlaylistResults Struct
```go
type PlaylistResults struct {
    Href string               `json:"href"`
    Next string               `json:"next"`
    Data []PlaylistSearchData `json:"data"`
}
```

### PlaylistSearchData Struct
```go
type PlaylistSearchData struct {
    ID         string `json:"id"`
    Type       string `json:"type"`
    Href       string `json:"href"`
    Attributes struct {
        CuratorName string `json:"curatorName"`
        Name        string `json:"name"`
        URL         string `json:"url"`
        Artwork     struct {
            Width  int    `json:"width"`
            Height int    `json:"height"`
            URL    string `json:"url"`
        } `json:"artwork"`
        PlayParams struct {
            ID   string `json:"id"`
            Kind string `json:"kind"`
        } `json:"playParams"`
    } `json:"attributes"`
}
```

## All Search Types

The `--search` flag now supports the following types:

| Type | Description | Example |
|------|-------------|---------|
| `album` | Search for albums | `--search album "1989 Taylor Swift"` |
| `song` | Search for songs | `--search song "Blinding Lights"` |
| `artist` | Search for artists | `--search artist "The Weeknd"` |
| `music-video` | Search for music videos | `--search music-video "Bad Guy"` |
| `playlist` | Search for playlists | `--search playlist "Today's Hits"` |

## Notes

- The search API returns limited information for playlists (name, curator, artwork, URL)
- Track count is not available in search results; it's only retrieved when fetching the full playlist
- Pagination is supported with 15 results per page
- Use arrow keys to navigate and Enter to select

