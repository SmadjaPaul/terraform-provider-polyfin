package polyfin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendAppsToSourceIsPartialAndVerified(t *testing.T) {
	settings := map[string]any{
		"sendAppsToSource":        false,
		"defaultAudioLanguage":    "fre",
		"publicAddress":           "https://tv.smadja.dev",
		"providerClientSecretSet": true,
		"versionOrder":            []string{"resolution-high", "source"},
	}
	puts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/admin/api/session" && r.Method == "GET":
			fmt.Fprint(w, `{"user":{"isAdministrator":true}}`)
		case r.URL.Path == "/admin/api/settings" && r.Method == "GET":
			_ = json.NewEncoder(w).Encode(settings)
		case r.URL.Path == "/admin/api/settings" && r.Method == "PUT":
			puts++
			var data map[string]bool
			if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
				t.Error(err)
			}
			if len(data) != 1 {
				t.Errorf("unwanted settings update: %v", data)
			}
			if v, ok := data["sendAppsToSource"]; ok {
				settings["sendAppsToSource"] = v
			}
			_ = json.NewEncoder(w).Encode(settings)
		default:
			http.Error(w, "not found", 404)
		}
	}))
	defer srv.Close()
	c, err := New(srv.URL, cookie(t))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := c.GetSendAppsToSource(context.Background()); err != nil || got {
		t.Fatalf("initial read %v %v", got, err)
	}
	if err := c.SetSendAppsToSource(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if puts != 1 {
		t.Fatalf("expected 1 partial PUT, got %d", puts)
	}
	if settings["publicAddress"] != "https://tv.smadja.dev" || settings["providerClientSecretSet"] != true || settings["defaultAudioLanguage"] != "fre" {
		t.Fatal("unrelated settings changed")
	}
	if got, err := c.GetSendAppsToSource(context.Background()); err != nil || !got {
		t.Fatalf("readback %v %v", got, err)
	}
}

func TestSendAppsToSourceRequiresNewServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/admin/api/session" {
			fmt.Fprint(w, `{"user":{"isAdministrator":true}}`)
			return
		}
		fmt.Fprint(w, `{"defaultAudioLanguage":""}`)
	}))
	defer srv.Close()
	c, err := New(srv.URL, cookie(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetSendAppsToSource(context.Background()); err == nil {
		t.Fatal("missing property accepted")
	}
}
