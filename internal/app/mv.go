package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"amdl/internal/amp-api"
	mvmedia "amdl/internal/media/mv"
	"amdl/internal/model"
	playreadyrip "amdl/internal/playready-rip"
	widevinerip "amdl/internal/widevine-rip"
	"amdl/internal/widevine-rip/runv5"
	"amdl/internal/wrapper"

	"github.com/itouakirai/go-mp4tag"
)

func (r *Runner) mvDownloader(adamID string, saveDir string, token string, storefront string, track *model.Track) error {
	MVInfo, err := ampapi.GetMusicVideoResp(storefront, adamID, r.Config.General.Language, token)
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

	tempDir := os.TempDir()
	if r.TempMgr != nil {
		tempDir = r.TempMgr.RootDir()
	}

	vidPath := filepath.Join(tempDir, fmt.Sprintf("%s_vid.mp4", adamID))
	audPath := filepath.Join(tempDir, fmt.Sprintf("%s_aud.mp4", adamID))
	if r.TempMgr != nil {
		_, _ = r.TempMgr.NewFilePath(fmt.Sprintf("%s_vid.mp4", adamID))
		_, _ = r.TempMgr.NewFilePath(fmt.Sprintf("%s_aud.mp4", adamID))
	}
	defer func() {
		if r.TempMgr != nil {
			_ = r.TempMgr.RemoveFile(vidPath)
			_ = r.TempMgr.RemoveFile(audPath)
		} else {
			_ = os.Remove(vidPath)
			_ = os.Remove(audPath)
		}
	}()

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

	mvm3u8url, err := wrapper.GetWebplayback(r.Config.General.LiteServer, adamID)
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
	ctx := context.Background()
	var videoStream widevinerip.EncryptedStream
	if usePlayReady {
		fmt.Println("Video DRM: PlayReady")
		videoStream, err = playreadyrip.FetchStream(ctx, adamID, videom3u8url, r.Config.General.LiteServer)
	} else {
		videoStream, err = runv5.FetchStream(ctx, adamID, videom3u8url, r.Config.General.LiteServer)
	}
	if err != nil {
		return fmt.Errorf("download video stream: %w", err)
	}
	if err := widevinerip.DownloadAndDecryptStream(ctx, videoStream, vidPath); err != nil {
		return fmt.Errorf("write video stream: %w", err)
	}
	var covPath string
	if r.Config.Metadata.Artwork.Embed {
		thumbURL := MVInfo.Data[0].Attributes.Artwork.URL
		baseThumbName := forbiddenNames.ReplaceAllString(mvSaveName, "_") + "_thumbnail"
		covPath, err = r.writeCover(tempDir, baseThumbName, thumbURL)
		if err != nil {
			fmt.Println("Failed to save MV thumbnail:", err)
			covPath = ""
		} else {
			defer func() {
				if r.TempMgr != nil {
					_ = r.TempMgr.RemoveFile(covPath)
				} else {
					_ = os.Remove(covPath)
				}
			}()
		}
	}

	fmt.Printf("MV Remuxing...")
	if err := mvmedia.Mux(vidPath, audPath, mvOutPath); err != nil {
		fmt.Printf("MV mux failed: %v\n", err)
		return err
	}
	fmt.Printf("\rMV Remuxed.   \n")

	// Subtitle handling: extract, download (WebVTT / SRT) or extract in-stream CC, optionally save/embed.
	if r.Config.Media.MV.EmbedSubtitles || r.Config.Media.MV.ExtractSubtitles {
		var subsToEmbed []SubtitleTrack
		baseMVName := forbiddenNames.ReplaceAllString(mvSaveName, "_")

		// 1. Check for external subtitle tracks in master HLS playlist (WebVTT).
		hlsSubs, err := r.extractMvSubtitles(mvm3u8url)
		var matchedHLSSubs []SubtitleRendition
		if err == nil && len(hlsSubs) > 0 {
			matchedHLSSubs = FilterSubtitleRenditions(hlsSubs, r.Config.Media.MV.SubtitleLanguages)
		}

		if len(matchedHLSSubs) > 0 {
			for _, sub := range matchedHLSSubs {
				tempVTTPath := filepath.Join(tempDir, fmt.Sprintf("%s.%s.vtt", baseMVName, sub.Language))
				if r.TempMgr != nil {
					_, _ = r.TempMgr.NewFilePath(fmt.Sprintf("%s.%s.vtt", baseMVName, sub.Language))
				}
				defer func(p string) {
					if r.TempMgr != nil {
						_ = r.TempMgr.RemoveFile(p)
					} else {
						_ = os.Remove(p)
					}
				}(tempVTTPath)

				if err := downloadWebVTT(sub.URL, tempVTTPath); err != nil {
					fmt.Printf("⚠ Failed to download subtitle (%s): %v\n", sub.Language, err)
					continue
				}

				// If extraction enabled, save sidecar files next to MV
				if r.Config.Media.MV.ExtractSubtitles {
					sidecarVTT := filepath.Join(saveDir, fmt.Sprintf("%s.%s.vtt", baseMVName, sub.Language))
					_ = copyFile(tempVTTPath, sidecarVTT)

					sidecarSRT := filepath.Join(saveDir, fmt.Sprintf("%s.%s.srt", baseMVName, sub.Language))
					if vttData, err := os.ReadFile(tempVTTPath); err == nil {
						srtData := webvttToSRT(string(vttData))
						_ = os.WriteFile(sidecarSRT, []byte(srtData), 0644)
					}
					fmt.Printf("Extracted subtitle (%s): %s\n", sub.Language, filepath.Base(sidecarVTT))
				}

				if r.Config.Media.MV.EmbedSubtitles {
					subsToEmbed = append(subsToEmbed, SubtitleTrack{
						Path:     tempVTTPath,
						Language: sub.Language,
						Title:    sub.Name,
					})
				}
			}
		}

		// 2. If no playlist subtitles found, extract in-stream closed captions (EIA-608 / CEA-708) from video.
		if len(subsToEmbed) == 0 && (!r.Config.Media.MV.ExtractSubtitles || len(matchedHLSSubs) == 0) {
			if isFFmpegAvailable() {
				videoSource := mvOutPath
				if exists, _ := fileExists(videoSource); !exists {
					videoSource = vidPath
				}
				ccSubs, err := ExtractSubtitlesFromVideo(videoSource, tempDir, baseMVName)
				if err != nil {
					fmt.Printf("⚠ Failed to extract closed captions: %v\n", err)
				} else if len(ccSubs) > 0 {
					matchedCC := FilterExtractedSubtitles(ccSubs, r.Config.Media.MV.SubtitleLanguages)
					for _, cc := range matchedCC {
						defer func(p string) {
							if r.TempMgr != nil {
								_ = r.TempMgr.RemoveFile(p)
							} else {
								_ = os.Remove(p)
							}
						}(cc.Path)
						if r.Config.Media.MV.ExtractSubtitles {
							sidecarSRT := filepath.Join(saveDir, fmt.Sprintf("%s.%s.srt", baseMVName, cc.Lang))
							_ = copyFile(cc.Path, sidecarSRT)
							sidecarVTT := filepath.Join(saveDir, fmt.Sprintf("%s.%s.vtt", baseMVName, cc.Lang))
							if srtData, err := os.ReadFile(cc.Path); err == nil {
								_ = os.WriteFile(sidecarVTT, []byte(srtToWebVTT(string(srtData))), 0644)
							}
							fmt.Printf("Extracted closed caption (%s): %s\n", cc.Lang, filepath.Base(sidecarSRT))
						}
						if r.Config.Media.MV.EmbedSubtitles {
							subsToEmbed = append(subsToEmbed, SubtitleTrack{
								Path:     cc.Path,
								Language: cc.Lang,
								Title:    cc.Lang,
							})
						}
					}
				}
			}
		}

		if len(matchedHLSSubs) == 0 && len(subsToEmbed) == 0 {
			fmt.Println("No subtitle tracks found in MV.")
		}

		// 3. Embed subtitles if requested and found
		if r.Config.Media.MV.EmbedSubtitles && len(subsToEmbed) > 0 {
			fmt.Print("Embedding subtitles...")
			if err := r.EmbedSubtitlesInMV(mvOutPath, subsToEmbed); err != nil {
				fmt.Printf("\r⚠ Subtitle embed failed: %v\n", err)
			} else {
				fmt.Print("\rSubtitles embedded.   \n")
			}
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
	case track != nil && (track.PreType == "playlists" || track.PreType == "stations") && !r.Config.Metadata.Tags.UseSongInfoForPlaylist:
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

	if r.Config.Metadata.Tags.SortOrder {
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

	if r.Config.Metadata.Artwork.Embed && coverPath != "" {
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
