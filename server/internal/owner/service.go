package owner

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"fenghuolun/internal/clock"
	"fenghuolun/internal/coded"
	"fenghuolun/internal/config"
	"fenghuolun/internal/neta"
	"fenghuolun/internal/store"
)

type Upstream interface {
	Refresh(refreshToken string) (neta.TokenPair, error)
	GetCurrentVehicle(accessToken string) ([]byte, error)
	GetAppVehicleData(accessToken, vin string) ([]byte, error)
	QueryEnergyByVin(accessToken, vin string, periodType int) ([]byte, error)
	SendLoginCode(phone string) error
	LoginBySMS(phone, code string) (neta.TokenPair, error)
}

type Service struct {
	Cfg    config.Config
	Store  *store.SQLite
	Client Upstream
	limits attemptBook
}

func New(cfg config.Config, st *store.SQLite) *Service {
	client := neta.NewClient()
	client.AppKey = cfg.NetaAppKey
	client.AppSecret = cfg.NetaAppSecret
	return &Service{
		Cfg:    cfg,
		Store:  st,
		Client: client,
	}
}

func (s *Service) Bind(refreshToken string) (*store.Binding, error) {
	if len(refreshToken) < 8 {
		return nil, fmt.Errorf("invalid_request: refresh_token too short")
	}
	pair, err := s.Client.Refresh(refreshToken)
	if err != nil {
		return nil, err
	}
	b, err := s.pullLive(pair)
	if err != nil {
		return nil, err
	}
	b.ID = store.NewSessionID()
	b.Session = store.NewSessionID()
	b.RefreshHint = store.HintToken(refreshToken)
	if err := s.Store.Put(b); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *Service) Rebind(session, refreshToken string) (*store.Binding, error) {
	cur := s.Store.Get(session)
	if cur == nil {
		return nil, fmt.Errorf("unauthorized")
	}
	accountID, err := s.Store.AccountBySession(session)
	if err != nil {
		return nil, err
	}
	pair, err := s.Client.Refresh(refreshToken)
	if err != nil {
		return nil, err
	}
	live, err := s.pullLive(pair)
	if err != nil {
		return nil, err
	}
	b, err := s.storeOfficial(pair, accountID, cur.ID, live)
	if err != nil {
		return nil, err
	}
	_ = s.Store.RevokeSession(session)
	return b, nil
}

func (s *Service) Unbind(session string) {
	accountID, _ := s.Store.AccountBySession(session)
	b := s.Store.Get(session)
	s.Store.Delete(session)
	if b != nil {
		_ = s.Store.ClearAccountBinding(accountID, b.ID)
	}
}

func (s *Service) Logout(session string) error {
	return s.Store.RevokeSession(session)
}

func (s *Service) Get(session string) *store.Binding {
	return s.Store.Get(session)
}

func (s *Service) Sync(session string) (*store.Binding, error) {
	b := s.Store.Get(session)
	if b == nil {
		return nil, fmt.Errorf("unauthorized")
	}
	return s.syncBinding(b, "manual")
}

func (s *Service) SyncByID(id, kind string) (*store.Binding, error) {
	b := s.Store.GetByID(id)
	if b == nil {
		return nil, fmt.Errorf("not_found")
	}
	return s.syncBinding(b, kind)
}

func (s *Service) SyncAllActive() {
	ids, err := s.Store.ListActiveIDs()
	if err != nil {
		return
	}
	for _, id := range ids {
		_, _ = s.SyncByID(id, "cron")
	}
}

func (s *Service) Disable(id string) error {
	return s.Store.SetDisabled(id, true)
}

func (s *Service) Enable(id string) error {
	return s.Store.SetDisabled(id, false)
}

func (s *Service) Kick(id string) error {
	return s.Store.KickSessions(id)
}

func (s *Service) StaleAfter() time.Duration {
	if s == nil || s.Store == nil {
		return neta.StaleAfter
	}
	return s.Store.StaleAfter()
}

