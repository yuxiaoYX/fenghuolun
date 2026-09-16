package owner

import (
	"context"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gogf/gf/v2/frame/g"

	v1 "fenghuolun/api/owner/v1"
	"fenghuolun/internal/coded"
	"fenghuolun/internal/middleware"
	"fenghuolun/internal/neta"
	"fenghuolun/internal/store"
)

func (c *ControllerV1) Bind(ctx context.Context, req *v1.BindReq) (res *v1.BindRes, err error) {
	b, err := c.Svc.Bind(strings.TrimSpace(req.RefreshToken))
	if err != nil {
		return nil, err
	}
	return &v1.BindRes{
		Session: b.Session,
		Vehicle: vehicleMap(b),
	}, nil
}

func (c *ControllerV1) Rebind(ctx context.Context, req *v1.RebindReq) (res *v1.BindRes, err error) {
	b, err := c.Svc.Rebind(sessionOf(ctx), strings.TrimSpace(req.RefreshToken))
	if err != nil {
		return nil, err
	}
	return &v1.BindRes{Session: b.Session, Vehicle: vehicleMap(b)}, nil
}

func (c *ControllerV1) Unbind(ctx context.Context, req *v1.UnbindReq) (res *v1.UnbindRes, err error) {
	c.Svc.Unbind(sessionOf(ctx))
	return &v1.UnbindRes{Unbound: true}, nil
}

func (c *ControllerV1) Vehicle(ctx context.Context, req *v1.VehicleReq) (res *v1.VehicleRes, err error) {
	b, err := c.must(ctx)
	if err != nil {
		return nil, err
	}
	v := vehicleMap(b)
	return &v1.VehicleRes{
		Nickname:   v["nickname"].(string),
		ModelCode:  v["modelCode"].(string),
		ModelName:  v["modelName"].(string),
		Trim:       v["trim"].(string),
		VinMasked:  v["vinMasked"].(string),
		IsExtender: v["isExtender"].(bool),
	}, nil
}

func (c *ControllerV1) VehiclePut(ctx context.Context, req *v1.VehiclePutReq) (res *v1.VehicleRes, err error) {
	b, err := c.must(ctx)
	if err != nil {
		return nil, err
	}
	nick := strings.TrimSpace(req.Nickname)
	if utf8.RuneCountInString(nick) > 32 {
		return nil, coded.New(http.StatusBadRequest, "invalid_request", "备注名最多 32 字")
	}
	if err := c.Svc.Store.SetNickname(b.ID, nick); err != nil {
		return nil, err
	}
	nb := c.Svc.Get(sessionOf(ctx))
	if nb == nil {
		return nil, coded.New(http.StatusUnauthorized, "unauthorized", "请先绑定")
	}
	v := vehicleMap(nb)
	return &v1.VehicleRes{
		Nickname:   v["nickname"].(string),
		ModelCode:  v["modelCode"].(string),
		ModelName:  v["modelName"].(string),
		Trim:       v["trim"].(string),
		VinMasked:  v["vinMasked"].(string),
		IsExtender: v["isExtender"].(bool),
	}, nil
}

func (c *ControllerV1) Snapshots(ctx context.Context, req *v1.SnapshotsReq) (res *v1.SnapshotsRes, err error) {
	b, err := c.must(ctx)
	if err != nil {
		return nil, err
	}
	items, total, err := c.Svc.Store.ListSnapshotSummaries(b.ID, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}
	return &v1.SnapshotsRes{Items: items, Total: total}, nil
}

func (c *ControllerV1) Snapshot(ctx context.Context, req *v1.SnapshotReq) (res *v1.SnapshotRes, err error) {
	b, err := c.must(ctx)
	if err != nil {
		return nil, err
	}
	if len(b.Snapshots) == 0 {
		return nil, coded.New(http.StatusNotFound, "decode", "还没有快照")
	}
	snap := b.Snapshots[len(b.Snapshots)-1]
	return &v1.SnapshotRes{Snapshot: snap, Stale: snap.StaleSince(time.Now(), c.Svc.StaleAfter())}, nil
}

func (c *ControllerV1) Energy(ctx context.Context, req *v1.EnergyReq) (res *v1.EnergyRes, err error) {
	b, err := c.must(ctx)
	if err != nil {
		return nil, err
	}
	fills, err := c.Svc.Store.ListFills(b.ID)
	if err != nil {
		return nil, err
	}
	return &v1.EnergyRes{
		Official: b.Energy,
		Fuel:     neta.FuelLedger(b.Snapshots),
		Fills:    fills,
		Spend:    neta.FillSpendOf(fills, b.Energy.TotalKwh),
		Capacity: neta.PackCapacityOf(fills),
		Note:     "电来自官方 kWh；油来自快照油量差分。花费只计充电/加油实付。自动充电草稿需要已解码的插枪或充电状态。",
	}, nil
}

