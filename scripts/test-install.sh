#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

fake_bin="$work/bin"
mkdir -p "$fake_bin" "$work/payload" "$work/project"

cat >"$work/payload/harn_linux_amd64" <<'EOF'
#!/bin/sh
echo "fake harn $*"
EOF
chmod +x "$work/payload/harn_linux_amd64"

(
  cd "$work/payload"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum harn_linux_amd64 >SHA256SUMS
  else
    shasum -a 256 harn_linux_amd64 >SHA256SUMS
  fi
)

cat >"$fake_bin/uname" <<'EOF'
#!/bin/sh
case "$1" in
  -s) echo Linux ;;
  -m) echo x86_64 ;;
  *) echo Linux ;;
esac
EOF
chmod +x "$fake_bin/uname"

cat >"$fake_bin/curl" <<EOF
#!/bin/sh
set -e
destination=""
while [ "\$#" -gt 0 ]; do
  case "\$1" in
    -o) destination="\$2"; shift 2 ;;
    *) shift ;;
  esac
done
case "\$destination" in
  *SHA256SUMS) cp "$work/payload/SHA256SUMS" "\$destination" ;;
  *) cp "$work/payload/harn_linux_amd64" "\$destination" ;;
esac
EOF
chmod +x "$fake_bin/curl"

PATH="$fake_bin:$PATH" sh "$root/install.sh" --version v0.2.0 --root "$work/project" >"$work/output" 2>&1

test -x "$work/project/.harness/bin/harn"
test "$("$work/project/.harness/bin/harn" version)" = "fake harn version"
grep -q "installed $work/project/.harness/bin/harn" "$work/output"
grep -q -- "--version tag:v0.2.0" "$work/output"

if PATH="$fake_bin:$PATH" sh "$root/install.sh" --bin-dir "$work/custom" >/dev/null 2>&1; then
  test -x "$work/custom/harn"
else
  echo "install without an explicit version should still succeed" >&2
  exit 1
fi

# A tampered asset must fail before anything is installed.
cp "$work/payload/harn_linux_amd64" "$work/payload/original"
printf '#!/bin/sh\necho tampered\n' >"$work/payload/harn_linux_amd64"
if PATH="$fake_bin:$PATH" sh "$root/install.sh" --bin-dir "$work/tampered" >"$work/tamper-output" 2>&1; then
  echo "install must reject a checksum mismatch" >&2
  exit 1
fi
grep -q "SHA256 mismatch" "$work/tamper-output"
test ! -e "$work/tampered/harn"

echo "install.sh tests passed"
