package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"amdl/internal/amp-api"
	mvmedia "amdl/internal/media/mv"
	"amdl/internal/model"
	playreadyrip "amdl/internal/playready-rip"
	"amdl/internal/widevine-rip/runv5"
	"amdl/internal/wrapper"

	"github.com/itouakirai/go-mp4tag"
)

func (r *Runner) mvDownloader(adamID string, saveDir string, token string, storefront string, track *model.Track) error {
	MVInfo, err := ampapi.GetMusicVideoResp(storefront, adamID, r.Config.Language, token)
	if err != nil {
		fmt.Println("\u26A0 Failed to get MV manifest:", err)
		return nil
	}
	if len(MVInfo.Data) == 0 {
		return errors.New("music video response contains no data")
	}

	if strings.HasSuffix(saveDir, ".") {
		saveDir = strings.ReplaceAll(saveDir, ".", "")
	}
	saveDir = strings.TrimSpace(saveDir)

	vidPath := filepath.Join(saveDir, fmt.Sprintf("%s_vid.mp4", adamID))
	audPath := filepath.Join(saveDir, fmt.Sprintf("%s_aud.mp4", adamID))
	mvSaveName := fmt.Sprintf("%s (%s)", MVInfo.Data[0].Attributes.Name, adamID)
	if track != nil {
		mvSaveName = fmt.Sprintf("%02d. %s", track.TaskNum, MVInfo.Data[0].Attributes.Name)
	}

	mvOutPath := filepath.Join(saveDir, fmt.Sprintf("%s.mp4", forbiddenNames.ReplaceAllString(mvSaveName, "_")))

	fmt.Println(MVInfo.Data[0].Attributes.Name)

	exists, _ := fileExists(mvOutPath)
	if exists {
		fmt.Println("MV already exists locally.")

		mvArtistName := MVInfo.Data[0].Attributes.ArtistName
		mvAlbumName := MVInfo.Data[0].Attributes.AlbumName
		mvName := MVInfo.Data[0].Attributes.Name
		mvArtistId := ""
		if len(MVInfo.Data[0].Relationships.Artists.Data) > 0 {
			mvArtistId = MVInfo.Data[0].Relationships.Artists.Data[0].ID
		}

		r.State.AddedTracks = append(r.State.AddedTracks, AddedTrack{
			Path:     mvOutPath,
			Artist:   mvArtistName,
			ArtistID: mvArtistId,
			Album:    mvAlbumName,
			Song:     mvName,
		})
		return nil
	}

	mvm3u8url, err := wrapper.GetWebplayback(r.Config.LiteServer, adamID)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(saveDir, os.ModePerm); err != nil {
		return err
	}

	videom3u8url, usePlayReady, err := r.extractVideoVariant(mvm3u8url)
	if err != nil {
		return fmt.Errorf("extract video manifest: %w", err)
	}
	var videokeyAndUrls string
	if usePlayReady {
		fmt.Println("Video DRM: PlayReady")
		videokeyAndUrls, err = playreadyrip.Run(adamID, videom3u8url, r.Config.LiteServer)
	} else {
		videokeyAndUrls, err = runv5.Run(adamID, videom3u8url, token, true, r.Config.LiteServer)
	}
	if err != nil {
		return fmt.Errorf("download video stream: %w", err)
	}
	if err := runv5.ExtMvData(videokeyAndUrls, vidPath); err != nil {
		return fmt.Errorf("write video stream: %w", err)
	}
	defer os.Remove(vidPath)

	audiom3u8url, err := r.extractMvAudio(mvm3u8url)
	if err != nil {
		return fmt.Errorf("extract audio manifest: %w", err)
	}
	audiokeyAndUrls, err := runv5.Run(adamID, audiom3u8url, token, true, r.Config.LiteServer)
	if err != nil {
		return fmt.Errorf("download audio stream: %w", err)
	}
	if err := runv5.ExtMvData(audiokeyAndUrls, audPath); err != nil {
		return fmt.Errorf("write audio stream: %w", err)
	}
	defer os.Remove(audPath)

	var covPath string
	if r.Config.EmbedCover {
		thumbURL := MVInfo.Data[0].Attributes.Artwork.URL
		baseThumbName := forbiddenNames.ReplaceAllString(mvSaveName, "_") + "_thumbnail"
		covPath, err = r.writeCover(saveDir, baseThumbName, thumbURL)
		if err != nil {
			fmt.Println("Failed to save MV thumbnail:", err)
			covPath = ""
		} else {
			defer os.Remove(covPath)
		}
	}

	fmt.Printf("MV Remuxing...")
	if err := mvmedia.Mux(vidPath, audPath, mvOutPath); err != nil {
		fmt.Printf("MV mux failed: %v\n", err)
		return err
	}
	fmt.Printf("\rMV Remuxed.   \n")

	// Subtitle handling: extract, download (WebVTT → SRT) or extract in-stream CC, optionally save/embed.
	var srtPathToEmbed string
	var srtLangToEmbed string
	if r.Config.MVEmbedSubtitles || r.Config.MVSaveSubtitleFile {
		var subs []ExtractedSubtitle
		baseMVName := forbiddenNames.ReplaceAllString(mvSaveName, "_")

		// 1. Check for external subtitle tracks in master HLS playlist (WebVTT).
		hlsSubs, err := r.extractMvSubtitles(mvm3u8url)
		if err == nil && len(hlsSubs) > 0 {
			for _, sub := range hlsSubs {
				srtName := fmt.Sprintf("%s_%s.srt", baseMVName, sub.Lang)
				srtPath := filepath.Join(saveDir, srtName)
				if err := downloadWebVTTtoSRT(sub.URL, srtPath); err != nil {
					fmt.Printf("⚠ Failed to download subtitle (%s): %v\n", sub.Lang, err)
					continue
				}
				subs = append(subs, ExtractedSubtitle{Lang: sub.Lang, Path: srtPath})
			}
		}

		// 2. If no playlist subtitles found, extract in-stream closed captions (EIA-608 / CEA-708) from video.
		if len(subs) == 0 {
			videoSource := mvOutPath
			if exists, _ := fileExists(videoSource); !exists {
				videoSource = vidPath
			}
			ccSubs, err := ExtractSubtitlesFromVideo(videoSource, saveDir, baseMVName)
			if err != nil {
				fmt.Printf("⚠ Failed to extract closed captions: %v\n", err)
			} else {
				subs = append(subs, ccSubs...)
			}
		}

		if len(subs) == 0 {
			fmt.Println("No subtitle tracks found in MV.")
		} else {
			for _, sub := range subs {
				fmt.Printf("Subtitle (%s): %s\n", sub.Lang, filepath.Base(sub.Path))
				if r.Config.MVEmbedSubtitles && srtPathToEmbed == "" {
					srtPathToEmbed = sub.Path
					srtLangToEmbed = sub.Lang
				}
				if !r.Config.MVSaveSubtitleFile {
					defer os.Remove(sub.Path)
				}
			}
		}
	}

	// Embed the subtitle track into the final MP4 before writing tags.
	if srtPathToEmbed != "" {
		fmt.Printf("Embedding subtitles...")
		if _, err := EmbedSubtitlesInMV(mvOutPath, srtPathToEmbed, srtLangToEmbed); err != nil {
			fmt.Printf("\r⚠ Subtitle embed failed: %v\n", err)
		} else {
			fmt.Printf("\rSubtitles embedded.   \n")
		}
	}

	if err := r.writeMVMP4Tags(mvOutPath, MVInfo, track, covPath); err != nil {
		_ = os.Remove(mvOutPath)
		fmt.Printf("MV tag writing failed: %v\n", err)
		return err
	}

	mvArtistName := MVInfo.Data[0].Attributes.ArtistName
	mvAlbumName := MVInfo.Data[0].Attributes.AlbumName
	mvName := MVInfo.Data[0].Attributes.Name
	mvArtistId := ""
	if len(MVInfo.Data[0].Relationships.Artists.Data) > 0 {
		mvArtistId = MVInfo.Data[0].Relationships.Artists.Data[0].ID
	}

	r.State.AddedTracks = append(r.State.AddedTracks, AddedTrack{
		Path:     mvOutPath,
		Artist:   mvArtistName,
		ArtistID: mvArtistId,
		Album:    mvAlbumName,
		Song:     mvName,
	})

	return nil
}

