#!/bin/sh
# Installs Zenith system-wide: binary, launcher, menu entry, icon and a link
# on the desktop of the user who ran sudo. Run from the project root after
# `wails build` (on recent distributions `wails build -tags webkit2_41`).
#
#   sudo sh build/linux/install.sh               install / update
#   sudo sh build/linux/install.sh --uninstall   remove
set -e

BIN=/usr/local/bin/zenith
LAUNCHER=/usr/local/bin/zenith-launch.sh
DESKTOP_FILE=/usr/share/applications/zenith.desktop
ICON=/usr/share/pixmaps/zenith.png

if [ "$(id -u)" -ne 0 ]; then
    echo "Esegui come root: sudo sh $0 $*" >&2
    exit 1
fi

# Desktop folder of the user who invoked sudo (empty if unknown).
user_desktop_dir() {
    [ -n "$SUDO_USER" ] && [ "$SUDO_USER" != root ] || return 0
    dir=$(sudo -u "$SUDO_USER" xdg-user-dir DESKTOP 2>/dev/null || true)
    if [ -z "$dir" ] || [ "$dir" = "$(getent passwd "$SUDO_USER" | cut -d: -f6)" ]; then
        dir="$(getent passwd "$SUDO_USER" | cut -d: -f6)/Desktop"
    fi
    [ -d "$dir" ] && echo "$dir"
    return 0
}

refresh_menu() {
    if command -v update-desktop-database >/dev/null 2>&1; then
        update-desktop-database -q /usr/share/applications || true
    fi
}

DESKTOP_DIR=$(user_desktop_dir)

if [ "$1" = "--uninstall" ]; then
    rm -f "$BIN" "$LAUNCHER" "$DESKTOP_FILE" "$ICON"
    [ -n "$DESKTOP_DIR" ] && rm -f "$DESKTOP_DIR/zenith.desktop"
    refresh_menu
    echo "Zenith rimosso. Il file dati JSON e le impostazioni (~/.config/Zenith) non sono stati toccati."
    exit 0
fi

if [ ! -f build/bin/zenith ]; then
    echo "build/bin/zenith non trovato: esegui prima 'wails build' dalla root del progetto." >&2
    exit 1
fi

install -m 0755 build/bin/zenith "$BIN"
install -m 0755 build/linux/zenith-launch.sh "$LAUNCHER"
install -m 0644 build/linux/zenith.desktop "$DESKTOP_FILE"
install -m 0644 build/appicon.png "$ICON"
refresh_menu

# Link on the user's desktop, owned by the user and marked as trusted
# (GNOME/Nautilus does not launch untrusted .desktop files).
if [ -n "$DESKTOP_DIR" ]; then
    install -m 0755 -o "$SUDO_USER" -g "$(id -gn "$SUDO_USER")" build/linux/zenith.desktop "$DESKTOP_DIR/zenith.desktop"
    sudo -u "$SUDO_USER" DBUS_SESSION_BUS_ADDRESS="unix:path=/run/user/$(id -u "$SUDO_USER")/bus" \
        gio set "$DESKTOP_DIR/zenith.desktop" metadata::trusted true 2>/dev/null || true
    echo "Collegamento creato in $DESKTOP_DIR"
fi

echo "Zenith installato: lo trovi nel menu delle applicazioni."
echo "Dipendenza runtime: libwebkit2gtk-4.1-0 (oppure libwebkit2gtk-4.0-37 se compilato senza -tags webkit2_41)."
