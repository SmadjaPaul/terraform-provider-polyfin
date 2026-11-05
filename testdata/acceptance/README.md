# Polyfin v0.60.0 isolated acceptance target (PLT-142)

**Not a deployment, not an acceptance result.** This is a reproducible,
local-only Docker Compose sandbox for testing the Terraform provider against
**real upstream Polyfin v0.60.0**, not a synthetic HTTP stub. Both image
digests are pinned to official releases. Polyfin's existing Kubernetes pilot
currently runs a **different downstream version**; do not test this provider
against the family server.

Images:
- Polyfin 0.60.0 digest `sha256:e2b1d162…125a6` from
  https://github.com/moodiness/polyfin/releases/tag/v0.60.0
- PostgreSQL 18.6 digest `sha256:5a5a84b1…67f298f2da9722` from
  https://github.com/moodiness/polyfin/blob/v0.60.0/compose.yaml

## Start (operator-controlled local Docker only)

Use a disposable machine with Docker Engine + Docker Compose. All traffic is
kept on a dedicated **internal** Docker network; the web UI binds only
`127.0.0.1:18096`. No Authentik, K8s, Doppler, production PVC, CDN,
AIOStreams or TorBox is mounted or linked.

Create an isolated private environment file locally **outside the Git tree**,
containing two new random sandbox-only values:

```sh
# Replace the example paths with a private directory on the disposable host.
umask 077
mkdir -p "$HOME/.config/polyfin-provider-acceptance"
# Create this file manually with secrets generated on the operator's host,
# NEVER from a copied production secret:
#   ACCEPTANCE_POSTGRES_PASSWORD=<random new value>
#   ACCEPTANCE_POLYFIN_SECRET_KEY=<new random value>
chmod 600 "$HOME/.config/polyfin-provider-acceptance/sandbox.env"
docker compose --env-file "$HOME/.config/polyfin-provider-acceptance/sandbox.env" \
  -f testdata/acceptance/compose.yaml config --quiet
docker compose --env-file "$HOME/.config/polyfin-provider-acceptance/sandbox.env" \
  -f testdata/acceptance/compose.yaml up -d
```

First visit `http://127.0.0.1:18096` and complete Polyfin's native
**first-administrator setup** using a synthetic sandbox-only account.
Capture a native administrator session *only in an operator-controlled
protected file*. If first boot requires the one-time setup code, inspect only
this sandbox's own Polyfin container startup logs locally. Do not publish
the code or credentials.

The provider deliberately accepts HTTP only for loopback and Kubernetes
Service qualification, so set `POLYFIN_ENDPOINT=http://127.0.0.1:18096`.
Configure either `POLYFIN_ADMIN_COOKIE_FILE` or both
`POLYFIN_ADMIN_USERNAME_FILE` and `POLYFIN_ADMIN_PASSWORD_FILE` from
restricted local files; never hardcode credentials in Terraform files or CLI
arguments.

## Acceptance matrix — requires an actual run

1. Confirm `GET /admin/api/session` returns
   `user.isAdministrator=true` for sandbox credentials.
2. Create a **synthetic Stremio/Eclipse manifest** on a separately reviewed,
   TLS-valid endpoint reachable from the sandbox network, with no live
   content, TorBox, real tracker, streaming or secret. Do not assume
   untrusted public addon manifests are safe test fixtures. If no such
   HTTPS fixture is ready, the mutating addon acceptance stays BLOCKED.
3. Use a disposable local Terraform/OpenTofu state (never the S3 backend).
   Create one `polyfin_addon` from an ephemeral `manifest_url_wo` input.
4. Prove import with native **installation UUID**, after separate
   import-state isolation, second plan no-op, enabled drift, update
   enabled flag, explicit URL version rotation, and remote removal
   refusal/no implicit destroy.
5. Check serialized state and plan metadata for absence of synthetic URL,
   credentials, cookie, query token and admin password. Disable debug
   logging of raw transport. Verify no unexpected network activity or
   external redirects; observe requests in the sandbox.
6. Run `go test -race ./...`, `go vet ./...` and publish evidence in
   PLT-142: `NATIVE_RUNTIME_PASS`, `NOOP_PLAN_PASS`,
   `IMPORT_PASS`, `SECRET_STATE_CLEAN_PASS`, `DESTROY_BLOCK_PASS`.
   Never report PASS based on only `docker compose up`, static fixture
   checks or synthetic mock server tests.

## Exit and cleanup

```sh
docker compose --env-file "$HOME/.config/polyfin-provider-acceptance/sandbox.env" \
  -f testdata/acceptance/compose.yaml down
```

Keep the temporary volumes by default for diagnosis. Once checks are
completed and **only the disposable Compose project** is confirmed,
a separate, explicit command `down --volumes` can remove its disposable
volumes. Never use `docker system prune`, broad volume cleanup, or
commands against the family host.

The Compose fixture is intentionally **not** added to CI by default:
running a real mutating acceptance test must use a sandbox-only identity,
a reviewed synthetic HTTPS addon manifest and explicit access boundaries.
