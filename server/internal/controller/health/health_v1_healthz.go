package health

import (
	"context"

	v1 "fenghuolun/api/health/v1"
	"fenghuolun/internal/version"
)

func (c *ControllerV1) Healthz(ctx context.Context, req *v1.HealthzReq) (res *v1.HealthzRes, err error) {
	return &v1.HealthzRes{Status: "ok", Phase: "2", Version: version.Display()}, nil
}
