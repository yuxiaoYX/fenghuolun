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
			_ = json.NewEncoder(w).Encode(githubLatest{TagName: "v0.1.0", HTMLURL: "https://example/v0.1.0"})
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
