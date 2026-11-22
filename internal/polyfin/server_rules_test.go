package polyfin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

// fakeSettings serves a synthetic admin settings API whose PUT is partial and
// normalizes knownProxies the way upstream stores them.
func fakeSettings(t *testing.T, settings map[string]any, puts *[]map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/admin/api/session" && r.Method == "GET":
			fmt.Fprint(w, `{"user":{"isAdministrator":true}}`)
		case r.URL.Path == "/admin/api/settings" && r.Method == "GET":
			_ = json.NewEncoder(w).Encode(settings)
		case r.URL.Path == "/admin/api/settings" && r.Method == "PUT":
			var data map[string]any
			if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
				t.Error(err)
			}
			*puts = append(*puts, data)
			for k, v := range data {
				if k == "knownProxies" {
					var entries []string
					for _, e := range v.([]any) {
						entries = append(entries, e.(string))
					}
					canonical, err := CanonicalKnownProxies(entries)
					if err != nil {
						http.Error(w, `{"error":"invalid_known_proxies"}`, 400)
						return
					}
					settings[k] = canonical
					continue
				}
				settings[k] = v
			}
			_ = json.NewEncoder(w).Encode(settings)
		default:
			http.Error(w, "not found", 404)
		}
	}))
}

func baseSettings() map[string]any {
	return map[string]any{
		"knownProxies":            []string{},
		"versionOrder":            []string{"resolution-high", "source"},
		"versionCodecOrder":       []string{"av1", "hevc", "vp9", "h264", "vc1", "mpeg4", "mpeg2video"},
		"defaultAudioLanguage":    "fre",
		"sendAppsToSource":        true,
		"publicAddress":           "https://tv.smadja.dev",
		"providerClientSecretSet": true,
	}
}

func TestCanonicalKnownProxies(t *testing.T) {
	got, err := CanonicalKnownProxies([]string{"Gateway.Example", "10.244.0.161/16", "10.0.20.5", "::ffff:10.1.2.3", "10.244.0.0/16"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"10.244.0.0/16", "10.0.20.5", "10.1.2.3", "gateway.example"}
	if !slices.Equal(got, want) {
		t.Fatalf("canonical %v, want %v", got, want)
	}
	for _, bad := range [][]string{{"192.168.1.300"}, {" 10.0.0.1"}, {""}, {"fe80::1%eth0"}, {"http://proxy"}, {"-bad.example"}} {
		if _, err := CanonicalKnownProxies(bad); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	many := make([]string, MaxKnownProxies+1)
	for i := range many {
		many[i] = fmt.Sprintf("10.0.0.%d", i)
	}
	if _, err := CanonicalKnownProxies(many); err == nil {
		t.Fatal("accepted more than 64 entries")
	}
	if got, err := CanonicalKnownProxies(nil); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("empty list must stay an empty, non-nil list: %v %v", got, err)
	}
}

func TestKnownProxiesIsPartialAndVerified(t *testing.T) {
	settings := baseSettings()
	var puts []map[string]any
	srv := fakeSettings(t, settings, &puts)
	defer srv.Close()
	c, err := New(srv.URL, cookie(t))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := c.GetKnownProxies(context.Background()); err != nil || len(got) != 0 {
		t.Fatalf("initial read %v %v", got, err)
	}
	if err := c.SetKnownProxies(context.Background(), []string{"10.244.0.0/16"}); err != nil {
		t.Fatal(err)
	}
	if len(puts) != 1 || len(puts[0]) != 1 || puts[0]["knownProxies"] == nil {
		t.Fatalf("expected one PUT holding only knownProxies, got %v", puts)
	}
	if settings["publicAddress"] != "https://tv.smadja.dev" || settings["sendAppsToSource"] != true || settings["providerClientSecretSet"] != true {
		t.Fatal("unrelated settings changed")
	}
	if got, _ := c.GetKnownProxies(context.Background()); !slices.Equal(got, []string{"10.244.0.0/16"}) {
		t.Fatalf("readback %v", got)
	}
	if err := c.SetKnownProxies(context.Background(), []string{"not a proxy"}); err == nil {
		t.Fatal("invalid entry accepted")
	}
	if len(puts) != 1 {
		t.Fatal("invalid entry was sent to Polyfin")
	}
}

func TestValidateVersionOrder(t *testing.T) {
	for _, ok := range [][]string{nil, {}, {"language"}, {"language", "cached", "resolution-high", "codec", "hdr-first", "seeders"}} {
		if err := ValidateVersionOrder(ok); err != nil {
			t.Fatalf("rejected %v: %v", ok, err)
		}
	}
	for _, bad := range [][]string{{"lang"}, {"resolution-high", "resolution-low"}, {"language", "language"}, {"language", "cached", "codec", "tier", "audio", "edition", "seeders"}} {
		if err := ValidateVersionOrder(bad); err == nil {
			t.Fatalf("accepted %v", bad)
		}
	}
}

func TestVersionOrderIsPartialAndVerified(t *testing.T) {
	settings := baseSettings()
	var puts []map[string]any
	srv := fakeSettings(t, settings, &puts)
	defer srv.Close()
	c, err := New(srv.URL, cookie(t))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := c.GetVersionOrder(context.Background()); err != nil || !slices.Equal(got, []string{"resolution-high", "source"}) {
		t.Fatalf("initial read %v %v", got, err)
	}
	if err := c.SetVersionOrder(context.Background(), []string{"language", "resolution-high"}); err != nil {
		t.Fatal(err)
	}
	if len(puts) != 1 || len(puts[0]) != 1 || puts[0]["versionOrder"] == nil {
		t.Fatalf("expected one PUT holding only versionOrder, got %v", puts)
	}
	codecs, _ := json.Marshal(settings["versionCodecOrder"])
	if string(codecs) != `["av1","hevc","vp9","h264","vc1","mpeg4","mpeg2video"]` || settings["defaultAudioLanguage"] != "fre" {
		t.Fatal("unrelated settings changed")
	}
	if err := c.SetVersionOrder(context.Background(), []string{"resolution-high", "resolution-low"}); err == nil || len(puts) != 1 {
		t.Fatal("invalid order accepted or sent")
	}
}

func TestServerRulesRequireProperties(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/admin/api/session" {
			fmt.Fprint(w, `{"user":{"isAdministrator":true}}`)
			return
		}
		fmt.Fprint(w, `{"defaultAudioLanguage":"fre"}`)
	}))
	defer srv.Close()
	c, err := New(srv.URL, cookie(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetKnownProxies(context.Background()); err == nil {
		t.Fatal("missing knownProxies accepted")
	}
	if _, err := c.GetVersionOrder(context.Background()); err == nil {
		t.Fatal("missing versionOrder accepted")
	}
}
