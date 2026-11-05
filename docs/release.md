# Registry release checklist

Target provider source: `registry.terraform.io/SmadjaPaul/polyfin`
(canonical lower-case service address: `registry.terraform.io/smadjapaul/polyfin`).

The release workflow is copied from terraform-provider-aiostreams:
GoReleaser archives (darwin/linux/windows/freebsd), SHA256SUMS, GPG detached
signatures, and `terraform-registry-manifest.json` on tagged releases.

Required before a first tag:
- Go tests/race and vet pass on CI.
- Verify OpenTofu/Terraform write-only attribute support on the target version.
- Qualify isolated Polyfin v0.60 against real admin sessions (no prod).
- Add `GPG_PRIVATE_KEY` and `PASSPHRASE` repository Actions secrets via
  authorized GitHub Settings UI; keep their values outside Git, state and logs.
- Register/authorize the new repository in the Terraform Registry using
  the owner's Registry account, with the public GPG key.
- Only then tag `v0.1.0`. A Github Release is not Registry registration.
