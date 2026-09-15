package neta

import (
	"testing"
	"time"

	"fenghuolun/internal/clock"
)

func TestSnapshotStale(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	old := clock.Of(now.Add(-3 * time.Hour))
	fresh := clock.Of(now.Add(-time.Hour))
	staleSnap := Snapshot{ReportedAt: &old}
	if !staleSnap.Stale(now) {
		t.Fatal("reported 3h ago should be stale")
	}
	freshSnap := Snapshot{ReportedAt: &fresh}
	if freshSnap.Stale(now) {
		t.Fatal("reported 1h ago should not be stale")
	}
	if (Snapshot{}).Stale(now) {
		t.Fatal("missing reportedAt is unknown, not stale")
	}
}
