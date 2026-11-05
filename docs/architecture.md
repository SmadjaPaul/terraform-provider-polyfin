# Polyfin 0.60 provider boundaries

Polyfin administrative API is granular, **not** a whole UserData document.
Keep the Polyfin server, database, identity and source credentials outside Terraform state.

First version: one `polyfin_addon` resource with a Polyfin-local installation ID,
and a strictly redacted `polyfin_addons` data source. A later resource may manage
the whole set of libraries, because the native API writes the list atomically.

Authentication is from `POLYFIN_ADMIN_COOKIE_FILE` only, not a Terraform
attribute or environment value containing a cookie. The cookie belongs to
a verified native administrator session and must come from protected
runtime secret material; it is **never** committed, logged or stored in state.
The API's admin session endpoint is checked on every operation.

`manifest_url_wo` is a write-only Terraform attribute and is never
placed in state. It requires Terraform >=1.11 or a compatible OpenTofu release.
`manifest_url_wo_version` is public metadata controlling explicit
rotation. The masked manifest URL cannot be compared on refresh, so this
provider does **not** claim full URL drift detection.
Do not use ordinary Terraform sensitive variables as persistent state
attributes for manifest URLs.

Deleting an addon from configuration is intentionally a hard error; no
implicit DELETE on `destroy` or resource removal. Imports must be manually
verified against an existing Polyfin-local UUID.

Native addon API contract verified against upstream tag v0.60.0:
https://github.com/moodiness/polyfin/blob/v0.60.0/internal/admin/addons.go

The provider does **not** own user accounts, OIDC identities, playback
histories, playlists, databases, family media, secrets or backups.
