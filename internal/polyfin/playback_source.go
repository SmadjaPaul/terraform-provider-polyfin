package polyfin

import (
	"context"
	"errors"
	"fmt"
)

// GetSendAppsToSource reads Polyfin's server-wide "Send apps to the source"
// setting (Settings > Playback). When true, apps are redirected to the
// version's source (the debrid CDN) whoever the user is, so the video bytes
// skip the server. When false, only administrators are redirected for a server
// addon and every other user is relayed through the server.
func (c *Client) GetSendAppsToSource(ctx context.Context) (bool, error) {
	if err := c.CheckAdmin(ctx); err != nil {
		return false, err
	}
	var result struct {
		SendAppsToSource *bool `json:"sendAppsToSource"`
	}
	if _, err := c.request(ctx, "GET", "/admin/api/settings", nil, &result); err != nil {
		return false, err
	}
	if result.SendAppsToSource == nil {
		return false, errors.New("Polyfin settings missing sendAppsToSource (requires Polyfin 0.83 or later)")
	}
	return *result.SendAppsToSource, nil
}

// SetSendAppsToSource sends only sendAppsToSource: upstream's settings PUT is
// partial, so every other setting keeps its value. The value is read back.
func (c *Client) SetSendAppsToSource(ctx context.Context, enabled bool) error {
	if err := c.CheckAdmin(ctx); err != nil {
		return err
	}
	if err := c.putSetting(ctx, map[string]any{"sendAppsToSource": enabled}); err != nil {
		return err
	}
	current, err := c.GetSendAppsToSource(ctx)
	if err != nil {
		return err
	}
	if current != enabled {
		return fmt.Errorf("Polyfin sendAppsToSource readback mismatch")
	}
	return nil
}
