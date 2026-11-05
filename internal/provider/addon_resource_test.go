package provider

import "testing"

func TestManifestURLValidation(t *testing.T) {
	for _, input := range []string{
		"https://example.org/user-token/manifest.json",
		"http://aiostreams.media.svc.cluster.local/user-token/manifest.json",
		"http://aiostreams.media.svc.cluster.local/user-token/manifest.json?key=opaque",
		"http://music.media.svc.cluster.local/manifest.json",
	} {
		if !validateURL(input) {
			t.Fatalf("valid addon manifest URL rejected")
		}
	}
	for _, input := range []string{
		"http://example.org/user/manifest.json",
		"http://aiostreams.media.svc.cluster.local.evil.org/user/manifest.json",
		"http://aiostreams.media.svc.cluster.local:80/user/manifest.json",
		"http://aiostreams.media.svc.cluster.local:3000/user/manifest.json",
		"http://aiostreams.svc.cluster.local/user/manifest.json",
		"http://aiostreams..svc.cluster.local/user/manifest.json",
		"http://127.0.0.1/user/manifest.json",
		"http://user:pw@aiostreams.media.svc.cluster.local/user/manifest.json",
		"https://user:pass@example.org/user/manifest.json",
		"https://example.org/stream/movie/tt42.json",
		"https://example.org/user/manifest.json#fragment",
		"https://example.org/missing/manifest.json" + string(rune(10)) + "Injected: evil",
		"file:///etc/passwd",
		"https://",
	} {
		if validateURL(input) {
			t.Fatalf("accepted malformed addon URL")
		}
	}
}
