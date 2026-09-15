package crypto

import "testing"

func TestSealOpen(t *testing.T) {
	k := KEK("test-kek")
	c, err := Seal(k, "refresh-token-value")
	if err != nil {
		t.Fatal(err)
	}
	s, err := Open(k, c)
	if err != nil || s != "refresh-token-value" {
		t.Fatalf("%q %v", s, err)
	}
}
