package store

import (
	"path/filepath"
	"testing"
	"time"

	"fenghuolun/internal/clock"
	"fenghuolun/internal/neta"
)

func sampleBinding() *Binding {
	fuel := 24.0
	return &Binding{
		Session:      NewSessionID(),
		RefreshHint:  "****xxxx",
		RefreshToken: "refresh-secret",
		AccessToken:  "access-secret",
		Meta: neta.VehicleMeta{
			VIN:        "TESTVIN0000000001",
			VinMasked:  "****0001",
			ModelCode:  "EP32",
			ModelName:  "哪吒L",
			IsExtender: true,
		},
		Snapshots: []neta.Snapshot{{
			FetchedAt: clock.Of(time.Unix(100, 0).UTC()),
			Extender:  &neta.Extender{FuelPct: &fuel},
		}},
		SyncStatus: "ok",
		SyncedAt:   clock.Of(time.Unix(100, 0).UTC()),
	}
}

func TestSQLitePutGetRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "t.db")
	s, err := OpenSQLite(p, "test-kek")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	fuel := 24.0
	b := &Binding{
		Session:      NewSessionID(),
		RefreshHint:  "****xxxx",
		RefreshToken: "refresh-secret",
		AccessToken:  "access-secret",
		Meta: neta.VehicleMeta{
			VIN:        "TESTVIN0000000001",
			VinMasked:  "****0001",
			ModelCode:  "EP32",
			ModelName:  "哪吒L",
			IsExtender: true,
		},
		Snapshots: []neta.Snapshot{{
			FetchedAt: clock.Of(time.Unix(100, 0).UTC()),
			Extender:  &neta.Extender{FuelPct: &fuel},
		}},
		SyncStatus: "ok",
		SyncedAt:   clock.Of(time.Unix(100, 0).UTC()),
	}
	if err := s.Put(b); err != nil {
		t.Fatal(err)
	}
	got := s.Get(b.Session)
	if got == nil {
		t.Fatal("nil")
	}
	if got.RefreshToken != "refresh-secret" || got.Meta.VIN != "TESTVIN0000000001" {
		t.Fatalf("%+v", got)
	}
	if len(got.Snapshots) != 1 || got.Snapshots[0].Extender == nil {
		t.Fatalf("snaps %+v", got.Snapshots)
	}
	s.Delete(b.Session)
	if s.Get(b.Session) != nil {
		t.Fatal("session should be gone")
	}
}

func TestSQLiteDisableClearsOwnerSession(t *testing.T) {
	p := filepath.Join(t.TempDir(), "t.db")
	s, err := OpenSQLite(p, "test-kek")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	b := &Binding{
		Session: NewSessionID(),
		Meta: neta.VehicleMeta{
			VIN:       "TESTVIN0000000001",
			VinMasked: "****0001",
			ModelCode: "EP32",
		},
		SyncStatus: "ok",
		SyncedAt:   clock.Of(time.Unix(100, 0).UTC()),
	}
	if err := s.Put(b); err != nil {
		t.Fatal(err)
	}
	if err := s.SetDisabled(b.ID, true); err != nil {
		t.Fatal(err)
	}
	if s.Get(b.Session) != nil {
		t.Fatal("disabled binding must drop owner session")
	}
	if s.GetByID(b.ID) != nil {
		t.Fatal("GetByID skips disabled")
	}
	rows, err := s.ListBindings()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || !rows[0].Disabled || rows[0].VinMasked != "****0001" {
		t.Fatalf("%+v", rows)
	}
}

func TestSQLiteLocationRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "t.db")
	s, err := OpenSQLite(p, "test-kek")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	lng, lat := 116.0, 40.0
	rep := time.Unix(1767225600, 0).UTC()
	b := &Binding{
		Session: NewSessionID(),
		Meta: neta.VehicleMeta{
			VIN:       "TESTVIN0000000001",
			VinMasked: "****0001",
			ModelCode: "EP32",
		},
		Snapshots: []neta.Snapshot{{
			FetchedAt: clock.Of(time.Unix(100, 0).UTC()),
			Location:  neta.Location{Lng: &lng, Lat: &lat, ReportedAt: clock.Ptr(&rep)},
		}},
		SyncStatus: "ok",
		SyncedAt:   clock.Of(time.Unix(100, 0).UTC()),
	}
	if err := s.Put(b); err != nil {
		t.Fatal(err)
	}
	// owner 路径：snapshot payload 含位置。
	got := s.Get(b.Session)
	if got == nil || len(got.Snapshots) != 1 {
		t.Fatalf("got %+v", got)
	}
	loc := got.Snapshots[0].Location
	if loc.Lng == nil || *loc.Lng != lng || loc.Lat == nil || *loc.Lat != lat {
		t.Fatalf("owner snapshot location %+v", loc)
	}
	// admin 路径：ListBindings 带出经纬度。
	rows, err := s.ListBindings()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Lng == nil || *rows[0].Lng != lng || rows[0].Lat == nil || *rows[0].Lat != lat {
		t.Fatalf("admin binding row %+v", rows)
	}
}

