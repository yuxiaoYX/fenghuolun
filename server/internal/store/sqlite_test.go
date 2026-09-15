package store

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fenghuolun/internal/clock"
	"fenghuolun/internal/config"
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

func TestSQLiteNicknamePreservedAndListed(t *testing.T) {
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
	if err := s.SetNickname(b.ID, "家里那辆"); err != nil {
		t.Fatal(err)
	}
	b.Meta.Nickname = "官方名"
	if err := s.Put(b); err != nil {
		t.Fatal(err)
	}
	got := s.Get(b.Session)
	if got == nil || got.Meta.Nickname != "家里那辆" {
		t.Fatalf("nickname %+v", got)
	}
	items, total, err := s.ListSnapshotSummaries(b.ID, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total < 1 || len(items) < 1 || items[0].ID == "" {
		t.Fatalf("summaries %d %+v", total, items)
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

func TestSQLiteBeginJobExclusiveAndFinish(t *testing.T) {
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
	jobID, err := s.BeginJob(b.ID, "admin")
	if err != nil || jobID == "" {
		t.Fatal(err)
	}
	if _, err := s.BeginJob(b.ID, "admin"); err == nil {
		t.Fatal("second running job must fail")
	}
	if err := s.FinishJob(jobID, "ok", "", "", time.Unix(300, 0).UTC()); err != nil {
		t.Fatal(err)
	}
	jobs, total, err := s.ListJobs(b.ID, "", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Fatalf("put + begin/finish want 2 got %d", total)
	}
	if jobs[0].Status != "ok" || jobs[0].Kind != "admin" {
		t.Fatalf("latest %+v", jobs[0])
	}
	if _, err := s.BeginJob(b.ID, "cron"); err != nil {
		t.Fatal(err)
	}
}

func TestSQLiteSyncJobHistory(t *testing.T) {
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
	if err := s.MarkSync(b.ID, "cron", "upstream", "官方云暂不可用", time.Unix(200, 0).UTC()); err != nil {
		t.Fatal(err)
	}
	jobs, total, err := s.ListJobs(b.ID, "", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Fatalf("want 2 jobs got %d %+v", total, jobs)
	}
	if jobs[0].Status != "upstream" || jobs[0].Kind != "cron" {
		t.Fatalf("latest %+v", jobs[0])
	}
	if jobs[1].Status != "ok" {
		t.Fatalf("first %+v", jobs[1])
	}
	rows, err := s.ListBindings()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].SyncStatus != "upstream" {
		t.Fatalf("list latest job %+v", rows)
	}
	if s.GetByIDAny(b.ID) == nil {
		t.Fatal("GetByIDAny")
	}
}

func TestSQLiteSettingsAndAdminSession(t *testing.T) {
	p := filepath.Join(t.TempDir(), "t.db")
	s, err := OpenFromConfig(config.Config{
		SQLitePath:    p,
		TokenKEK:      "test-kek",
		AdminUser:     "admin",
		AdminPassword: "secret-pass-xx",
		CronSync:      "15m",
		CORSOrigins:   "http://127.0.0.1:5173",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	got := s.Settings()
	if got.CronSync != "15m" || got.StaleAfterSec != DefaultStaleAfterSec {
		t.Fatalf("%+v", got)
	}
	saved, err := s.SaveSettings(Settings{CronSync: "off", CORSOrigins: "http://127.0.0.1:5173", SnapshotKeep: 2, StaleAfterSec: 3600})
	if err != nil {
		t.Fatal(err)
	}
	if saved.CronSync != "off" || saved.SnapshotKeep != 2 || saved.StaleAfterSec != 3600 {
		t.Fatalf("%+v", saved)
	}
	tok, err := s.AdminLogin("admin", "secret-pass-xx")
	if err != nil || tok == "" {
		t.Fatal(err)
	}
	if s.AdminUsername(tok) != "admin" {
		t.Fatalf("username %q", s.AdminUsername(tok))
	}
	if err := s.AdminChangePassword(tok, "secret-pass-xx", "new-pass-xx"); err != nil {
		t.Fatal(err)
	}
	s.AdminLogout(tok)
	if s.AdminOK(tok) {
		t.Fatal("logout should drop session")
	}
	if _, err := s.AdminLogin("admin", "new-pass-xx"); err != nil {
		t.Fatal(err)
	}
}

func TestSQLiteChangePasswordKicksOtherSessions(t *testing.T) {
	p := filepath.Join(t.TempDir(), "t.db")
	s, err := OpenFromConfig(config.Config{
		SQLitePath:    p,
		TokenKEK:      "test-kek",
		AdminUser:     "admin",
		AdminPassword: "secret-pass-xx",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	a, err := s.AdminLogin("admin", "secret-pass-xx")
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.AdminLogin("admin", "secret-pass-xx")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AdminChangePassword(a, "secret-pass-xx", "new-pass-xx"); err != nil {
		t.Fatal(err)
	}
	if !s.AdminOK(a) {
		t.Fatal("current session should remain")
	}
	if s.AdminOK(b) {
		t.Fatal("other sessions must be kicked")
	}
}

func TestSQLiteTableRedaction(t *testing.T) {
	p := filepath.Join(t.TempDir(), "t.db")
	s, err := OpenSQLite(p, "test-kek")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.Put(sampleBinding()); err != nil {
		t.Fatal(err)
	}
	page, err := s.ListTable("binding", "", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	raw := strings.ToLower(strings.Join(keysOf(page.Items), " "))
	if strings.Contains(raw, "cipher") && !strings.Contains(raw, "has_") {
		t.Fatalf("must not leak cipher columns: %v", page.Items)
	}
	for _, row := range page.Items {
		for k, v := range row {
			if k == "vin_cipher" || k == "refresh_cipher" || k == "access_cipher" {
				t.Fatalf("cipher field %s=%v", k, v)
			}
			if strings.Contains(fmtSprint(v), "TESTVIN0000000001") {
				t.Fatalf("full VIN in table: %v", row)
			}
		}
	}
}

func keysOf(items []map[string]any) []string {
	out := []string{}
	for _, item := range items {
		for k := range item {
			out = append(out, k)
		}
	}
	return out
}

func fmtSprint(v any) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.Join([]string{stringify(v)}, ""), "\x00", ""))
}

func stringify(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return ""
	}
}
