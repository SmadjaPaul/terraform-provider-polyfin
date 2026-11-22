# Resource: polyfin_known_proxies

Owns only Polyfin's server-wide **Known proxies** (Settings > Security): the
reverse proxies whose `X-Forwarded-*` headers name a request's client. Without
the Gateway in this list, every request through it counts as coming from the
Gateway, so local-network rules and sign-in throttling see the wrong client.

```hcl
resource "polyfin_known_proxies" "server" {
  entries = ["10.244.0.0/16"]
}
```

- `entries`: IP addresses and CIDR networks first, then host names, at most 64.
  Write them in Polyfin's canonical form (network host bits cleared, a single
  address without prefix length, host names in lower case, each once): the
  provider refuses another spelling so that a refresh never shows a diff.
- An empty list trusts no forwarded header (Polyfin's default).
- Only `knownProxies` is sent (partial `PUT /admin/api/settings`) and read back.
- Import with ID `server`. Destroy is refused: the setting stays in Polyfin.
