# Resource: polyfin_public_address

Manages **only** Polyfin's canonical public address used in links and OIDC callbacks.

```hcl
resource "polyfin_public_address" "canonical" {
  public_address = "https://tv.smadja.dev"
}
```

The upstream Polyfin v0.61.1 administrator API accepts partial `PUT /admin/api/settings` payloads. This resource sends **only** `{"publicAddress":"…"}` and re-reads the value. It never sends OIDC secrets or the rest of Polyfin's settings back to the server.

- Requires native Polyfin administrator authentication from protected runtime files (not Terraform variables).
- Only an HTTPS origin without user information, paths, queries, fragments or a trailing slash is accepted.
- Reads and reconciles drift, preserving all other settings.
- Singleton import ID: `server`. Run `tofu import polyfin_public_address.canonical server` only through the separately authorized production workflow.
- Removal through `tofu destroy` deliberately fails: the canonical address cannot accidentally be reset or deleted.
- Before adopting existing production settings, back up and prove backend identity, execute an exact plan, and do not enable apply based on a successful static CI alone.

Upstream contract: https://github.com/moodiness/polyfin/blob/v0.61.1/internal/admin/security_test.go
