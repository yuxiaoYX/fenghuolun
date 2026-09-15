package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	"fenghuolun/internal/store"
)

type LoginReq struct {
	g.Meta   `path:"/login" method:"post" tags:"Admin" summary:"管理员登录"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRes struct {
	Session string `json:"session"`
}

type BindingsReq struct {
	g.Meta `path:"/bindings" method:"get" tags:"Admin"`
}

type BindingsRes struct {
	Items []store.BindingRow `json:"items"`
}

type DisableBindingReq struct {
	g.Meta `path:"/bindings/{id}/disable" method:"post" tags:"Admin" summary:"停用绑定，只读运维，不下发车控"`
	ID     string `json:"id" in:"path" v:"required"`
}

type DisableBindingRes struct {
	Disabled bool `json:"disabled"`
}
