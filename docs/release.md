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

The public Registry publication flow does not use a separate Registry API
token. The owner signs in with GitHub, adds the public RSA/DSA signing key in
Registry User Settings, publishes `SmadjaPaul/polyfin`, and accepts the
repository webhook and terms. The release workflow uses the repository
`GITHUB_TOKEN` only to create the GitHub Release; a local GoReleaser run would
need a separate GitHub token with `public_repo` scope.

## Artifact contract

GoReleaser produces archives named
`terraform-provider-polyfin_<version>_<os>_<arch>.zip`, a
`terraform-provider-polyfin_<version>_SHA256SUMS` file, its detached binary
signature, and `terraform-provider-polyfin_<version>_manifest.json`. The
manifest declares Terraform Plugin Framework protocol `6.0`.

The Registry release must remain a published, non-draft GitHub Release. Never
replace an existing tag or release asset: checksum provenance is immutable.

## Rollback

If a provider release causes a plan or apply regression:

1. Stop Atlantis plans/applies for the affected Polyfin root.
2. Pin the last known-good provider version in `terraform/polyfin`.
3. Regenerate the root lock file only from that published version and review
   the resulting plan.
4. Restore the previous HCL configuration if needed.

The resource refuses Terraform destruction, so rollback must not remove native
Polyfin addons. Keep the failed release and its evidence for diagnosis; do
not delete or overwrite the Registry tag.
