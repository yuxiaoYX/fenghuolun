package clock

import (
	"encoding/json"
	"testing"
	"time"
)

func TestInstantJSONChina(t *testing.T) {
	in := Of(time.Date(2026, 1, 15, 4, 0, 0, 0, time.UTC))
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `"2026-01-15 12:00:00"` {
		t.Fatalf("got %s", b)
	}

	var fromRFC Instant
	if err := json.Unmarshal([]byte(`"2026-01-15T04:00:00Z"`), &fromRFC); err != nil {
		t.Fatal(err)
	}
	if !fromRFC.Time().Equal(in.Time()) {
		t.Fatalf("rfc %v want %v", fromRFC.Time(), in.Time())
	}

	var fromWall Instant
	if err := json.Unmarshal(b, &fromWall); err != nil {
		t.Fatal(err)
	}
	if !fromWall.Time().Equal(in.Time()) {
		t.Fatalf("wall %v want %v", fromWall.Time(), in.Time())
	}
}

func TestInstantZeroJSONNull(t *testing.T) {
	b, err := json.Marshal(Instant{})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "null" {
		t.Fatalf("got %s", b)
	}
	var z Instant
	if err := json.Unmarshal([]byte("null"), &z); err != nil {
		t.Fatal(err)
	}
	if !z.IsZero() {
		t.Fatalf("%v", z.Time())
	}
}

func TestParseEmpty(t *testing.T) {
	got, err := Parse("")
	if err != nil || !got.IsZero() {
		t.Fatalf("%v %v", got, err)
	}
}
