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
