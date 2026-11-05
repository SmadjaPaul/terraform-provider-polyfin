package provider

import "testing"

func TestManifestURLValidation(t *testing.T) {
 valid:="https://example.org/user-token/manifest.json"
 if !validateURL(valid) {t.Fatal("valid Stremio manifest URL rejected")}
 for _,input:=range []string{
  "http://example.org/user/manifest.json",
  "https://user:pass@example.org/user/manifest.json",
  "https://example.org/stream/movie/tt42.json",
  "https://example.org/user/manifest.json#fragment",
  "https://example.org/missing/manifest.json\nInjected: evil",
  "file:///etc/passwd",
  "https://",
 }{
  if validateURL(input){t.Fatalf("accepted malformed URL: %q",input)}
 }
}
