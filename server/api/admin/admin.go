package admin

import (
	"context"

	v1 "fenghuolun/api/admin/v1"
)

type IAdminV1 interface {
	Login(ctx context.Context, req *v1.LoginReq) (res *v1.LoginRes, err error)
	Logout(ctx context.Context, req *v1.LogoutReq) (res *v1.LogoutRes, err error)
	Account(ctx context.Context, req *v1.AccountReq) (res *v1.AccountRes, err error)
	Password(ctx context.Context, req *v1.PasswordReq) (res *v1.PasswordRes, err error)
	Health(ctx context.Context, req *v1.HealthReq) (res *v1.HealthRes, err error)
	Bindings(ctx context.Context, req *v1.BindingsReq) (res *v1.BindingsRes, err error)
	BindingGet(ctx context.Context, req *v1.BindingGetReq) (res *v1.BindingGetRes, err error)
	DisableBinding(ctx context.Context, req *v1.DisableBindingReq) (res *v1.DisableBindingRes, err error)
	EnableBinding(ctx context.Context, req *v1.EnableBindingReq) (res *v1.EnableBindingRes, err error)
	SyncBinding(ctx context.Context, req *v1.SyncBindingReq) (res *v1.SyncBindingRes, err error)
	KickBinding(ctx context.Context, req *v1.KickBindingReq) (res *v1.KickBindingRes, err error)
	Jobs(ctx context.Context, req *v1.JobsReq) (res *v1.JobsRes, err error)
	Tables(ctx context.Context, req *v1.TablesReq) (res *v1.TablesRes, err error)
	TableRows(ctx context.Context, req *v1.TableRowsReq) (res *v1.TableRowsRes, err error)
	SettingsGet(ctx context.Context, req *v1.SettingsGetReq) (res *v1.SettingsGetRes, err error)
	SettingsPut(ctx context.Context, req *v1.SettingsPutReq) (res *v1.SettingsPutRes, err error)
	System(ctx context.Context, req *v1.SystemReq) (res *v1.SystemRes, err error)
	SystemUpdate(ctx context.Context, req *v1.SystemUpdateReq) (res *v1.SystemUpdateRes, err error)
}
