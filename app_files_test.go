package main

import "testing"

func TestIsSafeURL(t *testing.T) {
	cases := map[string]bool{
		"https://example.com/a?b=c":            true,
		"http://example.com":                   true,
		"mailto:me@example.com":                true,
		"HTTPS://EXAMPLE.COM":                  true,
		"file:///C:/Windows/System32/calc.exe": false,
		"javascript:alert(1)":                  false,
		"ms-settings:":                         false,
		"steam://run/1":                        false,
		"relative/path":                        false,
		"https:///nohost":                      false,
	}
	for u, want := range cases {
		if got := isSafeURL(u); got != want {
			t.Errorf("isSafeURL(%q) = %v, want %v", u, got, want)
		}
	}
}
