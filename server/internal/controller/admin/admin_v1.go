package admin

import (
	"context"
	"net/http"
	"strings"

	v1 "fenghuolun/api/admin/v1"
	"fenghuolun/internal/coded"
)

func (c *ControllerV1) Login(ctx context.Context, req *v1.LoginReq) (res *v1.LoginRes, err error) {
	tok, err := c.Store.AdminLogin(strings.TrimSpace(req.Username), req.Password)
	if err != nil {
		return nil, coded.New(http.StatusUnauthorized, "unauthorized", "用户名或密码不对")
	}
	return &v1.LoginRes{Session: tok}, nil
}

func (c *ControllerV1) Bindings(ctx context.Context, req *v1.BindingsReq) (res *v1.BindingsRes, err error) {
	items, err := c.Store.ListBindings()
	if err != nil {
		return nil, err
	}
	return &v1.BindingsRes{Items: items}, nil
}

func (c *ControllerV1) DisableBinding(ctx context.Context, req *v1.DisableBindingReq) (res *v1.DisableBindingRes, err error) {
	if err := c.Svc.Disable(strings.TrimSpace(req.ID)); err != nil {
		return nil, err
	}
	return &v1.DisableBindingRes{Disabled: true}, nil
}
