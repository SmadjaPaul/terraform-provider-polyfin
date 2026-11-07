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

// GetPublicAddress reads only a non-secret setting from the administrator API.
// Other values returned by Polyfin are neither decoded nor logged.
func (c *Client) GetPublicAddress(ctx context.Context) (string, error) {
    if err := c.CheckAdmin(ctx); err != nil { return "", err }
    var out struct { PublicAddress *string `json:"publicAddress"` }
    _, err := c.request(ctx, "GET", "/admin/api/settings", nil, &out)
    if err != nil { return "", err }
    if out.PublicAddress == nil { return "", errors.New("Polyfin settings response has no publicAddress") }
    return *out.PublicAddress, nil
}

// SetPublicAddress PATCH-like behavior is the documented and upstream-tested
// contract of PUT /admin/api/settings (v0.61.1): unspecified keys are preserved.
// Do not send the full settings document, which can include redacted secrets.
func (c *Client) SetPublicAddress(ctx context.Context, address string) error {
    if err := ValidatePublicAddress(address); err != nil { return err }
    if err := c.CheckAdmin(ctx); err != nil { return err }
    _, err := c.request(ctx, "PUT", "/admin/api/settings", map[string]string{"publicAddress": address}, nil)
    if err != nil { return err }
    current, err := c.GetPublicAddress(ctx)
    if err != nil { return err }
    if current != address {
        return fmt.Errorf("Polyfin public address verification mismatch (values suppressed)")
    }
    return nil
}
