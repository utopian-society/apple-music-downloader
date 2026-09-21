package widevinerip

import "testing"

func TestFindKeyURIByFormat(t *testing.T) {
	const playlist = `#EXTM3U
#EXT-X-KEY:METHOD=SAMPLE-AES,URI="skd://itunes.apple.com/p996363250/c2",KEYFORMAT="com.apple.streamingkeydelivery",KEYFORMATVERSIONS="2"
#EXT-X-KEY:METHOD=SAMPLE-AES,URI="data:text/plain;charset=UTF-16;base64,playready",KEYFORMAT="com.microsoft.playready",KEYFORMATVERSIONS="2"
#EXT-X-KEY:METHOD=SAMPLE-AES,URI="data:text/plain;base64,widevine",KEYFORMAT="urn:uuid:edef8ba9-79d6-4ace-a3c8-27dcd51d21ed",KEYFORMATVERSIONS="2"
`

	if got := findKeyURIByFormat(playlist, "com.microsoft.playready"); got != "data:text/plain;charset=UTF-16;base64,playready" {
		t.Fatalf("unexpected PlayReady key URI: %q", got)
	}
	if got := findKeyURIByFormat(playlist, "missing"); got != "" {
		t.Fatalf("expected missing key format to return empty string, got %q", got)
	}
}

func TestSplitKeyURI(t *testing.T) {
	prefix, payload, err := splitKeyURI("data:text/plain;charset=UTF-16;base64,abc")
	if err != nil {
		t.Fatal(err)
	}
	if prefix != "data:text/plain;charset=UTF-16;base64" || payload != "abc" {
		t.Fatalf("unexpected split result: %q %q", prefix, payload)
	}

	if _, _, err := splitKeyURI("not-a-data-uri"); err == nil {
		t.Fatal("expected invalid key URI to fail")
	}
}
