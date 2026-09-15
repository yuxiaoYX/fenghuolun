package owner

import (
	"context"

	v1 "fenghuolun/api/owner/v1"
)

type IOwnerV1 interface {
	Bind(ctx context.Context, req *v1.BindReq) (res *v1.BindRes, err error)
	Rebind(ctx context.Context, req *v1.RebindReq) (res *v1.BindRes, err error)
	Unbind(ctx context.Context, req *v1.UnbindReq) (res *v1.UnbindRes, err error)
	Vehicle(ctx context.Context, req *v1.VehicleReq) (res *v1.VehicleRes, err error)
	VehiclePut(ctx context.Context, req *v1.VehiclePutReq) (res *v1.VehicleRes, err error)
	Snapshot(ctx context.Context, req *v1.SnapshotReq) (res *v1.SnapshotRes, err error)
	Snapshots(ctx context.Context, req *v1.SnapshotsReq) (res *v1.SnapshotsRes, err error)
	Energy(ctx context.Context, req *v1.EnergyReq) (res *v1.EnergyRes, err error)
	Sync(ctx context.Context, req *v1.SyncReq) (res *v1.SyncRes, err error)
	SyncLatest(ctx context.Context, req *v1.SyncLatestReq) (res *v1.SyncRes, err error)
}
