package middleware

import (
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"

	"fenghuolun/internal/owner"
)

func OwnerAuth(svc *owner.Service) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		if r.Method == http.MethodPost && ownerPublic(r.URL.Path) {
			r.Middleware.Next()
			return
		}
		if r.Method == http.MethodOptions {
			r.Middleware.Next()
			return
		}
		token := Bearer(r)
		if token != "" && svc.Get(token) != nil {
			r.Middleware.Next()
			return
		}
		if token != "" && svc.AccountSession(token) {
			r.Middleware.Next()
			return
		}
		WriteErr(r, http.StatusUnauthorized, "unauthorized", "请先登录")
		r.ExitAll()
	}
}

func ownerPublic(path string) bool {
	switch {
	case strings.HasSuffix(path, "/bind"):
		return true
	case strings.HasSuffix(path, "/register"), strings.HasSuffix(path, "/login"), strings.HasSuffix(path, "/sms/send"):
		return true
	default:
		return false
	}
}

func Bearer(r *ghttp.Request) string {
	h := r.Header.Get("Authorization")
	return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
}
