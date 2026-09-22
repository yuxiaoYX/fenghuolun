package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecideApplyPrefersBundleWithoutDockerSock(t *testing.T) {
	can, mode, hint := decideApply(applyInput{
		UpdateAvailable: true,
		Latest:          "v0.2.0",
		BundleReady:     true,
		ReleaseBuild:    true,
		InDocker:        true,
		DockerReady:     false,
	})
	if !can || mode != "bundle" {
		t.Fatalf("can=%v mode=%s hint=%s", can, mode, hint)
	}
	if strings.Contains(hint, "套接字") {
		t.Fatal(hint)
	}
}

func TestDecideApplyFallsBackToDocker(t *testing.T) {
	can, mode, _ := decideApply(applyInput{
		UpdateAvailable: true,
		Latest:          "v0.2.0",
		Image:           "ghcr.io/yuxiaoyx/fenghuolun:v0.2.0",
		ReleaseBuild:    true,
		InDocker:        true,
		DockerReady:     true,
		SelfOK:          true,
	})
	if !can || mode != "docker" {
		t.Fatalf("can=%v mode=%s", can, mode)
	}
}

func TestDecideApplyRefusesHostProcess(t *testing.T) {
	can, mode, hint := decideApply(applyInput{
		UpdateAvailable: true,
		BundleReady:     true,
		InDocker:        false,
	})
	if can || mode != "" || !strings.Contains(hint, "不是容器") {
		t.Fatalf("can=%v mode=%s hint=%s", can, mode, hint)
	}
}

func TestDecideApplyMissingBundle(t *testing.T) {
	_, _, hint := decideApply(applyInput{
		UpdateAvailable: true,
		ReleaseBuild:    true,
		InDocker:        true,
	})
	if !strings.Contains(hint, "程序包") {
		t.Fatal(hint)
	}
}

func TestValidateDownloadURL(t *testing.T) {
	ok := []string{
		"https://github.com/yuxiaoYX/fenghuolun/releases/download/v0.1.0/a.tar.gz",
		"https://objects.githubusercontent.com/github-production-release-asset/x",
		"https://release-assets.githubusercontent.com/github-production-release-asset/x",
	}
	for _, u := range ok {
		if err := validateDownloadURL(u); err != nil {
			t.Fatal(u, err)
		}
	}
	bad := []string{
		"http://github.com/a",
		"https://evil.example/a",
		"https://user:pass@github.com/a",
	}
	for _, u := range bad {
		if err := validateDownloadURL(u); err == nil {
			t.Fatal("accepted", u)
		}
	}
}

func TestApplyBundleInstallsAndKeepsPrevious(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "app", "current")
	if err := os.MkdirAll(old, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "marker"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	body := mustTarGz(t, map[string]string{
		"fenghuolun":       "#!/bin/sh\necho new\n",
		"admin/index.html": "<html>admin</html>",
		"owner/index.html": "<html>owner</html>",
	})
	sum := sha256.Sum256(body)
	name := bundleAssetName("v0.2.0", "linux", "amd64")
	sums := hex.EncodeToString(sum[:]) + "  " + name + "\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch filepath.Base(r.URL.Path) {
		case name:
			_, _ = w.Write(body)
		case "SHA256SUMS":
			_, _ = w.Write([]byte(sums))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	prev := checkDownloadURL
	checkDownloadURL = func(string) error { return nil }
	t.Cleanup(func() { checkDownloadURL = prev })

	rel := Release{
		Tag: "v0.2.0",
		Assets: []ReleaseAsset{
			{Name: name, URL: srv.URL + "/" + name},
			{Name: "SHA256SUMS", URL: srv.URL + "/SHA256SUMS"},
		},
	}
	if err := applyBundle(context.Background(), dir, rel, "linux", "amd64"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "app", "current", "admin", "index.html"))
	if err != nil || !strings.Contains(string(got), "admin") {
		t.Fatalf("admin: %v %s", err, got)
	}
	if _, err := os.Stat(filepath.Join(dir, "app", "current", "fenghuolun")); err != nil {
		t.Fatal(err)
	}
	prevMarker, err := os.ReadFile(filepath.Join(dir, "app", "previous", "marker"))
	if err != nil || string(prevMarker) != "old" {
		t.Fatalf("previous: %v %s", err, prevMarker)
	}
}

func TestExtractBundleRejectsTraversal(t *testing.T) {
	body := mustTarGz(t, map[string]string{"../escape": "nope"})
	dir := t.TempDir()
	src := filepath.Join(dir, "b.tar.gz")
	if err := os.WriteFile(src, body, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := extractBundle(src, filepath.Join(dir, "out")); err == nil {
		t.Fatal("expected rejection")
	}
	if _, err := os.Stat(filepath.Join(dir, "escape")); !os.IsNotExist(err) {
		t.Fatal("escaped", err)
	}
}

func TestVerifySHA256Mismatch(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.tar.gz")
	if err := os.WriteFile(p, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	sums := strings.Repeat("ab", 32) + "  a.tar.gz\n"
	if err := verifySHA256File(p, sums, "a.tar.gz"); err == nil {
		t.Fatal("expected mismatch")
	}
}

func mustTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		b := []byte(content)
		hdr := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(b))}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(b); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
