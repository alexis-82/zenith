package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"

	ghupdate "zenith/internal/update"
)

// fakeGitHub serves a "latest release" with one asset for the current OS.
type fakeGitHub struct {
	srv     *httptest.Server
	tag     string
	payload []byte
	status  int
}

func newFakeGitHub(t *testing.T, tag string) *fakeGitHub {
	f := &fakeGitHub{tag: tag, payload: []byte("installer"), status: http.StatusOK}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/dl/") {
			w.Write(f.payload)
			return
		}
		if f.status != http.StatusOK {
			w.WriteHeader(f.status)
			return
		}
		name := "Zenith-1.0.0-setup.exe"
		if goruntime.GOOS == "linux" {
			name = "zenith-linux-" + goruntime.GOARCH + ".tar.gz"
		}
		sum := sha256.Sum256(f.payload)
		fmt.Fprintf(w, `{"tag_name":%q,"body":"Note","html_url":"x","assets":[{"name":%q,"size":%d,"browser_download_url":%q,"digest":"sha256:%s"}]}`,
			f.tag, name, len(f.payload), f.srv.URL+"/dl/"+name, hex.EncodeToString(sum[:]))
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func withVersion(t *testing.T, v string) {
	old := version
	version = v
	t.Cleanup(func() { version = old })
}

// startUpdatableApp starts an app whose updater talks to the fake server and
// records installer launches and quits instead of performing them.
func startUpdatableApp(t *testing.T, gh *fakeGitHub) (*testApp, *[]string, *int) {
	a := startApp(t, t.TempDir())
	c := ghupdate.NewChecker(version)
	c.Client = gh.srv.Client()
	c.APIURL = gh.srv.URL + "/api"
	c.DownloadPrefix = gh.srv.URL + "/dl/"
	installed := []string{}
	quits := 0
	a.updates.checker = c
	a.updates.install = func(p string) error {
		data, _ := os.ReadFile(p)
		installed = append(installed, string(data))
		os.RemoveAll(filepath.Dir(p))
		return nil
	}
	a.updates.quit = func(context.Context) { quits++ }
	return a, &installed, &quits
}

func TestUpdate_CheckStates(t *testing.T) {
	if goruntime.GOOS != "windows" && goruntime.GOOS != "linux" {
		t.Skip("no installable asset on this OS")
	}
	withVersion(t, "v1.0.0")
	gh := newFakeGitHub(t, "v1.1.0")
	a, _, _ := startUpdatableApp(t, gh)

	info, err := a.CheckUpdate()
	if err != nil || !info.Available || !info.CanInstall || info.Latest != "v1.1.0" || info.Current != "v1.0.0" || info.Notes != "Note" {
		t.Fatalf("newer release: %+v %v", info, err)
	}

	gh.tag = "v1.0.0"
	if info, _ := a.CheckUpdate(); info.Available || info.CanInstall {
		t.Fatalf("same version must not be offered: %+v", info)
	}

	withVersion(t, "dev")
	gh.tag = "v9.9.9"
	if info, _ := a.CheckUpdate(); info.Available {
		t.Fatalf("dev build must not be offered updates: %+v", info)
	}

	gh.status = http.StatusNotFound
	if info, err := a.CheckUpdate(); err != nil || !info.NoRelease {
		t.Fatalf("no release yet: %+v %v", info, err)
	}

	gh.srv.Close()
	if _, err := a.CheckUpdate(); err == nil {
		t.Fatal("offline: expected an error")
	}
}

func TestUpdate_InstallSavesDataThenLaunchesAndQuits(t *testing.T) {
	if goruntime.GOOS != "windows" && goruntime.GOOS != "linux" {
		t.Skip("no installable asset on this OS")
	}
	withVersion(t, "v1.0.0")
	gh := newFakeGitHub(t, "v1.1.0")
	a, installed, quits := startUpdatableApp(t, gh)

	if err := a.InstallUpdate(); err == nil {
		t.Fatal("install before a check must fail")
	}

	p, b := newDataFile(t, a)
	s := a.GetSettings()
	s.AutosaveSeconds = 300
	a.SaveSettings(s)
	update(t, a, withProject(b, "Non ancora salvato"))

	a.CheckUpdate()
	if err := a.InstallUpdate(); err != nil {
		t.Fatal(err)
	}
	if len(*installed) != 1 || (*installed)[0] != "installer" || *quits != 1 {
		t.Fatalf("installer not launched / app not quit: %v quits=%d", *installed, *quits)
	}
	if readBoard(t, p).Projects[0].Name != "Non ancora salvato" {
		t.Fatal("pending changes not saved before updating")
	}
	if len(a.eventsNamed("update:progress")) == 0 {
		t.Fatal("no download progress events")
	}
}

func TestUpdate_InstallRefusedWhenDataCannotBeSaved(t *testing.T) {
	if goruntime.GOOS != "windows" && goruntime.GOOS != "linux" {
		t.Skip("no installable asset on this OS")
	}
	withVersion(t, "v1.0.0")
	gh := newFakeGitHub(t, "v1.1.0")
	a, installed, quits := startUpdatableApp(t, gh)
	p, b := newDataFile(t, a)
	s := a.GetSettings()
	s.AutosaveSeconds = 300
	a.SaveSettings(s)
	update(t, a, withProject(b, "Da salvare"))
	os.RemoveAll(filepath.Dir(p)) // save will fail

	a.CheckUpdate()
	if err := a.InstallUpdate(); err == nil || !strings.Contains(err.Error(), "salvare") {
		t.Fatalf("expected save error, got %v", err)
	}
	if len(*installed) != 0 || *quits != 0 {
		t.Fatal("installer launched although data could not be saved")
	}
}

func TestUpdate_CorruptDownloadNotInstalled(t *testing.T) {
	if goruntime.GOOS != "windows" && goruntime.GOOS != "linux" {
		t.Skip("no installable asset on this OS")
	}
	withVersion(t, "v1.0.0")
	gh := newFakeGitHub(t, "v1.1.0")
	a, installed, _ := startUpdatableApp(t, gh)
	a.CheckUpdate() // digest computed on "installer"
	gh.payload = []byte("tampered!")
	if err := a.InstallUpdate(); err == nil {
		t.Fatal("expected verification error")
	}
	if len(*installed) != 0 {
		t.Fatal("corrupt installer launched")
	}
}

func TestUpdate_ConcurrentInstallRejected(t *testing.T) {
	if goruntime.GOOS != "windows" && goruntime.GOOS != "linux" {
		t.Skip("no installable asset on this OS")
	}
	withVersion(t, "v1.0.0")
	gh := newFakeGitHub(t, "v1.1.0")
	a, _, _ := startUpdatableApp(t, gh)
	a.CheckUpdate()

	// The first install blocks inside the installer launcher.
	release := make(chan struct{})
	entered := make(chan struct{})
	a.updates.install = func(string) error {
		close(entered)
		<-release
		return nil
	}
	done := make(chan error)
	go func() { done <- a.InstallUpdate() }()
	<-entered
	if err := a.InstallUpdate(); err == nil || !strings.Contains(err.Error(), "in corso") {
		t.Fatalf("second install while busy: expected rejection, got %v", err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
