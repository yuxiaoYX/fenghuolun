package middleware

import (
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"

	"fenghuolun/internal/owner"
)

func OwnerAuth(svc *owner.Service) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/bind") {
			r.Middleware.Next()
			return
		}
		if r.Method == http.MethodOptions {
			r.Middleware.Next()
			return
		}
		token := Bearer(r)
		if token == "" || svc.Get(token) == nil {
			WriteErr(r, http.StatusUnauthorized, "unauthorized", "请先绑定")
			r.ExitAll()
			return
		}
		r.Middleware.Next()
	}
}

func Bearer(r *ghttp.Request) string {
	h := r.Header.Get("Authorization")
	return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
}
