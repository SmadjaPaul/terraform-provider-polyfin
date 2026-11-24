# Resource: polyfin_user

Creates one local Polyfin account through `POST /admin/api/users` (administrator
session required). Intended for service accounts such as `svc-music-assistant`.

The provider does not manage groups, parental control, devices or
identity-provider links. The name, password, administrator and hidden flags force
a replacement: the account is created once, and rotating its password means
recreating it. `enable_content_downloading` is updated in place.

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
- `enable_content_downloading` (Optional, default `false`) The "Can download"
  permission (`PATCH /admin/api/users/{id}` with `{"downloads": ...}`; Jellyfin
  `CanDownload` / `EnableContentDownloading`). Music Assistant 2.10 marks every
  track of an account without it as unavailable (`No playable items found`), so
  a Music Assistant service account needs `true`. Only this permission is
  written: every other setting of the account is left as is. A change is an
  in-place update; drift made in the Polyfin interface is detected and reverted.

## Attribute Reference

- `id` Polyfin user id. Import: `tofu import polyfin_user.name <id>` (the password is then unknown).

Destroying the resource deletes the account (`DELETE /admin/api/users/{id}`).
