package polyfin

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

// IdentityProvider is the OpenID Connect provider block of
// PUT /admin/api/settings (Settings > Security > Identity provider). The client
// secret is deliberately absent: the API only accepts it, never returns it.
type IdentityProvider struct {
	Issuer         string `json:"providerIssuer"`
	Name           string `json:"providerName"`
	ClientID       string `json:"providerClientId"`
	UserClaim      string `json:"providerUserClaim"`
	CreateAccounts bool   `json:"providerCreateAccounts"`
	AdminClaim     string `json:"providerAdminClaim"`
	AdminValue     string `json:"providerAdminValue"`
	SecretSet      bool   `json:"providerClientSecretSet"`
	RedirectURI    string `json:"providerRedirectUri"`
}

// GetIdentityProvider reads the provider settings (no secret is returned).
func (c *Client) GetIdentityProvider(ctx context.Context) (IdentityProvider, error) {
	if err := c.CheckAdmin(ctx); err != nil {
		return IdentityProvider{}, err
	}
	var out IdentityProvider
	if _, err := c.request(ctx, "GET", "/admin/api/settings", nil, &out); err != nil {
		return IdentityProvider{}, err
	}
	return out, nil
}

// SetIdentityProvider writes the provider settings. The secret is sent only
// when non-nil (an empty string removes it, as upstream defines); a nil secret
// leaves the stored one untouched because omitted keys are preserved.
func (c *Client) SetIdentityProvider(ctx context.Context, p IdentityProvider, secret *string) error {
	if err := validateIdentityProvider(p); err != nil {
		return err
	}
	if err := c.CheckAdmin(ctx); err != nil {
		return err
	}
	fields := map[string]any{
		"providerIssuer":         p.Issuer,
		"providerName":           p.Name,
		"providerClientId":       p.ClientID,
		"providerUserClaim":      p.UserClaim,
		"providerCreateAccounts": p.CreateAccounts,
		"providerAdminClaim":     p.AdminClaim,
		"providerAdminValue":     p.AdminValue,
	}
	if secret != nil {
		fields["providerClientSecret"] = *secret
	}
	if err := c.putSetting(ctx, fields); err != nil {
		return err
	}
	got, err := c.GetIdentityProvider(ctx)
	if err != nil {
		return err
	}
	want := p
	want.SecretSet, want.RedirectURI = got.SecretSet, got.RedirectURI
	if got != want {
		return errors.New("Polyfin identity provider readback mismatch (values suppressed)")
	}
	if secret != nil && (*secret != "") != got.SecretSet {
		return errors.New("Polyfin identity provider client secret readback mismatch")
	}
	return nil
}

func validateIdentityProvider(p IdentityProvider) error {
	if p.Issuer != "" {
		u, err := url.Parse(p.Issuer)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.RawQuery != "" || u.Fragment != "" {
			return errors.New("issuer must be an https URL without query or fragment")
		}
		if p.ClientID == "" {
			return errors.New("client_id is required when issuer is set")
		}
	}
	if len(p.Name) > 64 {
		return fmt.Errorf("button name is limited to 64 bytes")
	}
	if (p.AdminClaim == "") != (p.AdminValue == "") {
		return errors.New("admin_claim and admin_value must be set together")
	}
	return nil
}
