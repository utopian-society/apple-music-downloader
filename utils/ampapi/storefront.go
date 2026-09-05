package ampapi

// StorefrontResp represents the response from the Apple Music storefronts API
type StorefrontResp struct {
	Data []struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Name                  string   `json:"name"`
			DefaultLanguageTag    string   `json:"defaultLanguageTag"`
			SupportedLanguageTags []string `json:"supportedLanguageTags"`
			ExplicitContentPolicy string   `json:"explicitContentPolicy"`
		} `json:"attributes"`
	} `json:"data"`
}
