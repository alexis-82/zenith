//go:build windows

package systheme

import "golang.org/x/sys/windows/registry"

// Dark reads the "app mode" chosen in the Windows colour settings.
func Dark() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	light, _, err := key.GetIntegerValue("AppsUseLightTheme")
	return err == nil && light == 0
}
