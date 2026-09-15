package owner

import (
	api "fenghuolun/api/owner"
	"fenghuolun/internal/config"
	ownersvc "fenghuolun/internal/owner"
)

type ControllerV1 struct {
	Svc *ownersvc.Service
	Cfg config.Config
}

func NewV1(svc *ownersvc.Service, cfg config.Config) api.IOwnerV1 {
	return &ControllerV1{Svc: svc, Cfg: cfg}
}
