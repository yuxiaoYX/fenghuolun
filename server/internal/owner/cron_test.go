package owner

import "testing"

func TestCronPattern(t *testing.T) {
	if cronPattern("") != "" || cronPattern("off") != "" || cronPattern("0") != "" {
		t.Fatal("off")
	}
	if cronPattern("15m") != "@every 15m" {
		t.Fatalf("duration %q", cronPattern("15m"))
	}
	if cronPattern("@every 10m") != "@every 10m" {
		t.Fatal("passthrough")
	}
}
