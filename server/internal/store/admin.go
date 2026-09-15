package store

import (
	"fmt"
	"strings"
	"time"

	"fenghuolun/internal/clock"
	"fenghuolun/internal/model/do"
	"fenghuolun/internal/model/entity"
	"fenghuolun/internal/neta"

	"golang.org/x/crypto/bcrypt"
)

type JobRow struct {
	ID            string        `json:"id"`
	BindingId     string        `json:"bindingId"`
	VinMasked     string        `json:"vinMasked"`
	Kind          string        `json:"kind"`
	Status        string        `json:"status"`
	ErrorPublic   string        `json:"errorPublic"`
	ErrorInternal string        `json:"errorInternal"`
	StartedAt     clock.Instant `json:"startedAt"`
	FinishedAt    clock.Instant `json:"finishedAt"`
}

type BindingDetail struct {
	BindingRow
	Nickname      string          `json:"nickname"`
	Trim          string          `json:"trim"`
	IsExtender    bool            `json:"isExtender"`
	RefreshHint   string          `json:"refreshHint"`
	HasRefresh    bool            `json:"hasRefresh"`
	HasAccess     bool            `json:"hasAccess"`
	SessionCount  int             `json:"sessionCount"`
	LocReportedAt *clock.Instant  `json:"locReportedAt"`
	Snapshot      *neta.Snapshot  `json:"snapshot"`
	Stale         bool            `json:"stale"`
	Energy        neta.EnergyStat `json:"energy"`
	Jobs          []JobRow        `json:"jobs"`
}

type AdminHealth struct {
	BindingsTotal    int    `json:"bindingsTotal"`
	BindingsActive   int    `json:"bindingsActive"`
	BindingsDisabled int    `json:"bindingsDisabled"`
	TokenInvalid     int    `json:"tokenInvalid"`
	SyncOkToday      int    `json:"syncOkToday"`
	SyncFailToday    int    `json:"syncFailToday"`
	UpstreamStreak   int    `json:"upstreamStreak"`
	CronSync         string `json:"cronSync"`
}

type TablePage struct {
	Name   string           `json:"name"`
	Total  int              `json:"total"`
	Items  []map[string]any `json:"items"`
	Note   string           `json:"note"`
}

func (s *SQLite) BindingDetail(id string) (*BindingDetail, error) {
	b := s.GetByIDAny(id)
	if b == nil {
		return nil, fmt.Errorf("not_found")
	}
	ctx := s.ctx()
	var row entity.Binding
	if err := s.db.Model("binding").Ctx(ctx).Where("id", id).Scan(&row); err != nil || row.Id == "" {
		return nil, fmt.Errorf("not_found")
	}
	d := &BindingDetail{
		BindingRow: BindingRow{
			ID:         b.ID,
			VinMasked:  b.Meta.VinMasked,
			ModelCode:  b.Meta.ModelCode,
			ModelName:  b.Meta.ModelName,
			Lng:        row.Lng,
			Lat:        row.Lat,
			SyncStatus: b.SyncStatus,
			SyncError:  b.SyncError,
			SyncedAt:   b.SyncedAt,
			Disabled:   b.Disabled,
		},
		Nickname:     b.Meta.Nickname,
		Trim:         b.Meta.Trim,
		IsExtender:   b.Meta.IsExtender,
		RefreshHint:  b.RefreshHint,
		HasRefresh:   len(row.RefreshCipher) > 0,
		HasAccess:    len(row.AccessCipher) > 0,
		Energy:       b.Energy,
	}
	n, _ := s.db.Model("owner_session").Ctx(ctx).Where("binding_id", id).Count()
	d.SessionCount = n
	if row.LocReportedAt != nil && *row.LocReportedAt > 0 {
		at := clock.Of(time.Unix(*row.LocReportedAt, 0).UTC())
		d.LocReportedAt = &at
	}
	if len(b.Snapshots) > 0 {
		snap := b.Snapshots[len(b.Snapshots)-1]
		d.Snapshot = &snap
		d.Stale = snap.StaleSince(time.Now(), s.StaleAfter())
	}
	jobs, _, err := s.ListJobs(id, "", 1, 8)
	if err != nil {
		return nil, err
	}
	d.Jobs = jobs
	return d, nil
}

