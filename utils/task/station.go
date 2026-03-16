package task

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"

	"main/utils/ampapi"
)

type Station struct {
	Storefront string
	ID         string

	SaveDir   string
	SaveName  string
	Codec     string
	CoverPath string

	Language string
	Resp     ampapi.StationResp
	Type     string
	Name     string
	Tracks   []Track
}

func NewStation(st string, id string) *Station {
	a := new(Station)
	a.Storefront = st
	a.ID = id
	//fmt.Println("Album created")
	return a

}

func (a *Station) GetResp(mutoken, token, l string) error {
	var err error
	a.Language = l
	resp, err := ampapi.GetStationResp(a.Storefront, a.ID, a.Language, token)
	if err != nil {
		return errors.New("error getting station response")
	}
	a.Resp = *resp
	//简化高频调用名称
	a.Type = a.Resp.Data[0].Attributes.PlayParams.Format
	a.Name = a.Resp.Data[0].Attributes.Name
	if a.Type != "tracks" {
		return nil
	}
	tracksResp, err := ampapi.GetStationNextTracks(a.ID, mutoken, a.Language, token)
	if err != nil {
		return errors.New("error getting station tracks response")
	}
	//fmt.Println("Getting album response")
	//从resp中的Tracks数据中提取trackData信息到新的Track结构体中
	for i, trackData := range tracksResp.Data {
		albumResp, err := ampapi.GetAlbumRespByHref(trackData.Href, a.Language, token)
		if err != nil {
			fmt.Println("Error getting album response:", err)
			continue
		}
		albumLen := len(albumResp.Data[0].Relationships.Tracks.Data)
		a.Tracks = append(a.Tracks, Track{
			ID:         trackData.ID,
			Type:       trackData.Type,
			Name:       trackData.Attributes.Name,
			Language:   a.Language,
			Storefront: a.Storefront,

			//SaveDir:   filepath.Join(a.SaveDir, a.SaveName),
			//Codec:     a.Codec,
			TaskNum:   i + 1,
			TaskTotal: len(tracksResp.Data),
			M3u8:      trackData.Attributes.ExtendedAssetUrls.EnhancedHls,
			WebM3u8:   trackData.Attributes.ExtendedAssetUrls.EnhancedHls,
			//CoverPath: a.CoverPath,

			Resp:      trackData,
			PreType:   "stations",
			DiscTotal: albumResp.Data[0].Relationships.Tracks.Data[albumLen-1].Attributes.DiscNumber,
			PreID:     a.ID,
			AlbumData: albumResp.Data[0],
		})
		a.Tracks[i].PlaylistData.Attributes.Name = a.Name
		a.Tracks[i].PlaylistData.Attributes.ArtistName = "Apple Music Station"
	}
	return nil
}

func (a *Station) GetArtwork() string {
	return a.Resp.Data[0].Attributes.Artwork.URL
}

func (a *Station) ShowSelect() []int {
	trackTotal := len(a.Tracks)
	arr := make([]int, trackTotal)
	for i := 0; i < trackTotal; i++ {
		arr[i] = i + 1
	}
	selected := make([]int, 0, trackTotal)
	data := make([][]string, 0, trackTotal)
	for trackNum, track := range a.Tracks {
		trackNum++
		trackName := fmt.Sprintf("%s - %s", track.Resp.Attributes.Name, track.Resp.Attributes.ArtistName)
		data = append(data, []string{fmt.Sprint(trackNum),
			trackName,
			track.Resp.Attributes.ContentRating,
			track.Type})
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("", "Track Name", "Rating", "Type")
	table.Caption(tw.Caption{Text: fmt.Sprintf("Station: %d tracks", trackTotal)})
	for _, row := range data {
		if row[2] == "explicit" {
			row[2] = "E"
		} else if row[2] == "clean" {
			row[2] = "C"
		} else {
			row[2] = "None"
		}
		if row[3] == "music-videos" {
			row[3] = "MV"
		} else if row[3] == "songs" {
			row[3] = "SONG"
		}
		table.Append(row)
	}
	table.Render()
	fmt.Println("Please select from the track options above (multiple options separated by commas, ranges supported, or type 'all' to select all)")
	cyanColor := color.New(color.FgCyan)
	cyanColor.Print("select: ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println(err)
	}
	input = strings.TrimSpace(input)
	if input == "all" {
		fmt.Println("You have selected all options:")
		selected = arr
	} else {
		selectedOptions := [][]string{}
		parts := strings.Split(input, ",")
		for _, part := range parts {
			if strings.Contains(part, "-") {
				rangeParts := strings.Split(part, "-")
				selectedOptions = append(selectedOptions, rangeParts)
			} else {
				selectedOptions = append(selectedOptions, []string{part})
			}
		}
		for _, opt := range selectedOptions {
			if len(opt) == 1 {
				num, err := strconv.Atoi(opt[0])
				if err != nil {
					fmt.Println("Invalid option:", opt[0])
					continue
				}
				if num > 0 && num <= len(arr) {
					selected = append(selected, num)
				} else {
					fmt.Println("Option out of range:", opt[0])
				}
			} else if len(opt) == 2 {
				start, err1 := strconv.Atoi(opt[0])
				end, err2 := strconv.Atoi(opt[1])
				if err1 != nil || err2 != nil {
					fmt.Println("Invalid range:", opt)
					continue
				}
				if start < 1 || end > len(arr) || start > end {
					fmt.Println("Range out of range:", opt)
					continue
				}
				for i := start; i <= end; i++ {
					selected = append(selected, i)
				}
			} else {
				fmt.Println("Invalid option:", opt)
			}
		}
	}
	return selected
}
