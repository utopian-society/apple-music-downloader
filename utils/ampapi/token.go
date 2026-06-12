package ampapi

import (
	"errors"
	"io"
	"net/http"
	"regexp"
)

func GetToken() (string, error) {
	req, err := http.NewRequest("GET", "https://music.apple.com", nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}


	regex := regexp.MustCompile(`/assets/index[-~.][^/"]+\.js`)
	indexJsUri := regex.FindString(string(body))
	if indexJsUri == "" {

		regexBackup := regexp.MustCompile(`/assets/index[^/"]+\.js`)
		indexJsUri = regexBackup.FindString(string(body))
	}

	if indexJsUri == "" {
		return "", errors.New("failed to find index.js asset path")
	}

	req, err = http.NewRequest("GET", "https://music.apple.com"+indexJsUri, nil)
	if err != nil {
		return "", err
	}

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}


	jwtRegex := regexp.MustCompile(`eyJ[a-zA-Z0-9-_]+\.eyJ[a-zA-Z0-9-_]+\.[a-zA-Z0-9-_]+`)
	token := jwtRegex.FindString(string(body))

	if token == "" {
		return "", errors.New("developer token not found in JS asset")
	}

	return token, nil
}
