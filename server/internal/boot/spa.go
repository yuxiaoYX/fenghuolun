package boot

import (
	"os"
	"path/filepath"

	"github.com/gogf/gf/v2/net/ghttp"
)

func registerOwnerSPA(s *ghttp.Server, dir string) {
	dir = filepath.Clean(dir)
	if dir == "" || dir == "." {
		return
	}
	index := filepath.Join(dir, "index.html")
	if _, err := os.Stat(index); err != nil {
		return
	}
	s.BindHandler("GET:/", func(r *ghttp.Request) {
		r.Response.ServeFile(index)
	})
	mountDir(s, "/assets", filepath.Join(dir, "assets"))
	mountDir(s, "/static", filepath.Join(dir, "static"))
}

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
	s.BindHandler("GET:/admin", serveIndex)
	s.BindHandler("GET:/login", func(r *ghttp.Request) {
		r.Response.RedirectTo("/admin/login")
	})
	for _, p := range []string{"/admin/login", "/admin/bindings", "/admin/jobs", "/admin/data", "/admin/settings", "/admin/account"} {
		s.BindHandler("GET:"+p, serveIndex)
	}
	s.BindHandler("GET:/admin/bindings/:id", serveIndex)
	mountDir(s, "/admin/assets", filepath.Join(dir, "assets"))
}

func mountDir(s *ghttp.Server, urlPath, dir string) {
	fi, err := os.Stat(dir)
	if err != nil || !fi.IsDir() {
		return
	}
	s.AddStaticPath(urlPath, dir)
}
