package utils

import (
	"errors"
	"io"
	"main/utils/ampapi"
	"main/utils/structs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/zhaarey/go-mp4tag"
)

var countryNames = map[string]string{
	"ae": "UAE",
	"ag": "Antigua and Barbuda",
	"ai": "Anguilla",
	"am": "Armenia",
	"ao": "Angola",
	"ar": "Argentina",
	"at": "Austria",
	"au": "Australia",
	"az": "Azerbaijan",
	"ba": "Bosnia and Herzegovina",
	"bb": "Barbados",
	"be": "Belgium",
	"bg": "Bulgaria",
	"bh": "Bahrain",
	"bj": "Benin",
	"bm": "Bermuda",
	"bo": "Bolivia",
	"br": "Brazil",
	"bs": "Bahamas",
	"bt": "Bhutan",
	"bw": "Botswana",
	"by": "Belarus",
	"bz": "Belize",
	"ca": "Canada",
	"cd": "Democratic Republic of the Congo",
	"cg": "Republic of the Congo",
	"ch": "Switzerland",
	"ci": "Côte d'Ivoire",
	"cl": "Chile",
	"cm": "Cameroon",
	"cn": "China mainland",
	"co": "Colombia",
	"cr": "Costa Rica",
	"cv": "Cape Verde",
	"cy": "Cyprus",
	"cz": "Czech Republic",
	"de": "Germany",
	"dk": "Denmark",
	"dm": "Dominica",
	"do": "Dominican Republic",
	"dz": "Algeria",
	"ec": "Ecuador",
	"ee": "Estonia",
	"eg": "Egypt",
	"es": "Spain",
	"fj": "Fiji",
	"fi": "Finland",
	"fm": "Micronesia, Federated States of",
	"fr": "France",
	"ga": "Gabon",
	"gb": "United Kingdom",
	"gd": "Grenada",
	"ge": "Georgia",
	"gh": "Ghana",
	"gm": "Gambia",
	"gr": "Greece",
	"gt": "Guatemala",
	"gw": "Guinea-Bissau",
	"gy": "Guyana",
	"hk": "Hong Kong",
	"hn": "Honduras",
	"hr": "Croatia",
	"hu": "Hungary",
	"id": "Indonesia",
	"ie": "Ireland",
	"il": "Israel",
	"in": "India",
	"iq": "Iraq",
	"is": "Iceland",
	"it": "Italy",
	"jm": "Jamaica",
	"jo": "Jordan",
	"jp": "Japan",
	"ke": "Kenya",
	"kg": "Kyrgyzstan",
	"kh": "Cambodia",
	"kn": "St. Kitts and Nevis",
	"kr": "Korea, Republic of",
	"kw": "Kuwait",
	"ky": "Cayman Islands",
	"kz": "Kazakhstan",
	"la": "Lao People's Democratic Republic",
	"lb": "Lebanon",
	"lc": "St. Lucia",
	"lk": "Sri Lanka",
	"lr": "Liberia",
	"lt": "Lithuania",
	"lu": "Luxembourg",
	"lv": "Latvia",
	"ly": "Libya",
	"ma": "Morocco",
	"md": "Moldova",
	"me": "Montenegro",
	"mg": "Madagascar",
	"mk": "North Macedonia",
	"ml": "Mali",
	"mm": "Myanmar",
	"mn": "Mongolia",
	"mo": "Macao",
	"mr": "Mauritania",
	"ms": "Montserrat",
	"mt": "Malta",
	"mu": "Mauritius",
	"mv": "Maldives",
	"mw": "Malawi",
	"mx": "Mexico",
	"my": "Malaysia",
	"mz": "Mozambique",
	"na": "Namibia",
	"ne": "Niger",
	"ng": "Nigeria",
	"ni": "Nicaragua",
	"nl": "Netherlands",
	"no": "Norway",
	"np": "Nepal",
	"nz": "New Zealand",
	"om": "Oman",
	"pa": "Panama",
	"pe": "Peru",
	"pg": "Papua New Guinea",
	"ph": "Philippines",
	"pl": "Poland",
	"pt": "Portugal",
	"py": "Paraguay",
	"qa": "Qatar",
	"ro": "Romania",
	"rs": "Serbia",
	"ru": "Russia",
	"rw": "Rwanda",
	"sa": "Saudi Arabia",
	"sb": "Solomon Islands",
	"sc": "Seychelles",
	"se": "Sweden",
	"sg": "Singapore",
	"si": "Slovenia",
	"sk": "Slovakia",
	"sl": "Sierra Leone",
	"sn": "Senegal",
	"sr": "Suriname",
	"sv": "El Salvador",
	"sz": "Eswatini",
	"tc": "Turks and Caicos",
	"td": "Chad",
	"th": "Thailand",
	"tj": "Tajikistan",
	"tm": "Turkmenistan",
	"tn": "Tunisia",
	"to": "Tonga",
	"tr": "Türkiye",
	"tt": "Trinidad and Tobago",
	"tw": "Taiwan",
	"tz": "Tanzania",
	"ua": "Ukraine",
	"ug": "Uganda",
	"us": "United States",
	"uy": "Uruguay",
	"uz": "Uzbekistan",
	"vc": "St. Vincent and The Grenadines",
	"ve": "Venezuela",
	"vg": "British Virgin Islands",
	"vn": "Vietnam",
	"vu": "Vanuatu",
	"xk": "Kosovo",
	"ye": "Yemen",
	"za": "South Africa",
	"zm": "Zambia",
	"zw": "Zimbabwe",
}

