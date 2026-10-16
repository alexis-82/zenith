#!/bin/sh
# Creates build/bin/zenith-linux-<arch>.tar.gz, the Linux file to attach to a
# GitHub release. Run from the project root after the Linux build, e.g.
#   wails build -tags webkit2_41 -ldflags "-X main.version=v1.2.3"
#   sh build/linux/package.sh
# The archive keeps the project layout expected by install.sh.
set -e

if [ ! -f build/bin/zenith ]; then
    echo "build/bin/zenith non trovato: esegui prima 'wails build' dalla root del progetto." >&2
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

OUT="build/bin/zenith-linux-$ARCH.tar.gz"
tar -czf "$OUT" \
    build/bin/zenith \
    build/linux/install.sh \
    build/linux/zenith-launch.sh \
    build/linux/zenith-update.sh \
    build/linux/zenith.desktop \
    build/appicon.png
echo "Creato $OUT: allegalo alla release su GitHub."
