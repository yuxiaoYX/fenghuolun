package update

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLatestReleaseFromGitHub(t *testing.T) {
	cacheMu.Lock()
	cached = Release{}
	cachedUntil = time.Time{}
	cacheMu.Unlock()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("missing User-Agent")
		}
		switch r.URL.Path {
		case "/repos/yuxiaoYX/fenghuolun/releases/latest":
			http.NotFound(w, r)
		case "/repos/yuxiaoYX/fenghuolun/tags":
			_ = json.NewEncoder(w).Encode([]githubTag{{Name: "v0.1.0"}, {Name: "v0.1.1"}, {Name: "nightly"}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	prev := githubAPI
	githubAPI = srv.URL
	t.Cleanup(func() { githubAPI = prev })

	rel, err := LatestRelease(true)
	if err != nil {
		t.Fatal(err)
	}
	if rel.Tag != "v0.1.1" {
		t.Fatalf("tag %s", rel.Tag)
	}
}

func TestLatestReleasePrefersNewerTagOverStaleGitHubRelease(t *testing.T) {
	cacheMu.Lock()
	cached = Release{}
	cachedUntil = time.Time{}
	cacheMu.Unlock()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/yuxiaoYX/fenghuolun/releases/latest":
			_ = json.NewEncoder(w).Encode(githubRelease{TagName: "v0.1.0", HTMLURL: "https://example/v0.1.0"})
		case "/repos/yuxiaoYX/fenghuolun/tags":
			_ = json.NewEncoder(w).Encode([]githubTag{{Name: "v0.1.1"}, {Name: "v0.1.0"}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	prev := githubAPI
	githubAPI = srv.URL
	t.Cleanup(func() { githubAPI = prev })

	rel, err := LatestRelease(true)
	if err != nil {
		t.Fatal(err)
	}
	if rel.Tag != "v0.1.1" {
		t.Fatalf("want v0.1.1 from tags, got %s", rel.Tag)
	}
}

func TestImageRef(t *testing.T) {
	t.Setenv("FENGHUOLUN_UPDATE_IMAGE", "")
	if ImageRef("v0.1.1") != "ghcr.io/yuxiaoyx/fenghuolun:v0.1.1" {
		t.Fatal(ImageRef("v0.1.1"))
	}
}

func TestLatestReleaseKeepsAssets(t *testing.T) {
	cacheMu.Lock()
	cached = Release{}
	cachedUntil = time.Time{}
	cacheMu.Unlock()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/yuxiaoYX/fenghuolun/releases/latest":
			http.NotFound(w, r)
		case "/repos/yuxiaoYX/fenghuolun/tags":
			_ = json.NewEncoder(w).Encode([]githubTag{{Name: "v0.1.2"}})
		case "/repos/yuxiaoYX/fenghuolun/releases/tags/v0.1.2":
			_ = json.NewEncoder(w).Encode(githubRelease{
				TagName: "v0.1.2",
				HTMLURL: "https://example/v0.1.2",
				Assets: []githubAsset{{
					Name:               "fenghuolun_v0.1.2_linux_amd64.tar.gz",
					BrowserDownloadURL: "https://github.com/yuxiaoYX/fenghuolun/releases/download/v0.1.2/fenghuolun_v0.1.2_linux_amd64.tar.gz",
				}, {
					Name:               "SHA256SUMS",
					BrowserDownloadURL: "https://github.com/yuxiaoYX/fenghuolun/releases/download/v0.1.2/SHA256SUMS",
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	prev := githubAPI
	githubAPI = srv.URL
	t.Cleanup(func() { githubAPI = prev })

	rel, err := LatestRelease(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(rel.Assets) != 2 {
		t.Fatalf("assets %#v", rel.Assets)
	}
}
