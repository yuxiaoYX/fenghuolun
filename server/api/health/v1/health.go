package v1

import "github.com/gogf/gf/v2/frame/g"

type HealthzReq struct {
	g.Meta `path:"/healthz" method:"get" tags:"Health" summary:"存活检查"`
}

type HealthzRes struct {
	Status string `json:"status"`
	Phase  string `json:"phase"`
}
