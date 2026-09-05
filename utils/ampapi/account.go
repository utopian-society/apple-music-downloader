package ampapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

var accountHTTPClient = &http.Client{Timeout: 5 * time.Second}

func GetUserStorefront(token string, mediaUserToken string) (string, error) {
	var err error
	if token == "" {
		token, err = GetToken()
		if err != nil {
			return "", err
		}
	}
	if mediaUserToken == "" {
		return "", errors.New("media-user-token is empty")
	}

	req, err := http.NewRequest("GET", "https://amp-api.music.apple.com/v1/me/storefront", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Media-User-Token", mediaUserToken)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Origin", "https://music.apple.com")

	do, err := accountHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer do.Body.Close()
	if do.StatusCode != http.StatusOK {
		return "", errors.New(do.Status)
	}

	var obj struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(do.Body).Decode(&obj); err != nil {
		return "", err
	}
	if len(obj.Data) == 0 || strings.TrimSpace(obj.Data[0].ID) == "" {
		return "", errors.New("storefront data is empty")
	}
	return obj.Data[0].ID, nil
}
