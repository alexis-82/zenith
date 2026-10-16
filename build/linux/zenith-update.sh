#!/bin/sh
# Installs a Zenith release for Linux.
#
#   sudo zenith-update.sh                              download the latest GitHub release and install it
#   sudo zenith-update.sh zenith-linux-amd64.tar.gz    install a release file already downloaded
#
# Zenith runs it through pkexec with the file it has already downloaded and
# verified (size and SHA-256).
set -e

REPO=alexis-82/zenith

if [ "$(id -u)" -ne 0 ]; then
    echo "Esegui come root: sudo $0 $*" >&2
    exit 1
fi

case "$(uname -m)" in
    x86_64) ARCH=amd64 ;;
    aarch64 | arm64) ARCH=arm64 ;;
    *)
        echo "Architettura non supportata: $(uname -m)" >&2
        exit 1
        ;;
esac
ASSET="zenith-linux-$ARCH.tar.gz"

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT INT TERM

if [ -n "$1" ]; then
    TARBALL=$1
    if [ ! -f "$TARBALL" ]; then
        echo "File non trovato: $TARBALL" >&2
        exit 1
    fi
else
    # GitHub redirects this URL to the asset of the latest release.
    URL="https://github.com/$REPO/releases/latest/download/$ASSET"
    echo "Download di $URL"
    if command -v curl >/dev/null 2>&1; then
        curl -fL --proto '=https' --tlsv1.2 -o "$WORK/$ASSET" "$URL"
    elif command -v wget >/dev/null 2>&1; then
        wget -q --https-only -O "$WORK/$ASSET" "$URL"
    else
        echo "Serve curl oppure wget." >&2
        exit 1
    fi
    TARBALL="$WORK/$ASSET"
fi

mkdir "$WORK/pkg"
tar -xzf "$TARBALL" -C "$WORK/pkg" --no-same-owner
if [ ! -f "$WORK/pkg/build/bin/zenith" ] || [ ! -f "$WORK/pkg/build/linux/install.sh" ]; then
    echo "Pacchetto non valido: mancano build/bin/zenith o build/linux/install.sh" >&2
    exit 1
fi

cd "$WORK/pkg"
sh build/linux/install.sh