func (r *Runner) writeMVMP4Tags(path string, mvInfo *ampapi.MusicVideoResp, track *model.Track, coverPath string) error {
	if mvInfo == nil || len(mvInfo.Data) == 0 {
		return errors.New("music video response contains no data")
	}
	attrs := mvInfo.Data[0].Attributes

	tags := &mp4tag.MP4Tags{
		Title:       attrs.Name,
		Artist:      attrs.ArtistName,
		Album:       attrs.AlbumName,
		CustomGenre: firstGenre(attrs.GenreNames),
		Date:        attrs.ReleaseDate,
		TrackNumber: int16(attrs.TrackNumber),
		DiscNumber:  int16(attrs.DiscNumber),
		Custom: map[string]string{
			"PERFORMER":   attrs.ArtistName,
			"RELEASETIME": attrs.ReleaseDate,
			"ISRC":        attrs.Isrc,
		},
	}

	switch {
	case track != nil && (track.PreType == "playlists" || track.PreType == "stations") && !r.Config.UseSongInfoForPlaylist:
		tags.Album = track.PlaylistData.Attributes.Name
		tags.DiscNumber = 1
		tags.DiscTotal = 1
		tags.TrackNumber = int16(track.TaskNum)
		tags.TrackTotal = int16(track.TaskTotal)
		tags.AlbumArtist = track.PlaylistData.Attributes.ArtistName
		tags.Custom["PERFORMER"] = track.Resp.Attributes.ArtistName
	case track != nil:
		tags.Album = track.AlbumData.Attributes.Name
		tags.DiscNumber = int16(track.Resp.Attributes.DiscNumber)
		tags.DiscTotal = int16(track.DiscTotal)
		tags.TrackNumber = int16(track.Resp.Attributes.TrackNumber)
		tags.TrackTotal = int16(track.AlbumData.Attributes.TrackCount)
		tags.AlbumArtist = track.AlbumData.Attributes.ArtistName
		tags.Custom["PERFORMER"] = track.Resp.Attributes.ArtistName
		tags.Custom["UPC"] = track.AlbumData.Attributes.Upc
		tags.Custom["LABEL"] = track.AlbumData.Attributes.RecordLabel
		tags.Copyright = track.AlbumData.Attributes.Copyright
		tags.Publisher = track.AlbumData.Attributes.RecordLabel
	}

	if r.Config.TagSortOrder {
		tags.TitleSort = attrs.Name
		tags.ArtistSort = attrs.ArtistName
		tags.AlbumSort = tags.Album
		tags.AlbumArtistSort = tags.AlbumArtist
	}

	switch attrs.ContentRating {
	case "explicit":
		tags.ItunesAdvisory = mp4tag.ItunesAdvisoryExplicit
	case "clean":
		tags.ItunesAdvisory = mp4tag.ItunesAdvisoryClean
	default:
		tags.ItunesAdvisory = mp4tag.ItunesAdvisoryNone
	}

	if r.Config.EmbedCover && coverPath != "" {
		cover, err := os.ReadFile(coverPath)
		if err != nil {
			return fmt.Errorf("read MV cover: %w", err)
		}
		tags.Pictures = []*mp4tag.MP4Picture{{
			Format: mp4tag.ImageTypeAuto,
			Data:   cover,
		}}
	}

	mp4, err := mp4tag.Open(path)
	if err != nil {
		return err
	}
	defer mp4.Close()
	return mp4.Write(tags, []string{})
}

