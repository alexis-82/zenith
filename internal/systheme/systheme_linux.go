//go:build linux

package systheme

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Dark reads the same settings GTK uses, from which WebKitGTK derives
// prefers-color-scheme: GTK_THEME, then the GTK 3 settings.ini (also written
// by KDE), then the GNOME color scheme. When in doubt it reports light.
func Dark() bool {
	if theme := os.Getenv("GTK_THEME"); theme != "" {
		return isDarkThemeName(theme)
	}
	if dir, err := os.UserConfigDir(); err == nil {
		if f, err := os.Open(filepath.Join(dir, "gtk-3.0", "settings.ini")); err == nil {
			dark, found := gtkSettingsDark(f)
			f.Close()
			if found {
				return dark
			}
		}
	}
	return gnomePrefersDark()
}

// gtkSettingsDark reads a GTK settings.ini: gtk-application-prefer-dark-theme
// or a theme name with a dark variant. found is false when the file says
// nothing about the theme.
func gtkSettingsDark(r io.Reader) (dark, found bool) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		key, value, ok := strings.Cut(sc.Text(), "=")
		if !ok {
			continue
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		switch key {
		case "gtk-application-prefer-dark-theme":
			if value == "true" || value == "1" {
				return true, true
			}
			found = true
		case "gtk-theme-name":
			if isDarkThemeName(value) {
				return true, true
			}
			found = true
		}
	}
	return false, found
}

// isDarkThemeName recognises dark GTK themes: "Adwaita:dark", "Breeze-Dark"...
func isDarkThemeName(name string) bool {
	name = strings.ToLower(name)
	return strings.HasSuffix(name, ":dark") || strings.HasSuffix(name, "-dark")
}

// gnomePrefersDark asks gsettings for the GNOME color scheme. If gsettings is
// missing or slow it reports light: it must not delay the start.
func gnomePrefersDark() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	out, err := exec.CommandContext(ctx, "gsettings", "get", "org.gnome.desktop.interface", "color-scheme").Output()
	return err == nil && strings.TrimSpace(string(out)) == "'prefer-dark'"
}
