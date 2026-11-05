# Data Source: polyfin_addons

Reads a redacted JSON summary of the server-scoped Polyfin addons. The result
contains only `id`, `name`, `kind`, and `enabled`; manifest URLs and source
credentials are never returned.

## Example Usage

```terraform
data "polyfin_addons" "installed" {}

output "installed_polyfin_addons" {
  value = data.polyfin_addons.installed.addons_json
}
```

## Schema

### Read-only

- `addons_json` (String) — JSON array containing only addon UUID, display name,
  native kind, and enabled state.

