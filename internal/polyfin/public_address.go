package polyfin

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// ValidatePublicAddress accepts one canonical HTTPS origin without credentials,
// query, fragment or path. Polyfin uses this value as the OIDC redirect origin.
func ValidatePublicAddress(address string) error {
	if address == "" || len(address) > 512 || strings.TrimSpace(address) != address || strings.ContainsAny(address, "\r\n\t") {
		return errors.New("public address must be a nonempty canonical HTTPS origin")
	}
	u, err := url.Parse(address)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil ||
		u.Path != "" || u.RawPath != "" || u.RawQuery != "" || u.Fragment != "" ||
		strings.HasSuffix(address, "/") || u.Opaque != "" {
		return errors.New("public address must be a bare HTTPS origin without credentials, path, query, fragment or trailing slash")
	}
	return nil
}

type publicAddressSettings struct {
	ServerName          string  `json:"serverName"`
	QuickConnectEnabled bool    `json:"quickConnectEnabled"`
	LegacyAuthorization bool    `json:"legacyAuthorization"`
	Language            string  `json:"language"`
	PublicAddress       *string `json:"publicAddress"`
}

// readPublicAddressSettings reads only the non-secret fields needed by
// Polyfin's settings PUT contract. Other settings returned by Polyfin are
// neither decoded nor logged.
func (c *Client) readPublicAddressSettings(ctx context.Context) (publicAddressSettings, error) {
	if err := c.CheckAdmin(ctx); err != nil {
		return publicAddressSettings{}, err
	}
	var out publicAddressSettings
	_, err := c.request(ctx, "GET", "/admin/api/settings", nil, &out)
	if err != nil {
		return publicAddressSettings{}, err
	}
	if out.PublicAddress == nil {
		return publicAddressSettings{}, errors.New("Polyfin settings response has no publicAddress")
	}
	if out.ServerName == "" || out.Language == "" {
		return publicAddressSettings{}, errors.New("Polyfin settings response has incomplete required fields")
	}
	return out, nil
}

// GetPublicAddress reads only a non-secret setting from the administrator API.
func (c *Client) GetPublicAddress(ctx context.Context) (string, error) {
	settings, err := c.readPublicAddressSettings(ctx)
	if err != nil {
		return "", err
	}
	return *settings.PublicAddress, nil
}

// SetPublicAddress PATCH-like behavior is the documented and upstream-tested
// contract of PUT /admin/api/settings (v0.61.1): unspecified keys are preserved.
// Do not send the full settings document, which can include redacted secrets.
func (c *Client) SetPublicAddress(ctx context.Context, address string) error {
	if err := ValidatePublicAddress(address); err != nil {
		return err
	}
	settings, err := c.readPublicAddressSettings(ctx)
	if err != nil {
		return err
	}
	// Polyfin's settingsJSON keeps these four legacy scalar fields as
	// required JSON values. Send only those non-secret fields plus the
	// managed address; never round-trip the full settings document.
	payload := map[string]any{
		"serverName":          settings.ServerName,
		"quickConnectEnabled": settings.QuickConnectEnabled,
		"legacyAuthorization": settings.LegacyAuthorization,
		"language":            settings.Language,
		"publicAddress":       address,
	}
	_, err = c.request(ctx, "PUT", "/admin/api/settings", payload, nil)
	if err != nil {
		return err
	}
	current, err := c.GetPublicAddress(ctx)
	if err != nil {
		return err
	}
	if current != address {
		return fmt.Errorf("Polyfin public address verification mismatch (values suppressed)")
	}
	return nil
}
