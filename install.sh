#!/usr/bin/env sh
set -eu

# Install the Harn binary for the current platform.
#
#   curl -fsSL https://raw.githubusercontent.com/AwayerCode/Harn/main/install.sh | sh
#   curl -fsSL .../install.sh | sh -s -- --version v0.2.1 --root ./my-project

repo="AwayerCode/Harn"
version="${HARN_VERSION:-}"
root="."
bin_dir=""

usage() {
  cat <<'EOF'
Usage: install.sh [--version <tag>] [--root <dir>] [--bin-dir <dir>]

  --version <tag>  Install a specific release tag (default: latest release)
  --root <dir>     Project root; the binary goes to <dir>/.harness/bin
  --bin-dir <dir>  Override the binary directory
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      version="${2:-}"
      shift 2
      ;;
    --root)
      root="${2:-}"
      shift 2
      ;;
    --bin-dir)
      bin_dir="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      printf 'install.sh: unknown argument %s\n' "$1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

case "$(uname -s)" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *)
    printf 'install.sh: unsupported operating system %s\n' "$(uname -s)" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  arm64|aarch64) arch="arm64" ;;
  x86_64|amd64) arch="amd64" ;;
  *)
    printf 'install.sh: unsupported architecture %s\n' "$(uname -m)" >&2
    exit 1
    ;;
esac

asset="harn_${os}_${arch}"
if [ -n "$version" ]; then
  base="https://github.com/${repo}/releases/download/${version}"
else
  base="https://github.com/${repo}/releases/latest/download"
fi

if [ -z "$bin_dir" ]; then
  bin_dir="${root}/.harness/bin"
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

printf 'downloading %s\n' "$asset"
curl -fsSL "$base/$asset" -o "$tmp/$asset"
curl -fsSL "$base/SHA256SUMS" -o "$tmp/SHA256SUMS"

expected="$(grep "  $asset\$" "$tmp/SHA256SUMS" | cut -d' ' -f1)"
if [ -z "$expected" ]; then
  printf 'install.sh: %s is not listed in SHA256SUMS\n' "$asset" >&2
  exit 1
fi
if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$tmp/$asset" | cut -d' ' -f1)"
else
  actual="$(shasum -a 256 "$tmp/$asset" | cut -d' ' -f1)"
fi
if [ "$actual" != "$expected" ]; then
  printf 'install.sh: SHA256 mismatch for %s\n' "$asset" >&2
  exit 1
fi

mkdir -p "$bin_dir"
cp "$tmp/$asset" "$bin_dir/harn"
chmod +x "$bin_dir/harn"

printf 'installed %s\n' "$bin_dir/harn"
printf 'available skills:\n'
if [ -n "$version" ]; then
  if ! "$bin_dir/harn" search --source "https://github.com/$repo" --version "tag:$version"; then
    printf 'install.sh: installed successfully, but could not list skills; retry harn search later\n' >&2
  fi
else
  if ! "$bin_dir/harn" search --source "https://github.com/$repo"; then
    printf 'install.sh: installed successfully, but could not list skills; retry harn search later\n' >&2
  fi
fi
printf 'usage: https://github.com/%s/blob/main/HARN.md\n' "$repo"
