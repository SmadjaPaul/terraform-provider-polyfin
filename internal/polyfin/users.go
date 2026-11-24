package polyfin

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

// User is the part of a Polyfin account this provider owns. The password is
// never read back: Polyfin does not return it.
type User struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	IsAdministrator bool   `json:"isAdministrator"`
	IsHidden        bool   `json:"isHidden"`
	// Downloads is the "Can download" permission: Jellyfin clients read it as
	// CanDownload, and Music Assistant marks a track unavailable without it.
	Downloads bool `json:"downloads"`
}

// ListUsers reads GET /admin/api/users (administrator only).
func (c *Client) ListUsers(ctx context.Context) ([]User, error) {
	if err := c.CheckAdmin(ctx); err != nil {
		return nil, err
	}
	var users []User
	if _, err := c.request(ctx, "GET", "/admin/api/users", nil, &users); err != nil {
		return nil, err
	}
	return users, nil
}

// FindUser returns the account with the given id, if any.
func (c *Client) FindUser(ctx context.Context, id string) (User, bool, error) {
	users, err := c.ListUsers(ctx)
	if err != nil {
		return User{}, false, err
	}
	for _, u := range users {
		if u.ID == id {
			return u, true, nil
		}
	}
	return User{}, false, nil
}

// CreateUser creates a local account with POST /admin/api/users. The group is
// left out (no group). The password goes only in the request body.
func (c *Client) CreateUser(ctx context.Context, name, password string, admin, hidden bool) (User, error) {
	if name == "" || password == "" {
		return User{}, errors.New("Polyfin user needs a name and a password")
	}
	if err := c.CheckAdmin(ctx); err != nil {
		return User{}, err
	}
	body := map[string]any{"name": name, "password": password, "isAdministrator": admin, "isHidden": hidden}
	var created User
	if _, err := c.request(ctx, "POST", "/admin/api/users", body, &created); err != nil {
		return User{}, err
	}
	if created.ID == "" || created.Name != name || created.IsAdministrator != admin {
		return User{}, errors.New("Polyfin user readback mismatch")
	}
	return created, nil
}

// DeleteUser removes an account. A 404 means it is already gone.
func (c *Client) DeleteUser(ctx context.Context, id string) error {
	if err := c.CheckAdmin(ctx); err != nil {
		return err
	}
	code, err := c.request(ctx, "DELETE", "/admin/api/users/"+url.PathEscape(id), nil, nil)
	if err != nil && code != http.StatusNotFound {
		return err
	}
	return nil
}

// SetUserDownloads sets only the "Can download" permission of an account with
// PATCH /admin/api/users/{id}. The body carries that single field, so every
// other setting of the account stays as it is. The result is read back.
func (c *Client) SetUserDownloads(ctx context.Context, id string, allowed bool) (User, error) {
	if err := c.CheckAdmin(ctx); err != nil {
		return User{}, err
	}
	body := map[string]any{"downloads": allowed}
	if _, err := c.request(ctx, "PATCH", "/admin/api/users/"+url.PathEscape(id), body, nil); err != nil {
		return User{}, err
	}
	u, found, err := c.FindUser(ctx, id)
	if err != nil {
		return User{}, err
	}
	if !found || u.Downloads != allowed {
		return User{}, errors.New("Polyfin user downloads readback mismatch")
	}
	return u, nil
}
