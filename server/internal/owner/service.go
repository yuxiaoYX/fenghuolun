package owner

import (
	"fmt"
	"time"

	"fenghuolun/internal/clock"
	"fenghuolun/internal/config"
	"fenghuolun/internal/neta"
	"fenghuolun/internal/store"
)

type Upstream interface {
	Refresh(refreshToken string) (neta.TokenPair, error)
	GetCurrentVehicle(accessToken string) ([]byte, error)
	GetAppVehicleData(accessToken, vin string) ([]byte, error)
	QueryEnergyByVin(accessToken, vin string, periodType int) ([]byte, error)
}

type Service struct {
	Cfg    config.Config
	Store  *store.SQLite
	Client Upstream
}

func New(cfg config.Config, st *store.SQLite) *Service {
	return &Service{
		Cfg:    cfg,
		Store:  st,
		Client: neta.NewClient(),
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
	s.Store.Delete(session)
	return s.Bind(refreshToken)
}

func (s *Service) Unbind(session string) {
	s.Store.Delete(session)
}

func (s *Service) Get(session string) *store.Binding {
	return s.Store.Get(session)
}

func (s *Service) Sync(session string) (*store.Binding, error) {
	b := s.Store.Get(session)
	if b == nil {
		return nil, fmt.Errorf("unauthorized")
	}
	return s.syncBinding(b)
}

func (s *Service) SyncByID(id string) (*store.Binding, error) {
	b := s.Store.GetByID(id)
	if b == nil {
		return nil, fmt.Errorf("not_found")
	}
	return s.syncBinding(b)
}

func (s *Service) SyncAllActive() {
	ids, err := s.Store.ListActiveIDs()
	if err != nil {
		return
	}
	for _, id := range ids {
		_, _ = s.SyncByID(id)
	}
}

func (s *Service) Disable(id string) error {
	return s.Store.SetDisabled(id, true)
}

func (s *Service) syncBinding(b *store.Binding) (*store.Binding, error) {
	if b.Disabled {
		return nil, fmt.Errorf("not_found")
	}
	now := time.Now().UTC()
	rt := b.RefreshToken
	if rt == "" {
		_ = s.Store.MarkSync(b.ID, "auth_failed", "请重新填写 refresh_token", now)
		return b, neta.ErrTokenInvalid
	}
	pair, err := s.Client.Refresh(rt)
	if err != nil {
		_ = s.Store.MarkSync(b.ID, "auth_failed", "请重新填写 refresh_token", now)
		return b, err
	}
	nb, err := s.pullLive(pair)
	if err != nil {
		_ = s.Store.MarkSync(b.ID, "upstream", "官方云暂不可用", now)
		return b, err
	}
	nb.ID = b.ID
	nb.Session = b.Session
	nb.RefreshHint = b.RefreshHint
	if err := s.Store.Put(nb); err != nil {
		return nil, err
	}
	if b.Session != "" {
		return s.Store.Get(b.Session), nil
	}
	return s.Store.GetByID(b.ID), nil
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
