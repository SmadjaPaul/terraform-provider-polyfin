package polyfin

import (
	"context"
	"errors"
)

// putSetting sends one managed setting to PUT /admin/api/settings together
// with the four required scalar fields Polyfin validates on every PUT
// (serverName, quickConnectEnabled, legacyAuthorization, language). Without
// them the server answers 400 invalid_server_name / invalid_language even for
// a one-key body. They are re-sent unchanged; the full settings document,
// which holds redacted secrets, is never round-tripped.
func (c *Client) putSetting(ctx context.Context, fields map[string]any) error {
	var base publicAddressSettings
	if _, err := c.request(ctx, "GET", "/admin/api/settings", nil, &base); err != nil {
		return err
	}
	if base.ServerName == "" || base.Language == "" {
		return errors.New("Polyfin settings response has incomplete required fields")
	}
	payload := map[string]any{
		"serverName":          base.ServerName,
		"quickConnectEnabled": base.QuickConnectEnabled,
		"legacyAuthorization": base.LegacyAuthorization,
		"language":            base.Language,
	}
	for k, v := range fields {
		payload[k] = v
	}
	_, err := c.request(ctx, "PUT", "/admin/api/settings", payload, nil)
	return err
}
