package playreadyrip

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestLicenseUsesPlayReadyType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/license" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var request map[string]string
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if request["drm-type"] != "pr" {
			t.Errorf("unexpected drm-type: %q", request["drm-type"])
		}
		if request["adamId"] != "123" || request["challenge"] != "challenge" || request["uri"] != "data:test" {
			t.Errorf("unexpected request: %#v", request)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0,
			"data": map[string]string{
				"license": base64.StdEncoding.EncodeToString([]byte("license-xml")),
			},
		})
	}))
	defer server.Close()

	license, err := requestLicense("123", "challenge", "data:test", server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(license) != "license-xml" {
		t.Fatalf("unexpected license: %q", license)
	}
}
