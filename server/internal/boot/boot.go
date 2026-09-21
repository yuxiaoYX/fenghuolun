package boot

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"fenghuolun/internal/config"
	"fenghuolun/internal/controller/admin"
	"fenghuolun/internal/controller/health"
	"fenghuolun/internal/controller/owner"
	"fenghuolun/internal/middleware"
	ownersvc "fenghuolun/internal/owner"
	"fenghuolun/internal/store"
)

func Run(cfg config.Config) error {
	s := g.Server()
	s.SetDumpRouterMap(true)
	Register(s, cfg)
	s.SetAddr(cfg.HTTPAddr)
	s.Run()
	return nil
}

func Register(s *ghttp.Server, cfg config.Config) *ownersvc.Service {
	st, err := store.OpenFromConfig(cfg)
	if err != nil {
		panic(err)
	}
	svc := ownersvc.New(cfg, st)
	s.Use(middleware.CORS(func() string {
		if v := st.GetSetting(store.SettingCORS); v != "" {
			return v
		}
		return cfg.CORSOrigins
	}))
	s.Group("/", func(group *ghttp.RouterGroup) {
		group.Middleware(middleware.Envelope)
		group.Bind(health.NewV1(cfg))
	})
	s.Group("/api/v1/owner", func(group *ghttp.RouterGroup) {
		group.Middleware(middleware.Envelope, middleware.OwnerAuth(svc))
		group.Bind(owner.NewV1(svc, cfg))
	})
	s.Group("/api/v1/admin", func(group *ghttp.RouterGroup) {
		group.Middleware(middleware.Envelope, middleware.AdminAuth(st))
		group.Bind(admin.NewV1(st, svc, cfg))
	})
	if err := ownersvc.StartCron(context.Background(), cfg, svc); err != nil {
		panic(err)
	}
	registerAdminSPA(s, cfg.AdminDir)
	return svc
}
