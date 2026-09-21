package admin

import (
	api "fenghuolun/api/admin"
	"fenghuolun/internal/config"
	ownersvc "fenghuolun/internal/owner"
	"fenghuolun/internal/store"
)

type ControllerV1 struct {
	Store *store.SQLite
	Svc   *ownersvc.Service
	Cfg   config.Config
}

func NewV1(st *store.SQLite, svc *ownersvc.Service, cfg config.Config) api.IAdminV1 {
	return &ControllerV1{Store: st, Svc: svc, Cfg: cfg}
}
