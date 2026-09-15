package admin

import (
	api "fenghuolun/api/admin"
	ownersvc "fenghuolun/internal/owner"
	"fenghuolun/internal/store"
)

type ControllerV1 struct {
	Store *store.SQLite
	Svc   *ownersvc.Service
}

func NewV1(st *store.SQLite, svc *ownersvc.Service) api.IAdminV1 {
	return &ControllerV1{Store: st, Svc: svc}
}
