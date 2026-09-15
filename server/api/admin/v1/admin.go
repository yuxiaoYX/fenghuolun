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

type LogoutReq struct {
	g.Meta `path:"/logout" method:"post" tags:"Admin"`
}

type LogoutRes struct {
	Ok bool `json:"ok"`
}

type AccountReq struct {
	g.Meta `path:"/account" method:"get" tags:"Admin"`
}

type AccountRes struct {
	Username string `json:"username"`
}

type PasswordReq struct {
	g.Meta      `path:"/password" method:"post" tags:"Admin"`
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

type PasswordRes struct {
	Changed bool `json:"changed"`
}

type HealthReq struct {
	g.Meta `path:"/health" method:"get" tags:"Admin"`
}

type HealthRes struct {
	store.AdminHealth
	Settings store.Settings `json:"settings"`
}

type BindingsReq struct {
	g.Meta `path:"/bindings" method:"get" tags:"Admin"`
}

type BindingsRes struct {
	Items []store.BindingRow `json:"items"`
}

type BindingGetReq struct {
	g.Meta `path:"/bindings/{id}" method:"get" tags:"Admin"`
	ID     string `json:"id" in:"path" v:"required"`
}

type BindingGetRes struct {
	*store.BindingDetail
}

type DisableBindingReq struct {
	g.Meta `path:"/bindings/{id}/disable" method:"post" tags:"Admin" summary:"停用绑定，只读运维，不下发车控"`
	ID     string `json:"id" in:"path" v:"required"`
}

type DisableBindingRes struct {
	Disabled bool `json:"disabled"`
}

type EnableBindingReq struct {
	g.Meta `path:"/bindings/{id}/enable" method:"post" tags:"Admin"`
	ID     string `json:"id" in:"path" v:"required"`
}

type EnableBindingRes struct {
	Disabled bool `json:"disabled"`
}

type SyncBindingReq struct {
	g.Meta `path:"/bindings/{id}/sync" method:"post" tags:"Admin" summary:"只读同步，不发车控"`
	ID     string `json:"id" in:"path" v:"required"`
}

type SyncBindingRes struct {
	Status   string `json:"status"`
	Error    string `json:"error"`
	SyncedAt any    `json:"syncedAt"`
}

type KickBindingReq struct {
	g.Meta `path:"/bindings/{id}/kick" method:"post" tags:"Admin"`
	ID     string `json:"id" in:"path" v:"required"`
}

type KickBindingRes struct {
	Kicked bool `json:"kicked"`
}

type JobsReq struct {
	g.Meta     `path:"/jobs" method:"get" tags:"Admin"`
	BindingId  string `json:"bindingId" in:"query"`
	Status     string `json:"status" in:"query"`
	Page       int    `json:"page" in:"query"`
	PageSize   int    `json:"pageSize" in:"query"`
}

type JobsRes struct {
	Total int            `json:"total"`
	Items []store.JobRow `json:"items"`
}

type TablesReq struct {
	g.Meta `path:"/tables" method:"get" tags:"Admin"`
}

type TablesRes struct {
	Items []map[string]any `json:"items"`
}

type TableRowsReq struct {
	g.Meta     `path:"/tables/{name}" method:"get" tags:"Admin"`
	Name       string `json:"name" in:"path" v:"required"`
	BindingId  string `json:"bindingId" in:"query"`
	Page       int    `json:"page" in:"query"`
	PageSize   int    `json:"pageSize" in:"query"`
}

type TableRowsRes struct {
	*store.TablePage
}

type SettingsGetReq struct {
	g.Meta `path:"/settings" method:"get" tags:"Admin"`
}

type SettingsGetRes struct {
	store.Settings
}

type SettingsPutReq struct {
	g.Meta        `path:"/settings" method:"put" tags:"Admin"`
	CronSync      string `json:"cronSync"`
	CORSOrigins   string `json:"corsOrigins"`
	SnapshotKeep  int    `json:"snapshotKeep"`
	StaleAfterSec int    `json:"staleAfterSec"`
}

type SettingsPutRes struct {
	store.Settings
}
