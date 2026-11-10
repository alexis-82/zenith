//go:build !windows && !linux

package systheme

// Dark is unknown here before the page is loaded, so it reports light.
func Dark() bool { return false }
