package health

import (
	"context"

	v1 "fenghuolun/api/health/v1"
)

type IHealthV1 interface {
	Healthz(ctx context.Context, req *v1.HealthzReq) (res *v1.HealthzRes, err error)
}
