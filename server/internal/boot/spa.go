package boot

import (
	"os"
	"path/filepath"

	"github.com/gogf/gf/v2/net/ghttp"
)

func registerAdminSPA(s *ghttp.Server, dir string) {
	dir = filepath.Clean(dir)
	if dir == "" || dir == "." {
		return
	}
	index := filepath.Join(dir, "index.html")
	if _, err := os.Stat(index); err != nil {
		return
	}
	serveIndex := func(r *ghttp.Request) {
		r.Response.ServeFile(index)
	}
	s.BindHandler("GET:/", serveIndex)
	for _, p := range []string{"/login", "/bindings", "/jobs", "/data", "/settings", "/account"} {
		s.BindHandler("GET:"+p, serveIndex)
	}
	s.BindHandler("GET:/bindings/:id", serveIndex)
	assets := filepath.Join(dir, "assets")
	if fi, err := os.Stat(assets); err == nil && fi.IsDir() {
		s.AddStaticPath("/assets", assets)
	}
}
