data "polyfin_addons" "installed" {}

# The summary only holds id, name, kind and enabled: no manifest URL and no
# source credential ever leaves Polyfin.
output "installed_polyfin_addons" {
  value = data.polyfin_addons.installed.addons_json
}
