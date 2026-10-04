#!/usr/bin/env bash
# Installs the mdt binary on macOS without needing Homebrew or Go.
#
#   curl -fsSL https://raw.githubusercontent.com/JosePBrotons/mobile-dev-tools/master/install.sh | bash
#
# Environment:
#   MDT_VERSION      release to install, for example v0.1.0 (default: latest)
#   MDT_INSTALL_DIR  where to put the binary (default: $HOME/.local/bin)
#   MDT_BASE_URL     release download URL, to test against a local folder
set -euo pipefail

BASE_URL="${MDT_BASE_URL:-https://github.com/JosePBrotons/mobile-dev-tools/releases/download}"
INSTALL_DIR="${MDT_INSTALL_DIR:-$HOME/.local/bin}"

die() {
    echo "install.sh: $*" >&2
    exit 1
}

[[ "$(uname -s)" = "Darwin" ]] || die "mdt installs tools on macOS only"

version="${MDT_VERSION:-}"
if [[ -z "$version" ]]; then
    # The latest release URL redirects to .../releases/tag/<version>.
    latest="$(curl -fsSLI -o /dev/null -w '%{url_effective}' \
        https://github.com/JosePBrotons/mobile-dev-tools/releases/latest)" ||
        die "could not look up the latest release"
    version="${latest##*/}"
fi
[[ "$version" = v* ]] || die "unexpected version '$version'"

archive="mdt_${version#v}_darwin_all.tar.gz"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "Downloading mdt $version..."
curl -fsSL -o "$tmp/$archive" "$BASE_URL/$version/$archive" || die "could not download $archive"
curl -fsSL -o "$tmp/checksums.txt" "$BASE_URL/$version/checksums.txt" || die "could not download checksums.txt"

echo "Verifying checksum..."
(cd "$tmp" && grep " $archive\$" checksums.txt | shasum -a 256 -c - >/dev/null) ||
    die "checksum mismatch for $archive"

tar -xzf "$tmp/$archive" -C "$tmp" mdt
mkdir -p "$INSTALL_DIR"
install -m 0755 "$tmp/mdt" "$INSTALL_DIR/mdt"

echo "Installed mdt $version to $INSTALL_DIR/mdt"
case ":$PATH:" in
*":$INSTALL_DIR:"*) ;;
*) echo "Add it to your PATH: export PATH=\"$INSTALL_DIR:\$PATH\"" ;;
esac
echo "Run 'mdt' to start."