// EmbedSubtitlesInMV embeds SRT subtitles into an MP4 music video using ffmpeg.
// Applies -itsoffset 0.250 to shift subtitle timing by 250ms.
// Returns the path to the output file (overwrites input on success).
func EmbedSubtitlesInMV(videoPath, subtitlePath, language string) (string, error) {
	if _, err := os.Stat(videoPath); err != nil {
		return "", fmt.Errorf("video file not found: %w", err)
	}
	if _, err := os.Stat(subtitlePath); err != nil {
		return "", fmt.Errorf("subtitle file not found: %w", err)
	}

	if language == "" {
		language = "eng"
	}

	// Create temp file in same directory (for atomic rename on success).
	dir := filepath.Dir(videoPath)
	tmpFile, err := os.CreateTemp(dir, "*.mp4")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	// ffmpeg: embed subtitles with -itsoffset for timing, -c copy for zero re-encode.
	cmd := exec.Command(
		"ffmpeg",
		"-y",
		"-itsoffset", "0.250",
		"-i", videoPath,
		"-i", subtitlePath,
		"-map", "0:v", // video stream
		"-map", "0:a", // audio stream
		"-map", "1:s", // subtitle stream (from .srt)
		"-c", "copy", // copy video/audio streams unchanged
		"-c:s", "mov_text", // subtitle codec (tx3g in MP4)
		"-metadata:s:s:0", "language="+language,
		tmpPath,
	)

	if err := cmd.Run(); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("ffmpeg subtitle embedding failed: %w", err)
	}

	// Atomic rename: replace original with temp on success.
	if err := os.Rename(tmpPath, videoPath); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to replace original file: %w", err)
	}

	return videoPath, nil
}
