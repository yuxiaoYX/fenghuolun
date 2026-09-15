package middleware

import (
	"net/http"

	"github.com/gogf/gf/v2/net/ghttp"

	"fenghuolun/internal/store"
)

func AdminAuth(st *store.SQLite) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		if r.Method == http.MethodPost && (r.URL.Path == "/api/v1/admin/login" || r.URL.Path == "/login") {
			r.Middleware.Next()
			return
		}
		if r.Method == http.MethodOptions {
			r.Middleware.Next()
			return
		}
		if !st.AdminOK(Bearer(r)) {
			WriteErr(r, http.StatusUnauthorized, "unauthorized", "请先登录管理员")
			r.ExitAll()
			return
		}
		r.Middleware.Next()
	}
}
