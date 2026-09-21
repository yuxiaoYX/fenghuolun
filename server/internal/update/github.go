package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const defaultRepo = "yuxiaoYX/fenghuolun"

var githubAPI = "https://api.github.com"

type Release struct {
	Tag     string
	URL     string
	Notes   string
	Fetched time.Time
}

type githubLatest struct {
	TagName    string `json:"tag_name"`
	HTMLURL    string `json:"html_url"`
	Body       string `json:"body"`
	Prerelease bool   `json:"prerelease"`
	Draft      bool   `json:"draft"`
}

type githubTag struct {
	Name string `json:"name"`
}

var (
	httpClient = &http.Client{Timeout: 20 * time.Second}

	cacheMu     sync.Mutex
	cached      Release
	cachedUntil time.Time
)

func repoName() string {
	if v := strings.TrimSpace(os.Getenv("FENGHUOLUN_UPDATE_REPO")); v != "" {
		return v
	}
	return defaultRepo
}

func ImageRef(tag string) string {
	img := strings.TrimSpace(os.Getenv("FENGHUOLUN_UPDATE_IMAGE"))
	if img == "" {
		img = "ghcr.io/yuxiaoyx/fenghuolun"
	}
	img = strings.TrimRight(img, ":")
	tag = NormalizeTag(tag)
	if tag == "" {
		tag = "latest"
	}
	return img + ":" + tag
}

func LatestRelease(force bool) (Release, error) {
	cacheMu.Lock()
	if !force && time.Now().Before(cachedUntil) && cached.Tag != "" {
		r := cached
		cacheMu.Unlock()
		return r, nil
	}
	cacheMu.Unlock()

	r, err := fetchLatest()
	if err != nil {
		return Release{}, err
	}
	cacheMu.Lock()
	cached = r
	cachedUntil = time.Now().Add(5 * time.Minute)
	cacheMu.Unlock()
	return r, nil
}

func fetchLatest() (Release, error) {
	repo := repoName()
	if repo == "-" {
		return Release{}, fmt.Errorf("disabled")
	}
	rel, err := getJSON[githubLatest](githubAPI + "/repos/" + repo + "/releases/latest")
	if err == nil && !rel.Draft && !rel.Prerelease && IsReleaseTag(rel.TagName) {
		return Release{
			Tag:     NormalizeTag(rel.TagName),
			URL:     rel.HTMLURL,
			Notes:   strings.TrimSpace(rel.Body),
			Fetched: time.Now(),
		}, nil
	}
	tags, err2 := getJSON[[]githubTag](githubAPI + "/repos/" + repo + "/tags?per_page=30")
	if err2 != nil {
		if err != nil {
			return Release{}, fmt.Errorf("github: %v", err)
		}
		return Release{}, fmt.Errorf("github tags: %v", err2)
	}
	names := make([]string, 0, len(tags))
	for _, t := range tags {
		names = append(names, t.Name)
	}
	best, err := PickLatestRelease(names)
	if err != nil {
		return Release{}, fmt.Errorf("github: no production release yet")
	}
	return Release{
		Tag:     best,
		URL:     "https://github.com/" + repo + "/releases/tag/" + best,
		Fetched: time.Now(),
	}, nil
}

func getJSON[T any](url string) (T, error) {
	var zero T
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return zero, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "fenghuolun")
	resp, err := httpClient.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return zero, err
	}
	if resp.StatusCode != http.StatusOK {
		return zero, fmt.Errorf("%s: %s", resp.Status, truncate(string(body), 180))
	}
	var out T
	if err := json.Unmarshal(body, &out); err != nil {
		return zero, err
	}
	return out, nil
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
