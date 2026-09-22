package admin

import (
	"context"
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/frame/g"

	v1 "fenghuolun/api/admin/v1"
	"fenghuolun/internal/coded"
	"fenghuolun/internal/middleware"
	"fenghuolun/internal/store"
	"fenghuolun/internal/update"
)

func (c *ControllerV1) Login(ctx context.Context, req *v1.LoginReq) (res *v1.LoginRes, err error) {
	tok, err := c.Store.AdminLogin(strings.TrimSpace(req.Username), req.Password)
	if err != nil {
		return nil, coded.New(http.StatusUnauthorized, "unauthorized", "用户名或密码不对")
	}
	return &v1.LoginRes{Session: tok}, nil
}

func (c *ControllerV1) Logout(ctx context.Context, req *v1.LogoutReq) (res *v1.LogoutRes, err error) {
	c.Store.AdminLogout(sessionOf(ctx))
	return &v1.LogoutRes{Ok: true}, nil
}

func (c *ControllerV1) Account(ctx context.Context, req *v1.AccountReq) (res *v1.AccountRes, err error) {
	return &v1.AccountRes{Username: c.Store.AdminUsername(sessionOf(ctx))}, nil
}

func (c *ControllerV1) Password(ctx context.Context, req *v1.PasswordReq) (res *v1.PasswordRes, err error) {
	if err := c.Store.AdminChangePassword(sessionOf(ctx), req.OldPassword, req.NewPassword); err != nil {
		if strings.HasPrefix(err.Error(), "invalid_request") {
			return nil, coded.New(http.StatusBadRequest, "invalid_request", "新密码至少 8 位")
		}
		return nil, coded.New(http.StatusUnauthorized, "unauthorized", "当前密码不对")
	}
	return &v1.PasswordRes{Changed: true}, nil
}

func (c *ControllerV1) Health(ctx context.Context, req *v1.HealthReq) (res *v1.HealthRes, err error) {
	h, err := c.Store.AdminHealth()
	if err != nil {
		return nil, err
	}
	return &v1.HealthRes{AdminHealth: h, Settings: c.Store.Settings()}, nil
}

func (c *ControllerV1) Bindings(ctx context.Context, req *v1.BindingsReq) (res *v1.BindingsRes, err error) {
	items, err := c.Store.ListBindings()
	if err != nil {
		return nil, err
	}
	return &v1.BindingsRes{Items: items}, nil
}

func (c *ControllerV1) BindingGet(ctx context.Context, req *v1.BindingGetReq) (res *v1.BindingGetRes, err error) {
	d, err := c.Store.BindingDetail(strings.TrimSpace(req.ID))
	if err != nil {
		return nil, err
	}
	return &v1.BindingGetRes{BindingDetail: d}, nil
}

func (c *ControllerV1) DisableBinding(ctx context.Context, req *v1.DisableBindingReq) (res *v1.DisableBindingRes, err error) {
	if err := c.Svc.Disable(strings.TrimSpace(req.ID)); err != nil {
		return nil, err
	}
	return &v1.DisableBindingRes{Disabled: true}, nil
}

func (c *ControllerV1) EnableBinding(ctx context.Context, req *v1.EnableBindingReq) (res *v1.EnableBindingRes, err error) {
	if err := c.Svc.Enable(strings.TrimSpace(req.ID)); err != nil {
		return nil, err
	}
	return &v1.EnableBindingRes{Disabled: false}, nil
}

func (c *ControllerV1) SyncBinding(ctx context.Context, req *v1.SyncBindingReq) (res *v1.SyncBindingRes, err error) {
	b, err := c.Svc.SyncByID(strings.TrimSpace(req.ID), "admin")
	if err != nil {
		return nil, err
	}
	return &v1.SyncBindingRes{Status: b.SyncStatus, Error: b.SyncError, SyncedAt: b.SyncedAt}, nil
}

func (c *ControllerV1) KickBinding(ctx context.Context, req *v1.KickBindingReq) (res *v1.KickBindingRes, err error) {
	if err := c.Svc.Kick(strings.TrimSpace(req.ID)); err != nil {
		return nil, err
	}
	return &v1.KickBindingRes{Kicked: true}, nil
}

func (c *ControllerV1) Jobs(ctx context.Context, req *v1.JobsReq) (res *v1.JobsRes, err error) {
	items, total, err := c.Store.ListJobs(strings.TrimSpace(req.BindingId), strings.TrimSpace(req.Status), req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}
	return &v1.JobsRes{Total: total, Items: items}, nil
}

func (c *ControllerV1) Tables(ctx context.Context, req *v1.TablesReq) (res *v1.TablesRes, err error) {
	return &v1.TablesRes{Items: c.Store.TableNames()}, nil
}

func (c *ControllerV1) TableRows(ctx context.Context, req *v1.TableRowsReq) (res *v1.TableRowsRes, err error) {
	page, err := c.Store.ListTable(strings.TrimSpace(req.Name), strings.TrimSpace(req.BindingId), req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}
	return &v1.TableRowsRes{TablePage: page}, nil
}

func (c *ControllerV1) SettingsGet(ctx context.Context, req *v1.SettingsGetReq) (res *v1.SettingsGetRes, err error) {
	return &v1.SettingsGetRes{Settings: c.Store.Settings()}, nil
}

func (c *ControllerV1) SettingsPut(ctx context.Context, req *v1.SettingsPutReq) (res *v1.SettingsPutRes, err error) {
	prev := c.Store.Settings()
	saved, err := c.Store.SaveSettings(store.Settings{
		CronSync:      req.CronSync,
		CORSOrigins:   req.CORSOrigins,
		SnapshotKeep:  req.SnapshotKeep,
		StaleAfterSec: req.StaleAfterSec,
	})
	if err != nil {
		return nil, coded.New(http.StatusBadRequest, "invalid_request", "设置不合法")
	}
	if err := c.Svc.ApplyCron(ctx, saved.CronSync); err != nil {
		_, _ = c.Store.SaveSettings(prev)
		_ = c.Svc.ApplyCron(ctx, prev.CronSync)
		return nil, coded.New(http.StatusBadRequest, "invalid_request", "定时同步表达式无法使用")
	}
	return &v1.SettingsPutRes{Settings: saved}, nil
}

func (c *ControllerV1) System(ctx context.Context, req *v1.SystemReq) (res *v1.SystemRes, err error) {
	st := update.Probe(ctx, req.Refresh)
	return &v1.SystemRes{
		Version:         st.Version,
		Commit:          st.Commit,
		Latest:          st.Latest,
		LatestURL:       st.LatestURL,
		UpdateAvailable: st.UpdateAvailable,
		InDocker:        st.InDocker,
		DockerAvailable: st.DockerAvailable,
		CanApply:        st.CanApply,
		Mode:            st.Mode,
		Updating:        st.Updating,
		Target:          st.Target,
		Image:           st.Image,
		Hint:            st.Hint,
	}, nil
}

func (c *ControllerV1) SystemUpdate(ctx context.Context, req *v1.SystemUpdateReq) (res *v1.SystemUpdateRes, err error) {
	out, err := update.StartApply(ctx, c.Cfg.SQLitePath)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "already running") {
			return nil, coded.New(http.StatusBadRequest, "invalid_request", "正在更新，请稍候")
		}
		return nil, coded.New(http.StatusBadRequest, "invalid_request", msg)
	}
	return &v1.SystemUpdateRes{Target: out.Target, Status: out.Status}, nil
}

func sessionOf(ctx context.Context) string {
	return middleware.Bearer(g.RequestFromCtx(ctx))
}
