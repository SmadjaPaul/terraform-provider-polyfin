# Resource: polyfin_playback_source_policy

Owns only Polyfin's server-wide **Send apps to the source** setting
(Settings > Playback). Requires Polyfin 0.83 or later; the property is absent
on older servers and the resource then fails instead of guessing.

```hcl
resource "polyfin_playback_source_policy" "server" {
  send_apps_to_source = true
}
```

- `true`: every app is redirected to the version's source, the debrid CDN, so
  the video bytes skip the Polyfin server. The stream address, which can hold
  the key an addon was configured with, is then visible to every user's
  device.
- `false` (Polyfin's default): only administrators are redirected for a server
  addon; every other user is relayed through the server, which uses its
  bandwidth.

A version that must be converted or repackaged is read through the server
whatever this value is.

The resource sends only `sendAppsToSource` on upstream's partial
`PUT /admin/api/settings` and re-reads it to verify. It never touches the
public address, OIDC credentials, languages, version order or other settings.
Import with ID `server`. Deletion intentionally fails.

Upstream contract:
https://github.com/moodiness/polyfin/blob/v0.85.0/docs/playback.md#who-is-sent-to-the-source
