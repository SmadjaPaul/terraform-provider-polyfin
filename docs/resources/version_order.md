# Resource: polyfin_version_order

Owns only Polyfin's server-wide **version order** (Settings > Playback): up to
six criteria that rank a title's versions, each grouping them and the next
sorting within them.

```hcl
resource "polyfin_version_order" "server" {
  criteria = ["language"]
}
```

- `criteria`: one per kind among `language`, `resolution-high`,
  `resolution-low`, `size-large`, `size-small`, `size-per-minute`, `codec`,
  `hdr-first`, `hdr-last`, `source`, `tier`, `audio`, `edition`, `labels`,
  `cached`, `seeders` (`resolution-high` and `resolution-low` are one kind).
  `language` ranks the versions in the user's preferred audio language first
  (their own choice, else the server default audio language).
- An empty list keeps the addons' order.
- Only `versionOrder` is sent and read back; codec, source and edition
  sub-orders, hidden versions and label rules are not changed.
- Import with ID `server`. Destroy is refused.
