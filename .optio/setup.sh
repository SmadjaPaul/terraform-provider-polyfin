#!/usr/bin/env bash
# Toolchain preparation of the Optio agent pod.
#
# Optio runs this script once per pod, at initialisation, from the repository
# checkout (hook implemented by /opt/optio/repo-init.sh, which looks for
# .optio/setup.sh). The pod image ships Node.js only: without this script the
# agent has no Go and cannot run the provider test suite.
#
# Contract:
#   * version    : the `go` directive of go.mod, cross-checked with mise.lock;
#   * source     : the official dl.google.com archive named in mise.lock;
#   * checksum   : the sha256 recorded in mise.lock, verified before extraction;
#   * install    : $HOME/.local/optio/go, with a `go` shim in $HOME/.local/bin;
#   * rootless   : no privileged command, nothing written outside $HOME;
#   * idempotent : re-running an up-to-date installation downloads nothing.
#
# The exit status is always 0 in pod mode: /opt/optio/repo-init.sh runs with
# `set -e`, so a failure here would stop the pod from starting. Set
# OPTIO_SETUP_STRICT=1 to propagate the failures instead (tests, manual runs).
set -uo pipefail

STRICT="${OPTIO_SETUP_STRICT:-0}"
HOME="${HOME:-}"
if [ -z "${HOME}" ]; then
  printf '[optio/setup] ERROR: HOME is not set, the toolchain cannot be installed\n' >&2
  [ "${STRICT}" = "1" ] && exit 1
  exit 0
fi

REPO_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
GO_MOD="${REPO_ROOT}/go.mod"
MISE_LOCK="${REPO_ROOT}/mise.lock"
INSTALL_ROOT="${OPTIO_GO_INSTALL_ROOT:-${HOME}/.local/optio/go}"
ENV_FILE="${OPTIO_GO_ENV_FILE:-${HOME}/.local/optio/go-env.sh}"
BIN_DIR="${OPTIO_GO_BIN_DIR:-${HOME}/.local/bin}"
DOWNLOAD_BASE="${OPTIO_GO_DOWNLOAD_BASE_URL:-https://dl.google.com/go}"
OFFICIAL_BASE="https://dl.google.com/go"
STATUS=0

log() { printf '[optio/setup] %s\n' "$*"; }
warn() { printf '[optio/setup] WARNING: %s\n' "$*" >&2; }
fail() { printf '[optio/setup] ERROR: %s\n' "$*" >&2; STATUS=1; }

# lock_value prints the value of a key of a mise.lock section.
lock_value() {
  local section="$1" key="$2"
  awk -v section="$section" -v key="$key" '
    /^[[:space:]]*\[/ { inside = ($0 == section) }
    inside && $1 == key {
      value = $3
      gsub(/^"|"$/, "", value)
      print value
      exit
    }
  ' "${MISE_LOCK}"
}

# go_mod_version prints the Go version required by go.mod.
go_mod_version() {
  awk '$1 == "go" { print $2; exit }' "${GO_MOD}"
}

# detect_platform prints the mise platform key of the running machine.
detect_platform() {
  local os arch
  os="$(uname -s)"
  arch="$(uname -m)"
  case "${os}" in
    Linux)
      case "${arch}" in
        x86_64 | amd64) echo "linux-x64" ;;
        aarch64 | arm64) echo "linux-arm64" ;;
      esac
      ;;
    Darwin)
      case "${arch}" in
        arm64 | aarch64) echo "macos-arm64" ;;
        x86_64) echo "macos-x64" ;;
      esac
      ;;
  esac
}

# sha256_of prints the sha256 of a file, empty when no tool is available.
sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{ print $1 }'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{ print $1 }'
  fi
}

# download fetches an archive over HTTP(S) or from a local mirror.
download() {
  local url="$1" dest="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL --retry 3 --retry-delay 2 -o "${dest}" "${url}"
  elif command -v wget >/dev/null 2>&1; then
    wget -q -O "${dest}" "${url}"
  else
    return 1
  fi
}

# install_shim exposes the toolchain through a directory already on PATH.
install_shim() {
  if ! mkdir -p "${BIN_DIR}"; then
    warn "cannot create ${BIN_DIR}"
    return
  fi
  if ! ln -sf "${INSTALL_ROOT}/bin/go" "${BIN_DIR}/go"; then
    warn "cannot link ${BIN_DIR}/go"
    return
  fi
  if [ -x "${INSTALL_ROOT}/bin/gofmt" ]; then
    ln -sf "${INSTALL_ROOT}/bin/gofmt" "${BIN_DIR}/gofmt" 2>/dev/null || true
  fi
  case ":${PATH}:" in
    *":${BIN_DIR}:"*) ;;
    *) warn "${BIN_DIR} is not on PATH, add it to the shell profile to reach go" ;;
  esac
}

# write_profile documents the installation for the interactive shells.
write_profile() {
  mkdir -p "$(dirname "${ENV_FILE}")" 2>/dev/null || return
  {
    printf '# Go toolchain installed by .optio/setup.sh (Optio agent pod).\n'
    printf 'export PATH="%s/bin:$PATH"\n' "${INSTALL_ROOT}"
    printf 'export GOTOOLCHAIN=local\n'
  } >"${ENV_FILE}"
  local profile
  for profile in "${HOME}/.bashrc" "${HOME}/.profile"; do
    touch "${profile}" 2>/dev/null || continue
    sed -i.bak -e '/\.local\/optio\/go-env\.sh/d' -e '/\.local\/optio\/go\/env\.sh/d' "${profile}" 2>/dev/null &&
      rm -f "${profile}.bak"
    printf '\n. "%s"\n' "${ENV_FILE}" >>"${profile}"
  done
}

