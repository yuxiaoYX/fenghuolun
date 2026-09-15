package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
  binding_id TEXT PRIMARY KEY,
  status TEXT NOT NULL,
  error_public TEXT NOT NULL DEFAULT '',
  synced_at INTEGER NOT NULL,
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
	} {
		_, _ = s.db.Exec(ctx, stmt)
	}
	_, _ = s.db.Exec(ctx, `UPDATE binding SET created_at = updated_at WHERE created_at IS NULL OR created_at = 0`)
	_, _ = s.db.Exec(ctx, `UPDATE binding SET deleted_at = 0 WHERE deleted_at IS NULL`)
	_, _ = s.db.Exec(ctx, `UPDATE owner_session SET updated_at = created_at WHERE updated_at IS NULL OR updated_at = 0`)
	_, _ = s.db.Exec(ctx, `UPDATE owner_session SET deleted_at = 0 WHERE deleted_at IS NULL`)
	for _, table := range []string{"snapshot", "energy", "sync_job", "admin_user", "admin_session"} {
		_, _ = s.db.Exec(ctx, `UPDATE `+table+` SET deleted_at = 0 WHERE deleted_at IS NULL`)
	}
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
	row := do.Binding{
		Id:            b.ID,
		VinMasked:     b.Meta.VinMasked,
		VinCipher:     vinC,
		ModelCode:     b.Meta.ModelCode,
		ModelName:     b.Meta.ModelName,
		Nickname:      b.Meta.Nickname,
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
	return s.db.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
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
		_, err := tx.Model("sync_job").Ctx(ctx).Data(do.SyncJob{
			BindingId:   b.ID,
			Status:      b.SyncStatus,
			ErrorPublic: b.SyncError,
			SyncedAt:    b.SyncedAt.Unix(),
		}).OnConflict("binding_id").Save()
		return err
	})
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
	if err := s.db.Model("sync_job").Ctx(ctx).Where("binding_id", id).Scan(&job); err == nil && job.BindingId != "" {
		b.SyncStatus = job.Status
		b.SyncError = job.ErrorPublic
		if job.SyncedAt > 0 {
			b.SyncedAt = clock.Of(time.Unix(job.SyncedAt, 0).UTC())
		}
	}
	return b
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

func (s *SQLite) MarkSync(id, status, publicErr string, at time.Time) error {
	if id == "" {
		return fmt.Errorf("invalid_request: missing binding id")
	}
	_, err := s.db.Model("sync_job").Ctx(s.ctx()).Data(do.SyncJob{
		BindingId:   id,
		Status:      status,
		ErrorPublic: publicErr,
		SyncedAt:    at.Unix(),
	}).OnConflict("binding_id").Save()
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
		byID[j.BindingId] = j
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
			if j.SyncedAt > 0 {
				r.SyncedAt = clock.Of(time.Unix(j.SyncedAt, 0).UTC())
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
