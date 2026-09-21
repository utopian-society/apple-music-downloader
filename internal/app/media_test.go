package app

import "testing"

func TestStreamAllowedCPC(t *testing.T) {
	const master = `#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=1000,RESOLUTION=1920x1080,ALLOWED-CPC="com.microsoft.playready:SL2000,urn:uuid:edef8ba9-79d6-4ace-a3c8-27dcd51d21ed:WIDEVINE_HARDWARE"
video_1080.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=2000,RESOLUTION=3840x2160,ALLOWED-CPC="com.apple.streamingkeydelivery:Main/AppleMain,com.microsoft.playready:SL3000"
video_2160.m3u8
`

	if got := streamAllowedCPC(master, "video_1080.m3u8"); got != "com.microsoft.playready:SL2000,urn:uuid:edef8ba9-79d6-4ace-a3c8-27dcd51d21ed:WIDEVINE_HARDWARE" {
		t.Fatalf("unexpected 1080p ALLOWED-CPC: %q", got)
	}
	if got := streamAllowedCPC(master, "video_2160.m3u8"); got != "com.apple.streamingkeydelivery:Main/AppleMain,com.microsoft.playready:SL3000" {
		t.Fatalf("unexpected 2160p ALLOWED-CPC: %q", got)
	}
	if got := streamAllowedCPC(master, "missing.m3u8"); got != "" {
		t.Fatalf("expected missing variant to have empty ALLOWED-CPC, got %q", got)
	}
}
