package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	"fenghuolun/internal/clock"
	"fenghuolun/internal/neta"
	"fenghuolun/internal/store"
)

type BindReq struct {
	g.Meta       `path:"/bind" method:"post" tags:"Owner" summary:"绑定 refresh_token"`
	RefreshToken string `json:"refresh_token"`
}

type RebindReq struct {
	g.Meta       `path:"/rebind" method:"post" tags:"Owner"`
	RefreshToken string `json:"refresh_token"`
}

type UnbindReq struct {
	g.Meta `path:"/unbind" method:"post" tags:"Owner"`
}

type UnbindRes struct {
	Unbound bool `json:"unbound"`
}

type BindRes struct {
	Session string         `json:"session"`
	Vehicle map[string]any `json:"vehicle"`
}

type VehicleReq struct {
	g.Meta `path:"/vehicle" method:"get" tags:"Owner"`
}

type VehiclePutReq struct {
	g.Meta   `path:"/vehicle" method:"put" tags:"Owner" summary:"改车辆备注名"`
	Nickname string `json:"nickname"`
}

type VehicleRes struct {
	Nickname   string `json:"nickname"`
	ModelCode  string `json:"modelCode"`
	ModelName  string `json:"modelName"`
	Trim       string `json:"trim"`
	VinMasked  string `json:"vinMasked"`
	IsExtender bool   `json:"isExtender"`
}

type SnapshotReq struct {
	g.Meta `path:"/snapshot/latest" method:"get" tags:"Owner"`
}

type SnapshotsReq struct {
	g.Meta   `path:"/snapshots" method:"get" tags:"Owner"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

type SnapshotsRes struct {
	Items []store.SnapshotSummary `json:"items"`
	Total int                     `json:"total"`
}

type SnapshotRes struct {
	neta.Snapshot
	Stale bool `json:"stale"`
}

type EnergyReq struct {
	g.Meta `path:"/energy" method:"get" tags:"Owner"`
	Type   int `json:"type" dc:"官方 type，默认 1"`
}

type EnergyRes struct {
	Official neta.EnergyStat   `json:"official"`
	Fuel     []neta.FuelDelta  `json:"fuel"`
	Fills    []neta.Fill       `json:"fills"`
	Spend    neta.FillSpend    `json:"spend"`
	Capacity neta.PackCapacity `json:"capacity"`
	Note     string            `json:"note"`
}

type FillsReq struct {
	g.Meta `path:"/fills" method:"get" tags:"Owner"`
}

type FillsRes struct {
	Items    []neta.Fill       `json:"items"`
	Spend    neta.FillSpend    `json:"spend"`
	Capacity neta.PackCapacity `json:"capacity"`
}

type FillGetReq struct {
	g.Meta `path:"/fills/{id}" method:"get" tags:"Owner"`
	ID     string `p:"id"`
}

type FillPostReq struct {
	g.Meta `path:"/fills" method:"post" tags:"Owner"`
	neta.Fill
}

type FillPutReq struct {
	g.Meta `path:"/fills/{id}" method:"put" tags:"Owner"`
	ID     string `p:"id"`
	neta.Fill
}

type FillRes struct {
	neta.Fill
}

type FillDeleteReq struct {
	g.Meta `path:"/fills/{id}" method:"delete" tags:"Owner"`
	ID     string `p:"id"`
}

type FillDeleteRes struct {
	Deleted bool `json:"deleted"`
}

type SyncReq struct {
	g.Meta `path:"/sync" method:"post" tags:"Owner"`
}

type SyncLatestReq struct {
	g.Meta `path:"/sync/latest" method:"get" tags:"Owner"`
}

type SyncRes struct {
	Status   string        `json:"status"`
	Error    string        `json:"error"`
	SyncedAt clock.Instant `json:"syncedAt"`
}
