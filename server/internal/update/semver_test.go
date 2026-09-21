package update

import "testing"

func TestCompareTags(t *testing.T) {
	if CompareTags("v0.1.0", "v0.1.1") >= 0 {
		t.Fatal("0.1.0 should be older")
	}
	if CompareTags("v0.1.1", "v0.1.1") != 0 {
		t.Fatal("equal")
	}
	if CompareTags("v0.2.0", "v0.1.9") <= 0 {
		t.Fatal("0.2.0 newer")
	}
	if CompareTags("dev", "v0.1.0") >= 0 {
		t.Fatal("dev older than release")
	}
	if !IsReleaseTag("v1.2.3") || IsReleaseTag("v1.2.3-rc.1") || IsReleaseTag("sha-abc") {
		t.Fatal("release filter")
	}
}

func TestPickLatestRelease(t *testing.T) {
	got, err := PickLatestRelease([]string{"v0.1.0", "v0.1.1-rc.1", "master", "v0.2.0"})
	if err != nil || got != "v0.2.0" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := PickLatestRelease([]string{"dev", "foo"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeTag(t *testing.T) {
	if NormalizeTag("0.1.0") != "v0.1.0" {
		t.Fatal(NormalizeTag("0.1.0"))
	}
}
