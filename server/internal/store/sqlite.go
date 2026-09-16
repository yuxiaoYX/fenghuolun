package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fenghuolun/internal/clock"
	"fenghuolun/internal/crypto"
	"fenghuolun/internal/model/do"
	"fenghuolun/internal/model/entity"
	"fenghuolun/internal/neta"

	"github.com/gogf/gf/v2/database/gdb"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/gogf/gf/contrib/drivers/sqlite/v2"
)

type SQLite struct {
	db  gdb.DB
	kek []byte
}

func OpenSQLite(path string, kekRaw string) (*SQLite, error) {
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, err
		}
		path = filepath.ToSlash(path)
	}
	node := gdb.ConfigNode{
		Type:             "sqlite",
		Name:             path,
		Extra:            "busy_timeout=5000&foreign_keys=1",
		MaxOpenConnCount: 1,
		MaxIdleConnCount: 1,
	}
	db, err := gdb.New(node)
	if err != nil {
		return nil, err
	}
	s := &SQLite{db: db, kek: crypto.KEK(kekRaw)}
	if err := s.migrate(); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLite) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close(s.ctx())
}

func (s *SQLite) ctx() context.Context {
	return context.Background()
}

func (s *SQLite) migrate() error {
	ctx := s.ctx()
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS binding (
  id TEXT PRIMARY KEY,
  vin_masked TEXT NOT NULL DEFAULT '',
  vin_cipher BLOB,
  model_code TEXT NOT NULL DEFAULT '',
  model_name TEXT NOT NULL DEFAULT '',
  nickname TEXT NOT NULL DEFAULT '',
  trim TEXT NOT NULL DEFAULT '',
  is_extender INTEGER NOT NULL DEFAULT 0,
  refresh_cipher BLOB,
  access_cipher BLOB,
  refresh_hint TEXT NOT NULL DEFAULT '',
  disabled INTEGER NOT NULL DEFAULT 0,
  lng REAL,
  lat REAL,
  loc_reported_at INTEGER,
  created_at INTEGER,
  updated_at INTEGER,
  deleted_at INTEGER DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS owner_session (
  token_hash TEXT PRIMARY KEY,
  binding_id TEXT NOT NULL,
  created_at INTEGER,
  updated_at INTEGER,
  deleted_at INTEGER DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS snapshot (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  binding_id TEXT NOT NULL,
  fetched_at INTEGER NOT NULL,
  payload TEXT NOT NULL,
  created_at INTEGER,
  updated_at INTEGER,
  deleted_at INTEGER DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS energy (
  binding_id TEXT PRIMARY KEY,
  payload TEXT NOT NULL,
  fetched_at INTEGER NOT NULL,
  created_at INTEGER,
  updated_at INTEGER,
  deleted_at INTEGER DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS sync_job (
  id TEXT PRIMARY KEY,
  binding_id TEXT NOT NULL,
  kind TEXT NOT NULL DEFAULT 'unknown',
  status TEXT NOT NULL,
  error_public TEXT NOT NULL DEFAULT '',
  error_internal TEXT NOT NULL DEFAULT '',
  started_at INTEGER NOT NULL DEFAULT 0,
  finished_at INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER,
  updated_at INTEGER,
  deleted_at INTEGER DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS app_settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL DEFAULT '',
  created_at INTEGER,
  updated_at INTEGER,
  deleted_at INTEGER DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS admin_user (
  username TEXT PRIMARY KEY,
  password_hash TEXT NOT NULL,
  created_at INTEGER,
  updated_at INTEGER,
  deleted_at INTEGER DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS admin_session (
  token_hash TEXT PRIMARY KEY,
  created_at INTEGER,
  updated_at INTEGER,
  deleted_at INTEGER DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS owner_ledger (
  binding_id TEXT PRIMARY KEY,
  elec_cny_per_kwh REAL,
  fuel_cny_per_l REAL,
  tank_l REAL,
  created_at INTEGER,
  updated_at INTEGER,
  deleted_at INTEGER DEFAULT 0
)`,
		`CREATE TABLE IF NOT EXISTS fill_event (
  id TEXT PRIMARY KEY,
  binding_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  source TEXT NOT NULL DEFAULT 'manual',
  status TEXT NOT NULL DEFAULT 'draft',
  from_fetched INTEGER NOT NULL DEFAULT 0,
  to_fetched INTEGER NOT NULL DEFAULT 0,
  started_at INTEGER NOT NULL DEFAULT 0,
  finished_at INTEGER NOT NULL DEFAULT 0,
  odo_start REAL,
  odo_end REAL,
  soc_start REAL,
  soc_end REAL,
  fuel_start REAL,
  fuel_end REAL,
  energy_kwh REAL,
  liters REAL,
  paid_cny REAL,
  note TEXT NOT NULL DEFAULT '',
  created_at INTEGER,
  updated_at INTEGER,
  deleted_at INTEGER DEFAULT 0
)`,
	} {
		if _, err := s.db.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	// 旧库补列。列已存在时 ALTER 会失败，忽略。
	for _, stmt := range []string{
		`ALTER TABLE binding ADD COLUMN lng REAL`,
		`ALTER TABLE binding ADD COLUMN lat REAL`,
		`ALTER TABLE binding ADD COLUMN loc_reported_at INTEGER`,
		`ALTER TABLE binding ADD COLUMN created_at INTEGER`,
		`ALTER TABLE binding ADD COLUMN deleted_at INTEGER DEFAULT 0`,
		`ALTER TABLE owner_session ADD COLUMN updated_at INTEGER`,
		`ALTER TABLE owner_session ADD COLUMN deleted_at INTEGER DEFAULT 0`,
		`ALTER TABLE snapshot ADD COLUMN created_at INTEGER`,
		`ALTER TABLE snapshot ADD COLUMN updated_at INTEGER`,
		`ALTER TABLE snapshot ADD COLUMN deleted_at INTEGER DEFAULT 0`,
		`ALTER TABLE energy ADD COLUMN created_at INTEGER`,
		`ALTER TABLE energy ADD COLUMN updated_at INTEGER`,
		`ALTER TABLE energy ADD COLUMN deleted_at INTEGER DEFAULT 0`,
		`ALTER TABLE sync_job ADD COLUMN created_at INTEGER`,
		`ALTER TABLE sync_job ADD COLUMN updated_at INTEGER`,
		`ALTER TABLE sync_job ADD COLUMN deleted_at INTEGER DEFAULT 0`,
		`ALTER TABLE admin_user ADD COLUMN created_at INTEGER`,
		`ALTER TABLE admin_user ADD COLUMN updated_at INTEGER`,
		`ALTER TABLE admin_user ADD COLUMN deleted_at INTEGER DEFAULT 0`,
		`ALTER TABLE admin_session ADD COLUMN updated_at INTEGER`,
		`ALTER TABLE admin_session ADD COLUMN deleted_at INTEGER DEFAULT 0`,
		`ALTER TABLE admin_session ADD COLUMN username TEXT NOT NULL DEFAULT ''`,
	} {
		_, _ = s.db.Exec(ctx, stmt)
	}
	_, _ = s.db.Exec(ctx, `UPDATE binding SET created_at = updated_at WHERE created_at IS NULL OR created_at = 0`)
	_, _ = s.db.Exec(ctx, `UPDATE binding SET deleted_at = 0 WHERE deleted_at IS NULL`)
	_, _ = s.db.Exec(ctx, `UPDATE owner_session SET updated_at = created_at WHERE updated_at IS NULL OR updated_at = 0`)
	_, _ = s.db.Exec(ctx, `UPDATE owner_session SET deleted_at = 0 WHERE deleted_at IS NULL`)
	for _, table := range []string{"snapshot", "energy", "sync_job", "admin_user", "admin_session", "app_settings"} {
		_, _ = s.db.Exec(ctx, `UPDATE `+table+` SET deleted_at = 0 WHERE deleted_at IS NULL`)
	}
	return s.migrateSyncJobHistory()
}

func (s *SQLite) hasColumn(table, col string) bool {
	all, err := s.db.GetAll(s.ctx(), "PRAGMA table_info(`"+table+"`)")
	if err != nil {
		return false
	}
	for _, rec := range all {
		if rec["name"].String() == col {
			return true
		}
	}
	return false
}

func (s *SQLite) migrateSyncJobHistory() error {
	if s.hasColumn("sync_job", "id") {
		_, _ = s.db.Exec(s.ctx(), `CREATE INDEX IF NOT EXISTS idx_sync_job_binding_started ON sync_job(binding_id, started_at)`)
		return nil
	}
	ctx := s.ctx()
	if _, err := s.db.Exec(ctx, `ALTER TABLE sync_job RENAME TO sync_job_v1`); err != nil {
		return err
	}
	if _, err := s.db.Exec(ctx, `CREATE TABLE sync_job (
  id TEXT PRIMARY KEY,
  binding_id TEXT NOT NULL,
  kind TEXT NOT NULL DEFAULT 'unknown',
  status TEXT NOT NULL,
  error_public TEXT NOT NULL DEFAULT '',
  error_internal TEXT NOT NULL DEFAULT '',
  started_at INTEGER NOT NULL DEFAULT 0,
  finished_at INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER,
  updated_at INTEGER,
  deleted_at INTEGER DEFAULT 0
)`); err != nil {
		return err
	}
	if _, err := s.db.Exec(ctx, `INSERT INTO sync_job (id, binding_id, kind, status, error_public, error_internal, started_at, finished_at, created_at, updated_at, deleted_at)
SELECT lower(hex(randomblob(16))), binding_id, 'unknown', status, error_public, '', synced_at, synced_at, created_at, updated_at, IFNULL(deleted_at, 0)
FROM sync_job_v1`); err != nil {
		return err
	}
	if _, err := s.db.Exec(ctx, `DROP TABLE sync_job_v1`); err != nil {
		return err
	}
	_, _ = s.db.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_sync_job_binding_started ON sync_job(binding_id, started_at)`)
	return nil
}

func (s *SQLite) Put(b *Binding) error {
	if b.ID == "" {
		b.ID = NewSessionID()
	}
	vinC, err := crypto.Seal(s.kek, b.Meta.VIN)
	if err != nil {
		return err
	}
	rtC, err := crypto.Seal(s.kek, b.RefreshToken)
	if err != nil {
		return err
	}
	atC, err := crypto.Seal(s.kek, b.AccessToken)
	if err != nil {
		return err
	}
	ext := 0
	if b.Meta.IsExtender {
		ext = 1
	}
	keep := s.SnapshotKeep()
	nick := b.Meta.Nickname
	var existing entity.Binding
	if err := s.db.Model("binding").Ctx(s.ctx()).Where("id", b.ID).Scan(&existing); err == nil && existing.Nickname != "" {
		nick = existing.Nickname
	}
	row := do.Binding{
		Id:            b.ID,
		VinMasked:     b.Meta.VinMasked,
		VinCipher:     vinC,
		ModelCode:     b.Meta.ModelCode,
		ModelName:     b.Meta.ModelName,
		Nickname:      nick,
		Trim:          b.Meta.Trim,
		IsExtender:    ext,
		RefreshCipher: rtC,
		AccessCipher:  atC,
		RefreshHint:   b.RefreshHint,
	}
	if len(b.Snapshots) > 0 {
		last := b.Snapshots[len(b.Snapshots)-1]
		if last.Location.Lng != nil && last.Location.Lat != nil {
			row.Lng = *last.Location.Lng
			row.Lat = *last.Location.Lat
			if last.Location.ReportedAt != nil {
				row.LocReportedAt = last.Location.ReportedAt.Unix()
			}
		}
	}
	ctx := s.ctx()
	err = s.db.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if _, err := tx.Model("binding").Ctx(ctx).Data(row).OnConflict("id").Save(); err != nil {
			return err
		}
		if b.Session != "" {
			if _, err := tx.Model("owner_session").Ctx(ctx).Data(do.OwnerSession{
				TokenHash: hashToken(b.Session),
				BindingId: b.ID,
			}).OnConflict("token_hash").Save(); err != nil {
				return err
			}
		}
		if len(b.Snapshots) > 0 {
			last := b.Snapshots[len(b.Snapshots)-1]
			prev, err := tx.Model("snapshot").Ctx(ctx).Where("binding_id", b.ID).
				OrderDesc("fetched_at").OrderDesc("id").Value("fetched_at")
			if err != nil {
				return err
			}
			if last.FetchedAt.Unix() != prev.Int64() {
				raw, err := json.Marshal(last)
				if err != nil {
					return err
				}
				if _, err := tx.Model("snapshot").Ctx(ctx).Data(do.Snapshot{
					BindingId: b.ID,
					FetchedAt: last.FetchedAt.Unix(),
					Payload:   string(raw),
				}).Insert(); err != nil {
					return err
				}
				if err := pruneSnapshots(ctx, tx, b.ID, keep); err != nil {
					return err
				}
			}
		}
		if b.Energy.Days != nil || b.Energy.TotalKwh != nil {
			raw, err := json.Marshal(b.Energy)
			if err != nil {
				return err
			}
			fetched := b.Energy.FetchedAt.Unix()
			if fetched == 0 {
				fetched = time.Now().Unix()
			}
			if _, err := tx.Model("energy").Ctx(ctx).Data(do.Energy{
				BindingId: b.ID,
				Payload:   string(raw),
				FetchedAt: fetched,
			}).OnConflict("binding_id").Save(); err != nil {
				return err
			}
		}
		at := b.SyncedAt.Time()
		if at.IsZero() {
			at = time.Now().UTC()
		}
		if b.JobID != "" {
			_, err := tx.Model("sync_job").Ctx(ctx).Where("id", b.JobID).Data(do.SyncJob{
				Status:        b.SyncStatus,
				ErrorPublic:   b.SyncError,
				ErrorInternal: "",
				FinishedAt:    at.Unix(),
			}).Update()
			return err
		}
		_, err := tx.Model("sync_job").Ctx(ctx).Data(syncJobData(b.ID, b.SyncKind, b.SyncStatus, b.SyncError, at)).Insert()
		return err
	})
	if err != nil {
		return err
	}
	_ = s.SyncFills(b.ID)
	return nil
}

func syncJobData(bindingID, kind, status, publicErr string, at time.Time) do.SyncJob {
	unix := at.Unix()
	if kind == "" {
		kind = "manual"
	}
	return do.SyncJob{
		Id:          NewSessionID(),
		BindingId:   bindingID,
		Kind:        kind,
		Status:      status,
		ErrorPublic: publicErr,
		StartedAt:   unix,
		FinishedAt:  unix,
	}
}

func pruneSnapshots(ctx context.Context, tx gdb.TX, bindingID string, keep int) error {
	if keep <= 0 {
		return nil
	}
	var ids []int64
	if err := tx.Model("snapshot").Ctx(ctx).Where("binding_id", bindingID).
		OrderDesc("fetched_at").OrderDesc("id").Fields("id").Scan(&ids); err != nil {
		return err
	}
	if len(ids) <= keep {
		return nil
	}
	_, err := tx.Model("snapshot").Ctx(ctx).WhereIn("id", ids[keep:]).Delete()
	return err
}

func (s *SQLite) Get(session string) *Binding {
	if session == "" {
		return nil
	}
	v, err := s.db.Model("owner_session").Ctx(s.ctx()).Where("token_hash", hashToken(session)).Value("binding_id")
	if err != nil || v.IsEmpty() {
		return nil
	}
	return s.loadBinding(v.String(), session, false)
}

func (s *SQLite) GetByID(id string) *Binding {
	if id == "" {
		return nil
	}
	return s.loadBinding(id, "", false)
}

func (s *SQLite) GetByIDAny(id string) *Binding {
	if id == "" {
		return nil
	}
	return s.loadBinding(id, "", true)
}

func (s *SQLite) SetNickname(id, nick string) error {
	if id == "" {
		return fmt.Errorf("invalid_request: missing binding id")
	}
	_, err := s.db.Model("binding").Ctx(s.ctx()).Where("id", id).Data(do.Binding{
		Nickname: strings.TrimSpace(nick),
	}).Update()
	return err
}

type SnapshotSummary struct {
	ID           string         `json:"id"`
	FetchedAt    clock.Instant  `json:"fetchedAt"`
	ReportedAt   *clock.Instant `json:"reportedAt"`
	SocPct       *float64       `json:"socPct"`
	EvRangeKm    *float64       `json:"evRangeKm"`
	TotalRangeKm *float64       `json:"totalRangeKm"`
	OdometerKm   *float64       `json:"odometerKm"`
	ChargeStatus string         `json:"chargeStatus"`
	Online       *bool          `json:"online"`
	Stale        bool           `json:"stale"`
}

func (s *SQLite) ListSnapshotSummaries(bindingID string, page, pageSize int) ([]SnapshotSummary, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	ctx := s.ctx()
	m := s.db.Model("snapshot").Ctx(ctx).Where("binding_id", bindingID)
	total, err := m.Count()
	if err != nil {
		return nil, 0, err
	}
	var rows []entity.Snapshot
	if err := m.OrderDesc("fetched_at").OrderDesc("id").Page(page, pageSize).Scan(&rows); err != nil {
		return nil, 0, err
	}
	now := time.Now()
	staleAfter := s.StaleAfter()
	out := make([]SnapshotSummary, 0, len(rows))
	for _, row := range rows {
		var snap neta.Snapshot
		_ = json.Unmarshal([]byte(row.Payload), &snap)
		item := SnapshotSummary{
			ID:           fmt.Sprintf("%d", row.Id),
			FetchedAt:    snap.FetchedAt,
			ReportedAt:   snap.ReportedAt,
			SocPct:       snap.Power.SocPct,
			EvRangeKm:    snap.Power.EvRangeKm,
			TotalRangeKm: snap.Power.TotalRangeKm,
			OdometerKm:   snap.OdometerKm,
			ChargeStatus: snap.Power.ChargeStatus,
			Online:       snap.Online,
			Stale:        snap.StaleSince(now, staleAfter),
		}
		if item.FetchedAt.IsZero() && row.FetchedAt > 0 {
			item.FetchedAt = clock.Of(time.Unix(row.FetchedAt, 0).UTC())
		}
		out = append(out, item)
	}
	return out, total, nil
}

func (s *SQLite) loadBinding(id, session string, allowDisabled bool) *Binding {
	ctx := s.ctx()
	var row entity.Binding
	if err := s.db.Model("binding").Ctx(ctx).Where("id", id).Scan(&row); err != nil || row.Id == "" {
		return nil
	}
	if row.Disabled == 1 && !allowDisabled {
		return nil
	}
	vin, _ := crypto.Open(s.kek, row.VinCipher)
	rt, _ := crypto.Open(s.kek, row.RefreshCipher)
	at, _ := crypto.Open(s.kek, row.AccessCipher)
	b := &Binding{
		ID:           id,
		Session:      session,
		RefreshHint:  row.RefreshHint,
		AccessToken:  at,
		RefreshToken: rt,
		Disabled:     row.Disabled == 1,
		Meta: neta.VehicleMeta{
			VIN:        vin,
			VinMasked:  row.VinMasked,
			Nickname:   row.Nickname,
			ModelCode:  row.ModelCode,
			ModelName:  row.ModelName,
			Trim:       row.Trim,
			IsExtender: row.IsExtender == 1,
		},
	}
	var snapRows []entity.Snapshot
	if err := s.db.Model("snapshot").Ctx(ctx).Where("binding_id", id).
		OrderAsc("fetched_at").OrderAsc("id").Scan(&snapRows); err == nil {
		for _, row := range snapRows {
			var snap neta.Snapshot
			if json.Unmarshal([]byte(row.Payload), &snap) == nil {
				b.Snapshots = append(b.Snapshots, snap)
			}
		}
	}
	var energyRow entity.Energy
	if err := s.db.Model("energy").Ctx(ctx).Where("binding_id", id).Scan(&energyRow); err == nil && energyRow.Payload != "" {
		_ = json.Unmarshal([]byte(energyRow.Payload), &b.Energy)
	}
	var job entity.SyncJob
	if err := s.db.Model("sync_job").Ctx(ctx).Where("binding_id", id).
		OrderDesc("started_at").OrderDesc("created_at").Limit(1).Scan(&job); err == nil && job.BindingId != "" {
		b.SyncStatus = job.Status
		b.SyncError = job.ErrorPublic
		b.SyncKind = job.Kind
		if unix := jobFinishedUnix(job); unix > 0 {
			b.SyncedAt = clock.Of(time.Unix(unix, 0).UTC())
		}
	}
	return b
}

func jobFinishedUnix(j entity.SyncJob) int64 {
	if j.FinishedAt > 0 {
		return j.FinishedAt
	}
	return j.StartedAt
}

func (s *SQLite) Delete(session string) {
	b := s.Get(session)
	if b == nil {
		return
	}
	ctx := s.ctx()
	_, _ = s.db.Model("owner_session").Ctx(ctx).Where("token_hash", hashToken(session)).Delete()
	_, _ = s.db.Model("binding").Ctx(ctx).Where("id", b.ID).Data(do.Binding{
		RefreshCipher: gdb.Raw("NULL"),
		AccessCipher:  gdb.Raw("NULL"),
	}).Update()
}

func (s *SQLite) SetDisabled(id string, disabled bool) error {
	if id == "" {
		return fmt.Errorf("invalid_request: missing binding id")
	}
	flag := 0
	if disabled {
		flag = 1
	}
	ctx := s.ctx()
	res, err := s.db.Model("binding").Ctx(ctx).Where("id", id).Data(do.Binding{Disabled: flag}).Update()
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("not_found")
	}
	if disabled {
		_, _ = s.db.Model("owner_session").Ctx(ctx).Where("binding_id", id).Delete()
	}
	return nil
}

func (s *SQLite) ListActiveIDs() ([]string, error) {
	var ids []string
	err := s.db.Model("binding").Ctx(s.ctx()).Where("disabled", 0).OrderDesc("updated_at").Fields("id").Scan(&ids)
	return ids, err
}

func (s *SQLite) MarkSync(id, kind, status, publicErr string, at time.Time) error {
	if id == "" {
		return fmt.Errorf("invalid_request: missing binding id")
	}
	_, err := s.db.Model("sync_job").Ctx(s.ctx()).Data(syncJobData(id, kind, status, publicErr, at)).Insert()
	return err
}

func (s *SQLite) BeginJob(bindingID, kind string) (string, error) {
	if bindingID == "" {
		return "", fmt.Errorf("invalid_request: missing binding id")
	}
	ctx := s.ctx()
	n, err := s.db.Model("sync_job").Ctx(ctx).Where("binding_id", bindingID).Where("status", "running").Count()
	if err != nil {
		return "", err
	}
	if n > 0 {
		return "", fmt.Errorf("invalid_request: sync already running")
	}
	id := NewSessionID()
	now := time.Now().Unix()
	if kind == "" {
		kind = "manual"
	}
	_, err = s.db.Model("sync_job").Ctx(ctx).Data(do.SyncJob{
		Id:         id,
		BindingId:  bindingID,
		Kind:       kind,
		Status:     "running",
		StartedAt:  now,
		FinishedAt: 0,
	}).Insert()
	return id, err
}

func (s *SQLite) FinishJob(id, status, publicErr, internalErr string, at time.Time) error {
	if id == "" {
		return fmt.Errorf("invalid_request: missing job id")
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	_, err := s.db.Model("sync_job").Ctx(s.ctx()).Where("id", id).Data(do.SyncJob{
		Status:        status,
		ErrorPublic:   publicErr,
		ErrorInternal: scrubSecret(internalErr),
		FinishedAt:    at.Unix(),
	}).Update()
	return err
}

func hashToken(session string) string {
	sum := sha256.Sum256([]byte(session))
	return hex.EncodeToString(sum[:])
}

func MustKEK(raw string) (string, error) {
	if raw != "" {
		return raw, nil
	}
	return "", fmt.Errorf("FENGHUOLUN_TOKEN_KEK is required")
}

type BindingRow struct {
	ID         string        `json:"id"`
	VinMasked  string        `json:"vinMasked"`
	ModelCode  string        `json:"modelCode"`
	ModelName  string        `json:"modelName"`
	Lng        *float64      `json:"lng"`
	Lat        *float64      `json:"lat"`
	SyncStatus string        `json:"syncStatus"`
	SyncError  string        `json:"syncError"`
	SyncedAt   clock.Instant `json:"syncedAt"`
	Disabled   bool          `json:"disabled"`
}

func (s *SQLite) ListBindings() ([]BindingRow, error) {
	ctx := s.ctx()
	var bindings []entity.Binding
	if err := s.db.Model("binding").Ctx(ctx).OrderDesc("updated_at").Scan(&bindings); err != nil {
		return nil, err
	}
	var jobs []entity.SyncJob
	_ = s.db.Model("sync_job").Ctx(ctx).Scan(&jobs)
	byID := map[string]entity.SyncJob{}
	for _, j := range jobs {
		prev, ok := byID[j.BindingId]
		if !ok || j.StartedAt > prev.StartedAt || (j.StartedAt == prev.StartedAt && j.CreatedAt > prev.CreatedAt) {
			byID[j.BindingId] = j
		}
	}
	out := make([]BindingRow, 0, len(bindings))
	for _, row := range bindings {
		r := BindingRow{
			ID:        row.Id,
			VinMasked: row.VinMasked,
			ModelCode: row.ModelCode,
			ModelName: row.ModelName,
			Lng:       row.Lng,
			Lat:       row.Lat,
			Disabled:  row.Disabled == 1,
		}
		if j, ok := byID[row.Id]; ok {
			r.SyncStatus = j.Status
			r.SyncError = j.ErrorPublic
			if unix := jobFinishedUnix(j); unix > 0 {
				r.SyncedAt = clock.Of(time.Unix(unix, 0).UTC())
			}
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *SQLite) EnsureAdmin(username, password string) error {
	if username == "" || password == "" {
		return nil
	}
	ctx := s.ctx()
	n, err := s.db.Model("admin_user").Ctx(ctx).Count()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.db.Model("admin_user").Ctx(ctx).Data(do.AdminUser{
		Username:     username,
		PasswordHash: string(hash),
	}).Insert()
	return err
}

func (s *SQLite) AdminLogin(username, password string) (string, error) {
	ctx := s.ctx()
	v, err := s.db.Model("admin_user").Ctx(ctx).Where("username", username).Value("password_hash")
	if err != nil || v.IsEmpty() {
		return "", fmt.Errorf("unauthorized")
	}
	hash := v.String()
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return "", fmt.Errorf("unauthorized")
	}
	tok := NewSessionID()
	if _, err := s.db.Model("admin_session").Ctx(ctx).Data(do.AdminSession{
		TokenHash: hashToken(tok),
		Username:  username,
	}).Insert(); err != nil {
		return "", err
	}
	return tok, nil
}

func (s *SQLite) AdminOK(session string) bool {
	if session == "" {
		return false
	}
	n, err := s.db.Model("admin_session").Ctx(s.ctx()).Where("token_hash", hashToken(session)).Count()
	return err == nil && n > 0
}