func (s *SQLite) KickSessions(bindingID string) error {
	if bindingID == "" {
		return fmt.Errorf("invalid_request: missing binding id")
	}
	if s.GetByIDAny(bindingID) == nil {
		return fmt.Errorf("not_found")
	}
	_, err := s.db.Model("owner_session").Ctx(s.ctx()).Where("binding_id", bindingID).Delete()
	return err
}

func (s *SQLite) ListJobs(bindingID, status string, page, pageSize int) ([]JobRow, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	ctx := s.ctx()
	m := s.db.Model("sync_job").Ctx(ctx)
	if bindingID != "" {
		m = m.Where("binding_id", bindingID)
	}
	if status != "" {
		m = m.Where("status", status)
	}
	total, err := m.Clone().Count()
	if err != nil {
		return nil, 0, err
	}
	var jobs []entity.SyncJob
	if err := m.OrderDesc("started_at").OrderDesc("created_at").Page(page, pageSize).Scan(&jobs); err != nil {
		return nil, 0, err
	}
	vin := map[string]string{}
	var bindings []entity.Binding
	_ = s.db.Model("binding").Ctx(ctx).Fields("id, vin_masked").Scan(&bindings)
	for _, b := range bindings {
		vin[b.Id] = b.VinMasked
	}
	out := make([]JobRow, 0, len(jobs))
	for _, j := range jobs {
		row := JobRow{
			ID:            j.Id,
			BindingId:     j.BindingId,
			VinMasked:     vin[j.BindingId],
			Kind:          j.Kind,
			Status:        j.Status,
			ErrorPublic:   j.ErrorPublic,
			ErrorInternal: scrubSecret(j.ErrorInternal),
		}
		if j.StartedAt > 0 {
			row.StartedAt = clock.Of(time.Unix(j.StartedAt, 0).UTC())
		}
		if unix := jobFinishedUnix(j); unix > 0 {
			row.FinishedAt = clock.Of(time.Unix(unix, 0).UTC())
		}
		out = append(out, row)
	}
	return out, total, nil
}

func (s *SQLite) AdminHealth() (AdminHealth, error) {
	ctx := s.ctx()
	h := AdminHealth{CronSync: s.GetSetting(SettingCronSync)}
	var bindings []entity.Binding
	if err := s.db.Model("binding").Ctx(ctx).Scan(&bindings); err != nil {
		return h, err
	}
	h.BindingsTotal = len(bindings)
	for _, b := range bindings {
		if b.Disabled == 1 {
			h.BindingsDisabled++
		} else {
			h.BindingsActive++
		}
	}
	var jobs []entity.SyncJob
	_ = s.db.Model("sync_job").Ctx(ctx).OrderDesc("started_at").OrderDesc("created_at").Scan(&jobs)
	latest := map[string]entity.SyncJob{}
	for _, j := range jobs {
		if _, ok := latest[j.BindingId]; !ok {
			latest[j.BindingId] = j
		}
	}
	for _, j := range latest {
		if j.Status == "auth_failed" {
			h.TokenInvalid++
		}
	}
	now := time.Now().In(clock.China)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, clock.China).Unix()
	for _, j := range jobs {
		if j.StartedAt < dayStart {
			continue
		}
		if j.Status == "ok" {
			h.SyncOkToday++
		} else {
			h.SyncFailToday++
		}
	}
	for _, j := range jobs {
		if j.Status == "upstream" {
			h.UpstreamStreak++
			continue
		}
		break
	}
	return h, nil
}

func (s *SQLite) AdminLogout(session string) {
	if session == "" {
		return
	}
	_, _ = s.db.Model("admin_session").Ctx(s.ctx()).Where("token_hash", hashToken(session)).Delete()
}

func (s *SQLite) AdminUsername(session string) string {
	if session == "" {
		return ""
	}
	v, err := s.db.Model("admin_session").Ctx(s.ctx()).Where("token_hash", hashToken(session)).Value("username")
	if err != nil {
		return ""
	}
	return v.String()
}