func (c *ControllerV1) Fills(ctx context.Context, req *v1.FillsReq) (res *v1.FillsRes, err error) {
	b, err := c.must(ctx)
	if err != nil {
		return nil, err
	}
	fills, err := c.Svc.Store.ListFills(b.ID)
	if err != nil {
		return nil, err
	}
	return &v1.FillsRes{
		Items:    fills,
		Spend:    neta.FillSpendOf(fills, b.Energy.TotalKwh),
		Capacity: neta.PackCapacityOf(fills),
	}, nil
}

func (c *ControllerV1) FillGet(ctx context.Context, req *v1.FillGetReq) (res *v1.FillRes, err error) {
	b, err := c.must(ctx)
	if err != nil {
		return nil, err
	}
	f := c.Svc.Store.GetFill(b.ID, req.ID)
	if f == nil {
		return nil, coded.New(http.StatusNotFound, "not_found", "没有这条充能记录")
	}
	return &v1.FillRes{Fill: *f}, nil
}

func (c *ControllerV1) FillPost(ctx context.Context, req *v1.FillPostReq) (res *v1.FillRes, err error) {
	b, err := c.must(ctx)
	if err != nil {
		return nil, err
	}
	in := req.Fill
	in.ID = ""
	in.Source = neta.FillManual
	saved, err := c.Svc.Store.SaveFill(b.ID, in)
	if err != nil {
		return nil, coded.New(http.StatusBadRequest, "invalid_request", "充能记录数字不在合理范围")
	}
	return &v1.FillRes{Fill: saved}, nil
}

func (c *ControllerV1) FillPut(ctx context.Context, req *v1.FillPutReq) (res *v1.FillRes, err error) {
	b, err := c.must(ctx)
	if err != nil {
		return nil, err
	}
	in := req.Fill
	in.ID = req.ID
	saved, err := c.Svc.Store.SaveFill(b.ID, in)
	if err != nil {
		if err.Error() == "not_found" {
			return nil, coded.New(http.StatusNotFound, "not_found", "没有这条充能记录")
		}
		return nil, coded.New(http.StatusBadRequest, "invalid_request", "充能记录数字不在合理范围")
	}
	return &v1.FillRes{Fill: saved}, nil
}

func (c *ControllerV1) FillDelete(ctx context.Context, req *v1.FillDeleteReq) (res *v1.FillDeleteRes, err error) {
	b, err := c.must(ctx)
	if err != nil {
		return nil, err
	}
	if err := c.Svc.Store.DeleteFill(b.ID, req.ID); err != nil {
		return nil, coded.New(http.StatusNotFound, "not_found", "没有这条充能记录")
	}
	return &v1.FillDeleteRes{Deleted: true}, nil
}

func (c *ControllerV1) Sync(ctx context.Context, req *v1.SyncReq) (res *v1.SyncRes, err error) {
	b, err := c.Svc.Sync(sessionOf(ctx))
	if err != nil {
		return nil, err
	}
	return syncRes(b), nil
}

func (c *ControllerV1) SyncLatest(ctx context.Context, req *v1.SyncLatestReq) (res *v1.SyncRes, err error) {
	b, err := c.must(ctx)
	if err != nil {
		return nil, err
	}
	return syncRes(b), nil
}

func (c *ControllerV1) must(ctx context.Context) (*store.Binding, error) {
	b := c.Svc.Get(sessionOf(ctx))
	if b == nil {
		return nil, coded.New(http.StatusUnauthorized, "unauthorized", "请先绑定")
	}
	return b, nil
}

func sessionOf(ctx context.Context) string {
	return middleware.Bearer(g.RequestFromCtx(ctx))
}

func vehicleMap(b *store.Binding) map[string]any {
	return map[string]any{
		"nickname":   b.Meta.Nickname,
		"modelCode":  b.Meta.ModelCode,
		"modelName":  b.Meta.ModelName,
		"trim":       b.Meta.Trim,
		"vinMasked":  b.Meta.VinMasked,
		"isExtender": b.Meta.IsExtender,
	}
}

func syncRes(b *store.Binding) *v1.SyncRes {
	return &v1.SyncRes{Status: b.SyncStatus, Error: b.SyncError, SyncedAt: b.SyncedAt}
}
