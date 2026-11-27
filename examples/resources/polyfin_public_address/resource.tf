resource "polyfin_public_address" "canonical" {
  # Fictitious canonical origin: a bare HTTPS address, without path,
  # query, fragment or trailing slash.
  public_address = "https://media.example.com"
}

# The canonical address is a singleton: the provider refuses to destroy it,
# so removing the block from the configuration keeps the Polyfin value.
