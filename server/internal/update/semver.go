package update

import (
	"fmt"
	"strconv"
	"strings"
)

type triple struct {
	maj, min, patch int
}

func parseRelease(tag string) (triple, bool) {
	s := strings.TrimSpace(tag)
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimPrefix(s, "V")
	if s == "" || strings.ContainsAny(s, "-+") {
		return triple{}, false
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return triple{}, false
	}
	maj, err1 := strconv.Atoi(parts[0])
	min, err2 := strconv.Atoi(parts[1])
	pat, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return triple{}, false
	}
	if maj < 0 || min < 0 || pat < 0 {
		return triple{}, false
	}
	return triple{maj, min, pat}, true
}

// IsReleaseTag is vMAJOR.MINOR.PATCH with no pre-release suffix.
func IsReleaseTag(tag string) bool {
	_, ok := parseRelease(tag)
	return ok
}

// CompareTags returns -1 if a<b, 0 if equal, 1 if a>b.
// Non-release tags (dev, sha-*, rc) compare as older than any release.
func CompareTags(a, b string) int {
	ta, oa := parseRelease(a)
	tb, ob := parseRelease(b)
	switch {
	case !oa && !ob:
		return strings.Compare(strings.TrimSpace(a), strings.TrimSpace(b))
	case !oa:
		return -1
	case !ob:
		return 1
	case ta.maj != tb.maj:
		return cmpInt(ta.maj, tb.maj)
	case ta.min != tb.min:
		return cmpInt(ta.min, tb.min)
	default:
		return cmpInt(ta.patch, tb.patch)
	}
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func NormalizeTag(tag string) string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return tag
	}
	if _, ok := parseRelease(tag); ok && !strings.HasPrefix(tag, "v") && !strings.HasPrefix(tag, "V") {
		return "v" + tag
	}
	if strings.HasPrefix(tag, "V") && len(tag) > 1 {
		return "v" + tag[1:]
	}
	return tag
}

func PickLatestRelease(tags []string) (string, error) {
	best := ""
	for _, t := range tags {
		t = NormalizeTag(t)
		if !IsReleaseTag(t) {
			continue
		}
		if best == "" || CompareTags(t, best) > 0 {
			best = t
		}
	}
	if best == "" {
		return "", fmt.Errorf("no release tags")
	}
	return best, nil
}
