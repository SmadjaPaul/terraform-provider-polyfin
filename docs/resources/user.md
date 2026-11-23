# Resource: polyfin_user

Creates one local Polyfin account through `POST /admin/api/users` (administrator
session required). Intended for service accounts such as `svc-music-assistant`.

The provider does not manage groups, parental control, devices or
identity-provider links. Every attribute forces a replacement: the account is
created once, and rotating its password means recreating it.

The password is **write-only** (`password_wo`): it is sent once at creation and
never stored in the plan or the state. It needs OpenTofu 1.11+ or Terraform 1.11+.
Generate it with `random_password` and push it to the secret store from the same
root; never put it in Git.

## Example Usage

See `examples/resources/polyfin_user/resource.tf`.

## Argument Reference

- `name` (Required) Sign-in name.
- `password_wo` (Required, Write-only, Sensitive) Initial password.
- `is_administrator` (Required) Administrator flag.
- `is_hidden` (Required) Hide the account from sign-in screens.

## Attribute Reference

- `id` Polyfin user id. Import: `tofu import polyfin_user.name <id>` (the password is then unknown).

Destroying the resource deletes the account (`DELETE /admin/api/users/{id}`).
