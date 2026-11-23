# Resource: polyfin_identity_provider

Owns only Polyfin's OpenID Connect identity provider block (Settings > Security
> Identity provider), for example single sign-on with Authentik.

```hcl
resource "polyfin_identity_provider" "server" {
  issuer     = "https://auth.smadja.dev/application/o/polyfin/"
  button_name = "Authentik"
  client_id  = "polyfin"

  client_secret_wo         = var.polyfin_oidc_client_secret
  client_secret_wo_version = 1

  user_claim      = "preferred_username"
  create_accounts = false
  admin_claim     = "groups"
  admin_value     = "authentik-admins"
}
```

`PUT /admin/api/settings` is partial: this resource sends the provider keys
(`providerIssuer`, `providerName`, `providerClientId`, `providerUserClaim`,
`providerCreateAccounts`, `providerAdminClaim`, `providerAdminValue`) plus the
four required scalar fields, then re-reads them. It never touches the public
address, accounts, account links or other settings. The public address must be
set (see `polyfin_public_address`): the sign-in button appears once issuer,
client ID and public address are all set.

The client secret is **write-only** (`client_secret_wo`): it is never stored in
the plan or state and the API never returns it. It is sent on creation and each
time `client_secret_wo_version` changes (rotation). `client_secret_set` shows
whether the server holds one. Needs OpenTofu 1.11+ or Terraform 1.11+.

## Argument Reference

- `issuer` (Required) https issuer, exactly as the discovery document gives it.
- `client_id` (Required) OIDC client ID.
- `button_name` (Optional, default `""`) Text after "Sign in with" (64 bytes max).
- `client_secret_wo` (Optional, Write-only, Sensitive) Confidential client secret.
- `client_secret_wo_version` (Optional) Change it to send the secret again.
- `user_claim` (Optional, default `preferred_username`).
- `create_accounts` (Optional, default `false`).
- `admin_claim`, `admin_value` (Optional, set together) Administrator claim.

## Attribute Reference

- `id` Singleton ID `server`; import with `id = "server"`.
- `client_secret_set` Whether Polyfin holds a client secret.
- `redirect_uri` URI to register at the provider.

Destroy fails on purpose to protect live sign-in.
