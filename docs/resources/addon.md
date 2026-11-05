# Resource: polyfin_addon

Installs and manages one server-scoped Stremio or Eclipse addon in Polyfin.
The Polyfin installation UUID is the Terraform identity.

The resource is intentionally non-destructive: destroying it is refused, so a
configuration mistake cannot silently remove a native addon or its libraries.
Use the Polyfin administrator interface for a separately reviewed manual
decommission.

## Example Usage

```terraform
resource "polyfin_addon" "meteor" {
  name                    = "Meteor"
  enabled                 = true
  manifest_url_wo         = var.meteor_manifest_url
  manifest_url_wo_version = 1
}
```

`manifest_url_wo` is a write-only attribute. It is required when creating an
addon and when deliberately rotating its manifest. Increment
`manifest_url_wo_version` for each rotation. The URL must use HTTPS by default. Plain HTTP is allowed **only** for an
exact Kubernetes Service FQDN of the form `service.namespace.svc.cluster.local`
(with no explicit port), for native administrators operating in-cluster.
URLs must contain no user information or fragment and end in `/manifest.json`.
Private URLs may carry credentials in paths/queries: pass them only through
write-only values from an approved secret source, never Git or logs.

## Schema

### Required

- `name` (String) — Exact manifest name. Renaming an installed addon is not
  supported.
- `enabled` (Boolean) — Whether the native addon is enabled.

### Optional

- `manifest_url_wo` (String, Sensitive, Write-only) — Tokenized HTTPS manifest
  address. It is never persisted in state.
- `manifest_url_wo_version` (Number) — Non-sensitive rotation counter.

### Read-only

- `id` (String) — Polyfin-local installation UUID.
- `kind` (String) — Native addon kind, such as `stremio` or `eclipse`.

## Import

Import the Polyfin installation UUID, not the Stremio manifest ID:

```shell
tofu import polyfin_addon.meteor POLYFIN_INSTALLATION_UUID
```

After import, configure the non-secret `name` and `enabled` attributes. The
write-only manifest URL is not recoverable from state and is not required for
an unchanged plan.

