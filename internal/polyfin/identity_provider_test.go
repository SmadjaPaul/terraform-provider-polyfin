package polyfin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIdentityProviderPartialSettings(t *testing.T) {
	settings := map[string]any{
		"serverName": "Polyfin Media", "language": "fr", "quickConnectEnabled": true, "legacyAuthorization": false,
		"publicAddress": "https://tv.smadja.dev", "providerRedirectUri": "https://tv.smadja.dev/admin/api/provider/callback",
		"providerIssuer": "", "providerName": "", "providerClientId": "", "providerUserClaim": "preferred_username",
		"providerCreateAccounts": false, "providerAdminClaim": "", "providerAdminValue": "", "providerClientSecretSet": false,
		"versionOrder": []string{"language"},
	}
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/admin/api/session":
			fmt.Fprint(w, `{"user":{"isAdministrator":true}}`)
		case r.URL.Path == "/admin/api/settings" && r.Method == "GET":
			_ = json.NewEncoder(w).Encode(settings)
		case r.URL.Path == "/admin/api/settings" && r.Method == "PUT":
			var data map[string]any
			if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
				t.Error(err)
			}
			bodies = append(bodies, data)
			for k, v := range data {
				if k == "providerClientSecret" {
					settings["providerClientSecretSet"] = v != ""
					continue
				}
				settings[k] = v
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
	ctx := context.Background()
	p := IdentityProvider{
		Issuer: "https://auth.smadja.dev/application/o/polyfin/", Name: "Authentik", ClientID: "polyfin",
		UserClaim: "preferred_username", CreateAccounts: true, AdminClaim: "groups", AdminValue: "authentik-admins",
	}
	secret := "synthetic-secret"
	if err := c.SetIdentityProvider(ctx, p, &secret); err != nil {
		t.Fatal(err)
	}
	if bodies[0]["providerClientSecret"] != secret || bodies[0]["serverName"] != "Polyfin Media" {
		t.Fatalf("first PUT must carry the secret and required fields: %v", bodies[0])
	}
	if _, has := bodies[0]["publicAddress"]; has {
		t.Fatal("public address must not be sent")
	}
	if err := c.SetIdentityProvider(ctx, p, nil); err != nil {
		t.Fatal(err)
	}
	if _, has := bodies[1]["providerClientSecret"]; has {
		t.Fatal("a nil secret must not be sent")
	}
	if settings["providerClientSecretSet"] != true || settings["publicAddress"] != "https://tv.smadja.dev" {
		t.Fatal("stored secret or public address changed")
	}
	got, err := c.GetIdentityProvider(ctx)
	if err != nil || got.Issuer != p.Issuer || !got.SecretSet || got.RedirectURI == "" {
		t.Fatalf("readback %#v %v", got, err)
	}
}

func TestIdentityProviderValidation(t *testing.T) {
	bad := []IdentityProvider{
		{Issuer: "http://auth.example.com/", ClientID: "x"},
		{Issuer: "https://auth.example.com/?a=b", ClientID: "x"},
		{Issuer: "https://auth.example.com/"},
		{Issuer: "https://auth.example.com/", ClientID: "x", AdminClaim: "groups"},
	}
	for _, p := range bad {
		if err := validateIdentityProvider(p); err == nil {
			t.Fatalf("accepted %#v", p)
		}
	}
	if err := validateIdentityProvider(IdentityProvider{}); err != nil {
		t.Fatal(err)
	}
}