func (s *Service) syncBinding(b *store.Binding, kind string) (*store.Binding, error) {
	if kind == "" {
		kind = "manual"
	}
	if b.Disabled {
		return nil, fmt.Errorf("not_found")
	}
	jobID, err := s.Store.BeginJob(b.ID, kind)
	if err != nil {
		return nil, err
	}
	fail := func(status, public string, cause error) {
		internal := ""
		if cause != nil {
			internal = cause.Error()
		}
		_ = s.Store.FinishJob(jobID, status, public, internal, time.Now().UTC())
	}
	rt := b.RefreshToken
	if rt == "" {
		fail("auth_failed", "请重新填写 refresh_token", neta.ErrTokenInvalid)
		return b, neta.ErrTokenInvalid
	}
	pair, err := s.Client.Refresh(rt)
	if err != nil {
		status, public := classifySync(err)
		fail(status, public, err)
		return b, err
	}
	nb, err := s.pullLive(pair)
	if err != nil {
		status, public := classifySync(err)
		fail(status, public, err)
		return b, err
	}
	nb.ID = b.ID
	nb.Session = b.Session
	nb.RefreshHint = b.RefreshHint
	nb.SyncKind = kind
	nb.JobID = jobID
	if err := s.Store.Put(nb); err != nil {
		fail("upstream", "落库失败", err)
		return nil, err
	}
	if b.Session != "" {
		return s.Store.Get(b.Session), nil
	}
	return s.Store.GetByID(b.ID), nil
}

func classifySync(err error) (status, public string) {
	switch {
	case errors.Is(err, neta.ErrTokenInvalid):
		return "auth_failed", "请重新填写 refresh_token"
	case errors.Is(err, neta.ErrDecode):
		return "decode", "官方响应结构变了"
	default:
		return "upstream", "官方云暂不可用"
	}
}

func (s *Service) pullLive(pair neta.TokenPair) (*store.Binding, error) {
	now := time.Now().UTC()
	curRaw, err := s.Client.GetCurrentVehicle(pair.AccessToken)
	if err != nil {
		return nil, err
	}
	meta, err := neta.DecodeCurrentVehicle(curRaw)
	if err != nil {
		return nil, err
	}
	b := &store.Binding{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		Meta:         meta,
		SyncStatus:   "ok",
		SyncedAt:     clock.Of(now),
	}
	if meta.VIN != "" {
		dataRaw, err := s.Client.GetAppVehicleData(pair.AccessToken, meta.VIN)
		if err != nil {
			return nil, err
		}
		snap, err := neta.DecodeVehicleData(dataRaw, meta, now, s.Cfg.ScaleCandidate)
		if err != nil {
			return nil, err
		}
		b.Snapshots = []neta.Snapshot{snap}
		energyRaw, err := s.Client.QueryEnergyByVin(pair.AccessToken, meta.VIN, 1)
		if err == nil {
			if energy, e2 := neta.DecodeEnergyDays(energyRaw, 1, now); e2 == nil {
				b.Energy = energy
			}
		}
	}
	return b, nil
}

func (s *Service) storeOfficial(pair neta.TokenPair, accountID, preferredID string, b *store.Binding) (*store.Binding, error) {
	if preferredID != "" {
		b.ID = preferredID
	} else if accountID != "" {
		if acc, err := s.Store.AccountByID(accountID); err == nil && acc != nil && acc.BindingId != "" {
			if cur := s.Store.GetByIDAny(acc.BindingId); cur != nil && !cur.Disabled {
				b.ID = cur.ID
			}
		}
		if b.ID == "" {
			if cur := s.Store.FindBindingByVIN(b.Meta.VIN); cur != nil && !cur.Disabled {
				b.ID = cur.ID
			}
		}
	}
	if b.ID == "" {
		b.ID = store.NewSessionID()
	}
	b.Session = store.NewSessionID()
	b.RefreshHint = store.HintToken(pair.RefreshToken)
	if err := s.Store.Put(b); err != nil {
		return nil, err
	}
	if err := s.attachAccount(accountID, b); err != nil {
		return nil, err
	}
	got := s.Store.Get(b.Session)
	if got == nil {
		return nil, coded.New(http.StatusBadGateway, "upstream", "绑定已写入但会话没有建立")
	}
	return got, nil
}
