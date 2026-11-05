# Polyfin Provider

The Polyfin provider manages the native Polyfin 0.60 administration API.
The first release focuses on server-scoped Stremio and Eclipse addons while
keeping addon removal explicit and irreversible operations outside Terraform.

## Requirements

| Name | Version |
|------|---------|
| OpenTofu | 1.11 or later; Atlantis is qualified with 1.13.1 |
| Terraform | 1.14 or later when using Terraform instead of OpenTofu |
| Polyfin | 0.60.x native administration API |

## Authentication

The provider uses native administrator authentication. Configure either a
protected cookie file or protected username and password files in the process
environment:

```text
POLYFIN_ENDPOINT=https://polyfin.example.invalid
POLYFIN_ADMIN_USERNAME_FILE=/run/secrets/polyfin-admin-username
POLYFIN_ADMIN_PASSWORD_FILE=/run/secrets/polyfin-admin-password
```

The provider never accepts credentials as Terraform attributes. Do not put
credentials, cookies, or tokenized manifest URLs in Git, variables committed
to a repository, state, plans, or logs. Atlantis should receive these files
through its approved Doppler projection.

## Provider configuration

```terraform
terraform {
  required_providers {
    polyfin = {
      source  = "registry.terraform.io/SmadjaPaul/polyfin"
      version = "~> 0.1"
    }
  }
}

provider "polyfin" {
  endpoint = var.polyfin_endpoint
}
```

## Resources and data sources

- [`polyfin_addon`](resources/addon.md) installs and updates one native addon.
- [`polyfin_addons`](data-sources/addons.md) reads redacted addon summaries.

The provider refuses implicit destruction. Removing `polyfin_addon` from
configuration produces a diagnostic and leaves the native addon untouched.

## Versioning and rollback

Provider releases are immutable, signed GoReleaser GitHub releases. Pin an
explicit version in each root and regenerate the lock file only after the
published version is visible in the Registry. If a release must be rolled
back, stop the affected plan/apply, pin the last known-good provider version,
run a plan, and restore the previous configuration. Do not delete or replace a
published tag or release, and do not remove the native addon through
Terraform.

