package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v1.0.1", "v1.0.0", true},
		{"v1.10.0", "v1.9.9", true},
		{"v2.0.0", "v1.99.99", true},
		{"v1.0.0", "v1.0.0", false},
		{"v1.0.0", "v1.0.1", false},
		{"v1.2.0-beta", "v1.1.0", true},
	}
	for _, c := range cases {
		got, err := IsNewer(c.latest, c.current)
		if err != nil || got != c.want {
			t.Errorf("IsNewer(%s, %s) = %v, %v; want %v", c.latest, c.current, got, err, c.want)
		}
	}
	for _, bad := range []string{"dev", "1.0.0", "v1.0", "v1.x.0", ""} {
		if _, err := IsNewer("v1.0.0", bad); err == nil {
			t.Errorf("expected error for current %q", bad)
		}
	}
}

func TestPickAsset(t *testing.T) {
	assets := []Asset{
		{Name: "Zenith-1.2.0-amd64-installer.exe"},
		{Name: "zenith-linux-amd64.tar.gz"},
		{Name: "zenith-linux-arm64.tar.gz"},
		{Name: "notes.txt"},
	}
	if a, err := PickAsset(assets, "windows", "amd64"); err != nil || a.Name != "Zenith-1.2.0-amd64-installer.exe" {
		t.Fatalf("windows: %v %v", a, err)
	}
	if a, err := PickAsset(assets, "linux", "arm64"); err != nil || a.Name != "zenith-linux-arm64.tar.gz" {
		t.Fatalf("linux: %v %v", a, err)
	}
	if _, err := PickAsset(assets, "darwin", "arm64"); !errors.Is(err, ErrNoAsset) {
		t.Fatalf("darwin: expected ErrNoAsset, got %v", err)
	}
	if _, err := PickAsset(assets[1:], "windows", "amd64"); !errors.Is(err, ErrNoAsset) {
		t.Fatalf("windows without setup: expected ErrNoAsset, got %v", err)
	}
}

func checker(srv *httptest.Server) *Checker {
	c := NewChecker("v1.0.0")
	c.Client = srv.Client()
	c.APIURL = srv.URL + "/api/latest"
	c.DownloadPrefix = srv.URL + "/dl/"
	return c
}

func TestLatest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "Zenith/v1.0.0" {
			t.Errorf("missing user agent: %q", r.Header.Get("User-Agent"))
		}
		fmt.Fprint(w, `{"tag_name":"v1.2.0","body":"## Novità\n- x","html_url":"https://github.com/r/1",
			"assets":[{"name":"Zenith-1.2.0-amd64-installer.exe","size":10,"browser_download_url":"https://h/a.exe","digest":"sha256:ABCD"}]}`)
	}))
	defer srv.Close()

	r, err := checker(srv).Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if r.Version != "v1.2.0" || !strings.Contains(r.Notes, "Novità") || len(r.Assets) != 1 || r.Assets[0].SHA256 != "abcd" {
		t.Fatalf("unexpected release %+v", r)
	}
}

func TestLatestErrors(t *testing.T) {
	status := http.StatusNotFound
	body := `{"message":"Not Found"}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}))
	defer srv.Close()
	c := checker(srv)

	if _, err := c.Latest(context.Background()); !errors.Is(err, ErrNoRelease) {
		t.Fatalf("404: expected ErrNoRelease, got %v", err)
	}
	status = http.StatusForbidden // e.g. rate limit
	if _, err := c.Latest(context.Background()); err == nil || errors.Is(err, ErrNoRelease) {
		t.Fatalf("403: expected generic error, got %v", err)
	}
	status, body = http.StatusOK, `{"tag_name":"release-1"}`
	if _, err := c.Latest(context.Background()); err == nil {
		t.Fatal("bad tag: expected error")
	}

	srv.Close() // offline
	if _, err := c.Latest(context.Background()); err == nil {
		t.Fatal("offline: expected error")
	}
}

func TestDownload(t *testing.T) {
	payload := []byte("setup-binary-content")
	sum := sha256.Sum256(payload)
	served := payload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(served)
	}))
	defer srv.Close()
	c := checker(srv)
	asset := Asset{Name: "Zenith-1.2.0-amd64-installer.exe", URL: srv.URL + "/dl/v1.2.0/Zenith-1.2.0-amd64-installer.exe", Size: int64(len(payload)), SHA256: hex.EncodeToString(sum[:])}

	var last int64
	path, err := c.Download(context.Background(), asset, func(done, total int64) { last = done })
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(filepath.Dir(path))
	got, _ := os.ReadFile(path)
	if string(got) != string(payload) || filepath.Base(path) != asset.Name || last != asset.Size {
		t.Fatalf("download mismatch: %q %s progress=%d", got, path, last)
	}

	// wrong checksum
	bad := asset
	bad.SHA256 = strings.Repeat("0", 64)
	if _, err := c.Download(context.Background(), bad, nil); err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("expected checksum error, got %v", err)
	}

	// size mismatch (truncated or longer body)
	for _, b := range [][]byte{payload[:5], append(append([]byte{}, payload...), 'x')} {
		served = b
		if _, err := c.Download(context.Background(), asset, nil); err == nil {
			t.Fatalf("expected size error for %d bytes", len(b))
		}
	}
	served = payload

	// URL outside the allowed prefix
	evil := asset
	evil.URL = srv.URL + "/other/Zenith-1.2.0-amd64-installer.exe"
	if _, err := c.Download(context.Background(), evil, nil); err == nil || !strings.Contains(err.Error(), "non consentito") {
		t.Fatalf("expected prefix error, got %v", err)
	}

	// path traversal in the asset name
	trav := asset
	trav.Name = "../evil.exe"
	if _, err := c.Download(context.Background(), trav, nil); err == nil {
		t.Fatal("expected invalid name error")
	}
}

func TestDefaultPrefixMatchesRepo(t *testing.T) {
	if !strings.HasPrefix("https://github.com/alexis-82/zenith/releases/download/v1.0.0/Zenith-1.0.0-amd64-installer.exe", DefaultDownloadPrefix) {
		t.Fatal("default prefix does not match the repo download URLs")
	}
}

// Opt-in check against the real GitHub API: ZENITH_LIVE_GITHUB=1 go test ./internal/update -run Live
func TestLiveGitHub(t *testing.T) {
	if os.Getenv("ZENITH_LIVE_GITHUB") != "1" {
		t.Skip("set ZENITH_LIVE_GITHUB=1 to query api.github.com")
	}
	r, err := NewChecker("v0.0.0").Latest(context.Background())
	if errors.Is(err, ErrNoRelease) {
		t.Log("no release published yet")
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("latest %s, assets: %d", r.Version, len(r.Assets))
	for _, a := range r.Assets {
		if !strings.HasPrefix(a.URL, DefaultDownloadPrefix) {
			t.Errorf("asset URL outside the allowed prefix: %s", a.URL)
		}
	}
}
