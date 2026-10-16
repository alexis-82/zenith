#!/bin/sh
# Starts Zenith from the menu entry / desktop link.
# WebKitGTK's DMA-BUF renderer shows a blank window with some GPU drivers
# (NVIDIA, some Wayland setups): disable it unless the user set it already.
export WEBKIT_DISABLE_DMABUF_RENDERER="${WEBKIT_DISABLE_DMABUF_RENDERER:-1}"
# The binary sits next to this launcher: /usr/local/bin (install.sh) or
# /usr/bin (.deb package).
exec "$(dirname "$(readlink -f "$0")")/zenith" "$@"
