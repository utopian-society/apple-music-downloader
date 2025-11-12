# Music Video Search Feature

## Overview
The application now supports searching for music videos directly from the command line.

## Usage

### Search for Music Videos
```bash
./am --search music-video <query>
```

### Example
```bash
./am --search music-video "Bad Guy Billie Eilish"
```

## How It Works

1. **Search API**: The search functionality queries Apple Music's API with the `music-videos` type
2. **Interactive Selection**: Search results are displayed with:
   - Music video title
   - Artist name
   - Release year
3. **Pagination**: Navigate through results using "Previous Page" and "Next Page" options
4. **Quality Selection**: After selecting a music video, choose from available download qualities

## Implementation Details

### Modified Files

1. **utils/ampapi/search.go**
   - Added `MusicVideoResults` struct to `SearchResults`
   - Added `MusicVideoResults` type definition for handling music video search results

2. **main.go**
   - Updated `handleSearch()` function to support "music-video" search type
   - Added case handling for music video results in the search results switch statement
   - Updated help text to include "music-video" as a valid search type

## API Changes

### SearchResults Struct
```go
type SearchResults struct {
    Songs       *SongResults       `json:"songs,omitempty"`
    Albums      *AlbumResults      `json:"albums,omitempty"`
    Artists     *ArtistResults     `json:"artists,omitempty"`
    MusicVideos *MusicVideoResults `json:"music-videos,omitempty"` // NEW
}
```

### MusicVideoResults Struct
```go
type MusicVideoResults struct {
    Href string               `json:"href"`
    Next string               `json:"next"`
    Data []MusicVideoRespData `json:"data"`
}
```

## Valid Search Types
- `album` - Search for albums
- `song` - Search for songs
- `artist` - Search for artists
- `music-video` - Search for music videos (NEW)

## Notes
- Music video search uses the existing `MusicVideoRespData` struct from `musicvideo.go`
- The search type "music-video" is automatically converted to "music-videos" for the API call
- Results display artist name and release year for easy identification

