resource "polyfin_addon" "example" {
  name = "My reviewed Stremio addon"
  enabled = true
  manifest_url_wo = var.addon_manifest_url
  manifest_url_wo_version = 1
}
# var.addon_manifest_url must be ephemeral / write-only capable.
# Never place an addon manifest URL (often containing a token) in Git/state.
