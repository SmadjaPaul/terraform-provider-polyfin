terraform {
  required_providers {
    polyfin = {
      source = "SmadjaPaul/polyfin"
      version = "~> 0.1"
    }
  }
}
provider "polyfin" {
  endpoint = "https://media.smadja.dev"
}
# Configure POLYFIN_ADMIN_COOKIE_FILE to a protected 0600 file in Atlantis.