func getCountryName(code string) string {
	code = strings.ToLower(code)
	if name, ok := countryNames[code]; ok {
		return name
	}
	// Return uppercase code if not found
	return strings.ToUpper(code)
}

// WriteCover downloads and saves the cover image for an album/track
func WriteCover(sanAlbumFolder, name string, url string, config structs.ConfigSet) (string, error) {
	covPath := filepath.Join(sanAlbumFolder, name+"."+config.CoverFormat)
	if config.CoverFormat == "original" {
		ext := strings.Split(url, "/")[len(strings.Split(url, "/"))-2]
		ext = ext[strings.LastIndex(ext, ".")+1:]
		covPath = filepath.Join(sanAlbumFolder, name+"."+ext)
	}
	exists, err := FileExists(covPath)
	if err != nil {
		return "", err
	}
	if exists {
		_ = os.Remove(covPath)
	}
	if config.CoverFormat == "png" {
		re := regexp.MustCompile(`{w}x{h}`)
		parts := re.Split(url, 2)
		url = parts[0] + "{w}x{h}" + strings.Replace(parts[1], ".jpg", ".png", 1)
	}
	url = strings.Replace(url, "{w}x{h}", config.CoverSize, 1)
	if config.CoverFormat == "original" {
		url = strings.Replace(url, "is1-ssl.mzstatic.com/image/thumb", "a5.mzstatic.com/us/r1000/0", 1)
		url = url[:strings.LastIndex(url, "/")]
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	do, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer do.Body.Close()
	if do.StatusCode != http.StatusOK {
		return "", errors.New(do.Status)
	}
	f, err := os.Create(covPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	_, err = io.Copy(f, do.Body)
	if err != nil {
		return "", err
	}
	return covPath, nil
}

// WriteLyrics writes lyrics to a file
func WriteLyrics(sanAlbumFolder, filename string, lrc string) error {
	lyricspath := filepath.Join(sanAlbumFolder, filename)
	f, err := os.Create(lyricspath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(lrc)
	if err != nil {
		return err
	}
	return nil
}

// WriteMP4Tags writes metadata tags to an MP4 file
func WriteMP4Tags(trackPath, lrc string, meta *ampapi.AlbumResp, trackNum, trackTotal int, config structs.ConfigSet) error {
	index := trackNum - 1

	// Build custom tags map
	customTags := map[string]string{
		"PERFORMER":   meta.Data[0].Relationships.Tracks.Data[index].Attributes.ArtistName,
		"RELEASETIME": meta.Data[0].Relationships.Tracks.Data[index].Attributes.ReleaseDate,
		"ISRC":        meta.Data[0].Relationships.Tracks.Data[index].Attributes.Isrc,
		"LABEL":       meta.Data[0].Attributes.RecordLabel,
		"UPC":         meta.Data[0].Attributes.Upc,
	}

	// Add Apple Music metadata
	if meta.Data[0].Relationships.Tracks.Data[index].ID != "" {
		customTags["CATALOG"] = meta.Data[0].Relationships.Tracks.Data[index].ID
	}

	if meta.Data[0].ID != "" {
		customTags["ALBUMID"] = meta.Data[0].ID
	}

	if len(meta.Data[0].Relationships.Tracks.Data[index].Relationships.Artists.Data) > 0 {
		customTags["ARTISTID"] = meta.Data[0].Relationships.Tracks.Data[index].Relationships.Artists.Data[0].ID
	}

	if strings.Contains(meta.Data[0].ID, "pl.") {
		customTags["PLAYLISTID"] = meta.Data[0].ID
	}

	if len(meta.Data[0].Relationships.Tracks.Data[index].Attributes.GenreNames) > 0 {
		customTags["GENREID"] = meta.Data[0].Relationships.Tracks.Data[index].Attributes.GenreNames[0]
	}

	if config.Storefront != "" {
		customTags["COUNTRY"] = getCountryName(config.Storefront)
	}

	customTags["PURCHASEDATE"] = time.Now().Format("2006-01-02 15:04:05")

	if meta.Data[0].Attributes.RecordLabel != "" && meta.Data[0].Relationships.Tracks.Data[index].Attributes.Isrc != "" {
		customTags["VENDOR"] = meta.Data[0].Attributes.RecordLabel + ":isrc:" + meta.Data[0].Relationships.Tracks.Data[index].Attributes.Isrc
	}

	t := &mp4tag.MP4Tags{
		Title:        meta.Data[0].Relationships.Tracks.Data[index].Attributes.Name,
		TitleSort:    meta.Data[0].Relationships.Tracks.Data[index].Attributes.Name,
		Artist:       meta.Data[0].Relationships.Tracks.Data[index].Attributes.ArtistName,
		ArtistSort:   meta.Data[0].Relationships.Tracks.Data[index].Attributes.ArtistName,
		Custom:       customTags,
		Composer:     meta.Data[0].Relationships.Tracks.Data[index].Attributes.ComposerName,
		ComposerSort: meta.Data[0].Relationships.Tracks.Data[index].Attributes.ComposerName,
		Date:         meta.Data[0].Attributes.ReleaseDate,
		CustomGenre:  meta.Data[0].Relationships.Tracks.Data[index].Attributes.GenreNames[0],
		Copyright:    meta.Data[0].Attributes.Copyright,
		Publisher:    meta.Data[0].Attributes.RecordLabel,
		Lyrics:       lrc,
	}

	// Add EditorialNotes as comment if available
	if meta.Data[0].Attributes.EditorialNotes.Standard != "" {
		reHTML := regexp.MustCompile("<[^>]*>")
		textWithoutHTML := reHTML.ReplaceAllString(meta.Data[0].Attributes.EditorialNotes.Standard, "")
		reNewlines := regexp.MustCompile(`\n{2,}`)
		cleanComment := reNewlines.ReplaceAllString(textWithoutHTML, "\n")
		t.Comment = strings.TrimSpace(cleanComment)
	}

	if !strings.Contains(meta.Data[0].ID, "pl.") {
		albumID, err := strconv.ParseUint(meta.Data[0].ID, 10, 32)
		if err == nil {
			t.ItunesAlbumID = int32(albumID)
		}
	}

	if len(meta.Data[0].Relationships.Artists.Data) > 0 {
		if len(meta.Data[0].Relationships.Tracks.Data[index].Relationships.Artists.Data) > 0 {
			artistID, err := strconv.ParseUint(meta.Data[0].Relationships.Tracks.Data[index].Relationships.Artists.Data[0].ID, 10, 32)
			if err == nil {
				t.ItunesArtistID = int32(artistID)
			}
		}
	}

	if strings.Contains(meta.Data[0].ID, "pl.") && !config.UseSongInfoForPlaylist {
		t.DiscNumber = 1
		t.DiscTotal = 1
		t.TrackNumber = int16(trackNum)
		t.TrackTotal = int16(trackTotal)
		t.Album = meta.Data[0].Attributes.Name
		t.AlbumSort = meta.Data[0].Attributes.Name
		t.AlbumArtist = meta.Data[0].Attributes.ArtistName
		t.AlbumArtistSort = meta.Data[0].Attributes.ArtistName
	} else if strings.Contains(meta.Data[0].ID, "pl.") && config.UseSongInfoForPlaylist {
		t.DiscNumber = int16(meta.Data[0].Relationships.Tracks.Data[index].Attributes.DiscNumber)
		t.DiscTotal = int16(meta.Data[0].Relationships.Tracks.Data[trackTotal-1].Attributes.DiscNumber)
		t.TrackNumber = int16(meta.Data[0].Relationships.Tracks.Data[index].Attributes.TrackNumber)
		t.TrackTotal = int16(trackTotal)
		t.Album = meta.Data[0].Relationships.Tracks.Data[index].Attributes.AlbumName
		t.AlbumSort = meta.Data[0].Relationships.Tracks.Data[index].Attributes.AlbumName
		t.AlbumArtist = meta.Data[0].Relationships.Tracks.Data[index].Relationships.Albums.Data[0].Attributes.ArtistName
		t.AlbumArtistSort = meta.Data[0].Relationships.Tracks.Data[index].Relationships.Albums.Data[0].Attributes.ArtistName
	} else {
		t.DiscNumber = int16(meta.Data[0].Relationships.Tracks.Data[index].Attributes.DiscNumber)
		t.DiscTotal = int16(meta.Data[0].Relationships.Tracks.Data[trackTotal-1].Attributes.DiscNumber)
		t.TrackNumber = int16(meta.Data[0].Relationships.Tracks.Data[index].Attributes.TrackNumber)
		t.TrackTotal = int16(trackTotal)
		t.Album = meta.Data[0].Relationships.Tracks.Data[index].Attributes.AlbumName
		t.AlbumSort = meta.Data[0].Relationships.Tracks.Data[index].Attributes.AlbumName
		t.AlbumArtist = meta.Data[0].Attributes.ArtistName
		t.AlbumArtistSort = meta.Data[0].Attributes.ArtistName
	}

	if meta.Data[0].Relationships.Tracks.Data[index].Attributes.ContentRating == "explicit" {
		t.ItunesAdvisory = mp4tag.ItunesAdvisoryExplicit
	} else if meta.Data[0].Relationships.Tracks.Data[index].Attributes.ContentRating == "clean" {
		t.ItunesAdvisory = mp4tag.ItunesAdvisoryClean
	} else {
		t.ItunesAdvisory = mp4tag.ItunesAdvisoryNone
	}

	mp4, err := mp4tag.Open(trackPath)
	if err != nil {
		return err
	}
	defer mp4.Close()
	err = mp4.Write(t, []string{})
	if err != nil {
		return err
	}
	return nil
}

// FileExists checks if a file exists and is not a directory
func FileExists(path string) (bool, error) {
	f, err := os.Stat(path)
	if err == nil {
		return !f.IsDir(), nil
	} else if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