func TestSQLiteLocationPreservedWhenNewSnapshotHasNoCoords(t *testing.T) {
	p := filepath.Join(t.TempDir(), "t.db")
	s, err := OpenSQLite(p, "test-kek")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	lng, lat := 116.0, 40.0
	rep := time.Unix(1767225600, 0).UTC()
	b := &Binding{
		ID:      NewSessionID(),
		Session: NewSessionID(),
		Meta: neta.VehicleMeta{
			VIN:       "TESTVIN0000000001",
			VinMasked: "****0001",
			ModelCode: "EP32",
		},
		Snapshots: []neta.Snapshot{{
			FetchedAt: clock.Of(time.Unix(100, 0).UTC()),
			Location:  neta.Location{Lng: &lng, Lat: &lat, ReportedAt: clock.Ptr(&rep)},
		}},
		SyncStatus: "ok",
		SyncedAt:   clock.Of(time.Unix(100, 0).UTC()),
	}
	if err := s.Put(b); err != nil {
		t.Fatal(err)
	}
	// 第二次同步：车进地库无 GPS，快照无坐标。旧坐标必须保留，不能被 NULL 覆盖。
	b2 := &Binding{
		ID:      b.ID,
		Session: b.Session,
		Meta:    b.Meta,
		Snapshots: []neta.Snapshot{{
			FetchedAt: clock.Of(time.Unix(200, 0).UTC()),
			// Location 零值：Lng/Lat 为 nil
		}},
		SyncStatus: "ok",
		SyncedAt:   clock.Of(time.Unix(200, 0).UTC()),
	}
	if err := s.Put(b2); err != nil {
		t.Fatal(err)
	}
	rows, err := s.ListBindings()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Lng == nil || *rows[0].Lng != lng || rows[0].Lat == nil || *rows[0].Lat != lat {
		t.Fatalf("coords must be preserved across no-GPS sync: %+v", rows)
	}
}

func TestSQLiteAutoRowTimes(t *testing.T) {
	p := filepath.Join(t.TempDir(), "t.db")
	s, err := OpenSQLite(p, "test-kek")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	b := sampleBinding()
	if err := s.Put(b); err != nil {
		t.Fatal(err)
	}
	created, err := s.db.Model("binding").Where("id", b.ID).Value("created_at")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := s.db.Model("binding").Where("id", b.ID).Value("updated_at")
	if err != nil {
		t.Fatal(err)
	}
	deleted, err := s.db.Model("binding").Where("id", b.ID).Value("deleted_at")
	if err != nil {
		t.Fatal(err)
	}
	if created.Int64() < 1_000_000_000 {
		t.Fatalf("created_at should be unix seconds, got %v", created)
	}
	if updated.Int64() < created.Int64() {
		t.Fatalf("updated_at %v created_at %v", updated, created)
	}
	if deleted.Int64() != 0 {
		t.Fatalf("deleted_at want 0 got %v", deleted)
	}
	firstCreated := created.Int64()
	b.Meta.Nickname = "改名"
	if err := s.Put(b); err != nil {
		t.Fatal(err)
	}
	created2, err := s.db.Model("binding").Where("id", b.ID).Value("created_at")
	if err != nil {
		t.Fatal(err)
	}
	if created2.Int64() != firstCreated {
		t.Fatalf("created_at must not change on update: %v -> %v", firstCreated, created2)
	}
}

func TestSQLiteSoftDeleteOwnerSession(t *testing.T) {
	p := filepath.Join(t.TempDir(), "t.db")
	s, err := OpenSQLite(p, "test-kek")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	b := sampleBinding()
	if err := s.Put(b); err != nil {
		t.Fatal(err)
	}
	s.Delete(b.Session)
	if s.Get(b.Session) != nil {
		t.Fatal("soft-deleted session must not load")
	}
	n, err := s.db.Model("owner_session").Unscoped().Where("token_hash", hashToken(b.Session)).Count()
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("session row should remain after soft delete, n=%d", n)
	}
	deleted, err := s.db.Model("owner_session").Unscoped().Where("token_hash", hashToken(b.Session)).Value("deleted_at")
	if err != nil {
		t.Fatal(err)
	}
	if deleted.Int64() == 0 {
		t.Fatal("deleted_at should be unix seconds after Delete")
	}
}