# toolchain_ready tells whether the requested version is already installed.
toolchain_ready() {
  local version="$1"
  [ -x "${INSTALL_ROOT}/bin/go" ] || return 1
  "${INSTALL_ROOT}/bin/go" version 2>/dev/null | grep -qF "go${version}"
}

main() {
  if [ ! -f "${GO_MOD}" ]; then
    fail "go.mod not found in ${REPO_ROOT}"
    return
  fi
  if [ ! -f "${MISE_LOCK}" ]; then
    fail "mise.lock not found in ${REPO_ROOT}"
    return
  fi

  local version lock_version platform archive url expected actual tmp
  version="$(go_mod_version)"
  if [ -z "${version}" ]; then
    fail "no go directive in go.mod"
    return
  fi
  lock_version="$(lock_value '[[tools.go]]' version)"
  if [ "${lock_version}" != "${version}" ]; then
    fail "mise.lock pins go ${lock_version} but go.mod requires ${version}: refresh the lock file with 'mise lock'"
    return
  fi

  platform="$(detect_platform)"
  if [ -z "${platform}" ]; then
    fail "unsupported platform $(uname -s)/$(uname -m)"
    return
  fi

  url="$(lock_value "[tools.go.\"platforms.${platform}\"]" url)"
  expected="$(lock_value "[tools.go.\"platforms.${platform}\"]" checksum)"
  if [ -z "${url}" ] || [ -z "${expected}" ]; then
    fail "mise.lock has no archive or checksum for ${platform}"
    return
  fi
  case "${url}" in
    "${OFFICIAL_BASE}"/*) ;;
    *)
      fail "mise.lock archive ${url} does not come from the official Go download"
      return
      ;;
  esac
  expected="${expected#sha256:}"
  archive="${url##*/}"

  case "${INSTALL_ROOT}" in
    "${HOME}"/*) ;;
    *)
      fail "refusing to install the toolchain outside of ${HOME}: ${INSTALL_ROOT}"
      return
      ;;
  esac

  if toolchain_ready "${version}"; then
    log "go ${version} is already installed in ${INSTALL_ROOT}"
    install_shim
    write_profile
    return
  fi

  if ! tmp="$(mktemp -d "${TMPDIR:-/tmp}/optio-go-XXXXXX")"; then
    fail "cannot create a temporary directory"
    return
  fi

  log "downloading go ${version} (${platform}) from ${DOWNLOAD_BASE}/${archive}"
  if ! download "${DOWNLOAD_BASE}/${archive}" "${tmp}/${archive}"; then
    fail "download of ${DOWNLOAD_BASE}/${archive} failed"
    rm -rf "${tmp}"
    return
  fi

  actual="$(sha256_of "${tmp}/${archive}")"
  if [ -z "${actual}" ]; then
    fail "no sha256 tool available to verify ${archive}"
    rm -rf "${tmp}"
    return
  fi
  if [ "${actual}" != "${expected}" ]; then
    fail "checksum mismatch for ${archive}: expected sha256:${expected}, got sha256:${actual}"
    rm -rf "${tmp}"
    return
  fi
  log "checksum verified: sha256:${actual}"

  if ! tar -C "${tmp}" -xzf "${tmp}/${archive}"; then
    fail "cannot extract ${archive}"
    rm -rf "${tmp}"
    return
  fi
  if [ ! -x "${tmp}/go/bin/go" ]; then
    fail "${archive} does not contain the go toolchain"
    rm -rf "${tmp}"
    return
  fi

  case "${INSTALL_ROOT}" in
    "${HOME}"/*) ;;
    *)
      fail "refusing to install the toolchain outside of ${HOME}: ${INSTALL_ROOT}"
      rm -rf "${tmp}"
      return
      ;;
  esac
  mkdir -p "$(dirname "${INSTALL_ROOT}")" || {
    fail "cannot create $(dirname "${INSTALL_ROOT}")"
    rm -rf "${tmp}"
    return
  }
  rm -rf "${INSTALL_ROOT}"
  if ! mv "${tmp}/go" "${INSTALL_ROOT}"; then
    fail "cannot install the toolchain in ${INSTALL_ROOT}"
    rm -rf "${tmp}"
    return
  fi
  rm -rf "${tmp}"

  install_shim
  write_profile

  # The module cache is warmed up so the agent can run the test suite even
  # without network, and the toolchain policy matches the mise contract.
  (cd "${REPO_ROOT}" && "${INSTALL_ROOT}/bin/go" env -w GOTOOLCHAIN=local) ||
    warn "cannot pin GOTOOLCHAIN=local in the go environment file"
  (cd "${REPO_ROOT}" && "${INSTALL_ROOT}/bin/go" mod download) ||
    warn "cannot pre-download the go modules, they will be fetched on demand"

  log "go is ready: $("${INSTALL_ROOT}/bin/go" version)"
}

main

if [ "${STATUS}" -ne 0 ]; then
  if [ "${STRICT}" = "1" ]; then
    exit "${STATUS}"
  fi
  log "setup finished with errors, the pod keeps starting without the toolchain"
fi
exit 0