func (s *SQLite) AdminChangePassword(session, old, newPass string) error {
	if len(newPass) < 8 {
		return fmt.Errorf("invalid_request: password too short")
	}
	username := s.AdminUsername(session)
	if username == "" {
		return fmt.Errorf("unauthorized")
	}
	ctx := s.ctx()
	v, err := s.db.Model("admin_user").Ctx(ctx).Where("username", username).Value("password_hash")
	if err != nil || v.IsEmpty() {
		return fmt.Errorf("unauthorized")
	}
	if bcrypt.CompareHashAndPassword([]byte(v.String()), []byte(old)) != nil {
		return fmt.Errorf("unauthorized")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.db.Model("admin_user").Ctx(ctx).Where("username", username).Data(do.AdminUser{
		PasswordHash: string(hash),
	}).Update()
	if err != nil {
		return err
	}
	_, err = s.db.Model("admin_session").Ctx(ctx).Where("username", username).Where("token_hash <> ?", hashToken(session)).Delete()
	return err
}

func (s *SQLite) TableNames() []map[string]any {
	return []map[string]any{
		{"name": "binding", "label": "绑定"},
		{"name": "snapshot", "label": "快照"},
		{"name": "energy", "label": "能耗"},
		{"name": "sync_job", "label": "同步任务"},
		{"name": "owner_session", "label": "车主会话"},
		{"name": "admin_user", "label": "管理员"},
		{"name": "admin_session", "label": "管理员会话"},
		{"name": "app_settings", "label": "运行时设置"},
	}
}

func (s *SQLite) ListTable(name, bindingID string, page, pageSize int) (*TablePage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	ctx := s.ctx()
	m := s.db.Model(name).Ctx(ctx)
	switch name {
	case "binding":
		total, err := m.Count()
		if err != nil {
			return nil, err
		}
		var rows []entity.Binding
		if err := m.OrderDesc("updated_at").Page(page, pageSize).Scan(&rows); err != nil {
			return nil, err
		}
		items := make([]map[string]any, 0, len(rows))
		for _, r := range rows {
			items = append(items, map[string]any{
				"id": r.Id, "vin_masked": r.VinMasked, "model_code": r.ModelCode, "model_name": r.ModelName,
				"nickname": r.Nickname, "trim": r.Trim, "is_extender": r.IsExtender == 1, "refresh_hint": r.RefreshHint,
				"disabled": r.Disabled == 1, "lng": r.Lng, "lat": r.Lat, "loc_reported_at": wallTimePtr(r.LocReportedAt),
				"created_at": wallTime(r.CreatedAt), "updated_at": wallTime(r.UpdatedAt), "deleted_at": wallTime(r.DeletedAt),
				"has_vin_cipher": len(r.VinCipher) > 0, "has_refresh": len(r.RefreshCipher) > 0, "has_access": len(r.AccessCipher) > 0,
			})
		}
		return &TablePage{Name: name, Total: total, Items: items, Note: "密文列只显示有/无"}, nil
	case "snapshot":
		if bindingID != "" {
			m = m.Where("binding_id", bindingID)
		}
		total, err := m.Count()
		if err != nil {
			return nil, err
		}
		var rows []entity.Snapshot
		if err := m.OrderDesc("fetched_at").OrderDesc("id").Page(page, pageSize).Scan(&rows); err != nil {
			return nil, err
		}
		items := make([]map[string]any, 0, len(rows))
		for _, r := range rows {
			items = append(items, map[string]any{
				"id": r.Id, "binding_id": r.BindingId, "fetched_at": wallTime(r.FetchedAt), "payload_len": len(r.Payload),
				"created_at": wallTime(r.CreatedAt), "updated_at": wallTime(r.UpdatedAt), "deleted_at": wallTime(r.DeletedAt),
			})
		}
		return &TablePage{Name: name, Total: total, Items: items, Note: "不返回快照 JSON；详情页看解码字段"}, nil
	case "energy":
		if bindingID != "" {
			m = m.Where("binding_id", bindingID)
		}
		total, err := m.Count()
		if err != nil {
			return nil, err
		}
		var rows []entity.Energy
		if err := m.OrderDesc("fetched_at").Page(page, pageSize).Scan(&rows); err != nil {
			return nil, err
		}
		items := make([]map[string]any, 0, len(rows))
		for _, r := range rows {
			items = append(items, map[string]any{
				"binding_id": r.BindingId, "fetched_at": wallTime(r.FetchedAt), "payload_len": len(r.Payload),
				"created_at": wallTime(r.CreatedAt), "updated_at": wallTime(r.UpdatedAt), "deleted_at": wallTime(r.DeletedAt),
			})
		}
		return &TablePage{Name: name, Total: total, Items: items}, nil
	case "sync_job":
		if bindingID != "" {
			m = m.Where("binding_id", bindingID)
		}
		total, err := m.Count()
		if err != nil {
			return nil, err
		}
		var rows []entity.SyncJob
		if err := m.OrderDesc("started_at").OrderDesc("created_at").Page(page, pageSize).Scan(&rows); err != nil {
			return nil, err
		}
		items := make([]map[string]any, 0, len(rows))
		for _, r := range rows {
			items = append(items, map[string]any{
				"id": r.Id, "binding_id": r.BindingId, "kind": r.Kind, "status": r.Status,
				"error_public": scrubSecret(r.ErrorPublic), "error_internal": scrubSecret(r.ErrorInternal),
				"started_at": wallTime(r.StartedAt), "finished_at": wallTime(r.FinishedAt),
				"created_at": wallTime(r.CreatedAt), "updated_at": wallTime(r.UpdatedAt), "deleted_at": wallTime(r.DeletedAt),
			})
		}
		return &TablePage{Name: name, Total: total, Items: items}, nil
	case "owner_session":
		if bindingID != "" {
			m = m.Where("binding_id", bindingID)
		}
		total, err := m.Count()
		if err != nil {
			return nil, err
		}
		var rows []entity.OwnerSession
		if err := m.OrderDesc("created_at").Page(page, pageSize).Scan(&rows); err != nil {
			return nil, err
		}
		items := make([]map[string]any, 0, len(rows))
		for _, r := range rows {
			items = append(items, map[string]any{
				"binding_id": r.BindingId, "created_at": wallTime(r.CreatedAt), "updated_at": wallTime(r.UpdatedAt), "deleted_at": wallTime(r.DeletedAt),
			})
		}
		return &TablePage{Name: name, Total: total, Items: items, Note: "不返回 token 哈希"}, nil
	case "admin_user":
		total, err := m.Count()
		if err != nil {
			return nil, err
		}
		var rows []entity.AdminUser
		if err := m.OrderDesc("created_at").Page(page, pageSize).Scan(&rows); err != nil {
			return nil, err
		}
		items := make([]map[string]any, 0, len(rows))
		for _, r := range rows {
			items = append(items, map[string]any{
				"username": r.Username, "created_at": wallTime(r.CreatedAt), "updated_at": wallTime(r.UpdatedAt), "deleted_at": wallTime(r.DeletedAt),
			})
		}
		return &TablePage{Name: name, Total: total, Items: items, Note: "不返回密码哈希"}, nil
	case "admin_session":
		total, err := m.Count()
		if err != nil {
			return nil, err
		}
		var rows []entity.AdminSession
		if err := m.OrderDesc("created_at").Page(page, pageSize).Scan(&rows); err != nil {
			return nil, err
		}
		items := make([]map[string]any, 0, len(rows))
		for _, r := range rows {
			items = append(items, map[string]any{
				"username": r.Username, "created_at": wallTime(r.CreatedAt), "updated_at": wallTime(r.UpdatedAt), "deleted_at": wallTime(r.DeletedAt),
			})
		}
		return &TablePage{Name: name, Total: total, Items: items, Note: "不返回 token 哈希"}, nil
	case "app_settings":
		total, err := m.Count()
		if err != nil {
			return nil, err
		}
		var rows []entity.AppSetting
		if err := m.OrderAsc("key").Page(page, pageSize).Scan(&rows); err != nil {
			return nil, err
		}
		items := make([]map[string]any, 0, len(rows))
		for _, r := range rows {
			items = append(items, map[string]any{
				"key": r.Key, "value": r.Value, "created_at": wallTime(r.CreatedAt), "updated_at": wallTime(r.UpdatedAt), "deleted_at": wallTime(r.DeletedAt),
			})
		}
		return &TablePage{Name: name, Total: total, Items: items}, nil
	default:
		return nil, fmt.Errorf("invalid_request: unknown table")
	}
}

func scrubSecret(s string) string {
	low := strings.ToLower(s)
	if strings.Contains(low, "bearer") || strings.Contains(low, "refresh") || strings.Contains(low, "access_token") || strings.Contains(low, "token") {
		return "[redacted]"
	}
	return s
}

func wallTime(unix int64) any {
	if unix <= 0 {
		return nil
	}
	return clock.Of(time.Unix(unix, 0).UTC())
}

func wallTimePtr(p *int64) any {
	if p == nil {
		return nil
	}
	return wallTime(*p)
}
