# terraform-provider-polyfin

A security-first OpenTofu/Terraform provider for the native **Polyfin v0.60**
administration API, published as `SmadjaPaul/polyfin` after the first qualified
GPG-signed Registry release.

## Initial scope

- `polyfin_addon`: declarative Stremio/Eclipse source with a write-only
  manifest URL, non-secret rotation counter and explicit enabled flag.
  Import uses Polyfin's **installation UUID**, not the manifest ID.
- `polyfin_addons` data source: JSON summaries of only
  `id,name,kind,enabled`, never raw manifest URLs.

The provider refuses implicit addon deletion and refuses creation when
another installed source has the same display name. It does not access
playback history, profile data, family accounts or existing databases.

## Authentication

Set either `POLYFIN_ADMIN_COOKIE_FILE` to a protected session cookie,
or both `POLYFIN_ADMIN_USERNAME_FILE` and `POLYFIN_ADMIN_PASSWORD_FILE`
for native noninteractive login from Atlantis. Never pass these as Terraform
provider attributes, shell command arguments or checked-in variables.

## Local development

```sh
go test -race ./...
go vet ./...
```

See [architecture](docs/architecture.md) and [release checklist](docs/release.md).
Production Atlantis configuration and a Registry release are **not** part of
this bootstrap. No Polyfin admin cookie, TorBox key or manifest address is
included here.

Registry documentation is available in [docs/index.md](docs/index.md),
including the write-only manifest contract, import identity and rollback
procedure.
