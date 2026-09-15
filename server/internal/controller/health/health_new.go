package health

import (
	"fenghuolun/api/health"
	"fenghuolun/internal/config"
)

type ControllerV1 struct {
	Cfg config.Config
}

func NewV1(cfg config.Config) health.IHealthV1 {
	return &ControllerV1{Cfg: cfg}
}
