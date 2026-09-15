package middleware

import (
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"
)

func CORS(origins func() string) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		allow := map[string]struct{}{}
		raw := ""
		if origins != nil {
			raw = origins()
		}
		for _, item := range strings.Split(raw, ",") {
			item = strings.TrimSpace(item)
			if item != "" {
				allow[item] = struct{}{}
			}
		}
		origin := r.Header.Get("Origin")
		ok := false
		if origin != "" {
			if _, hit := allow[origin]; hit {
				ok = true
			}
			if strings.HasPrefix(origin, "http://127.0.0.1:") || strings.HasPrefix(origin, "http://localhost:") {
				ok = true
			}
		}
		if ok {
			r.Response.Header().Set("Access-Control-Allow-Origin", origin)
			r.Response.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			r.Response.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			r.Response.WriteStatus(http.StatusNoContent)
			r.ExitAll()
		}
		r.Middleware.Next()
	}
}
