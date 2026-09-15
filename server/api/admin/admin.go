package admin

import (
	"context"

	v1 "fenghuolun/api/admin/v1"
)

type IAdminV1 interface {
	Login(ctx context.Context, req *v1.LoginReq) (res *v1.LoginRes, err error)
	Bindings(ctx context.Context, req *v1.BindingsReq) (res *v1.BindingsRes, err error)
	DisableBinding(ctx context.Context, req *v1.DisableBindingReq) (res *v1.DisableBindingRes, err error)
}
