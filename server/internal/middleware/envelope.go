package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"

	"fenghuolun/internal/coded"
	"fenghuolun/internal/neta"
)

func Envelope(r *ghttp.Request) {
	r.Middleware.Next()
	if r.Response.BufferLength() > 0 {
		return
	}
	if err := r.GetError(); err != nil {
		WriteBiz(r, err)
		return
	}
	WriteOK(r, r.GetHandlerResponse())
}

func WriteOK(r *ghttp.Request, data any) {
	r.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
	r.Response.Status = http.StatusOK
	r.Response.WriteJson(map[string]any{"ok": true, "data": data})
}

func WriteErr(r *ghttp.Request, status int, code, message string) {
	r.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
	r.Response.Status = status
	r.Response.WriteJson(map[string]any{
		"ok": false,
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func WriteBiz(r *ghttp.Request, err error) {
	var ce *coded.Error
	if errors.As(err, &ce) {
		WriteErr(r, ce.HTTP, ce.Code, ce.Msg)
		return
	}
	msg := err.Error()
	switch {
	case strings.HasPrefix(msg, "invalid_request"):
		public := "请求不合法"
		switch {
		case strings.Contains(msg, "refresh_token") || strings.TrimSpace(strings.TrimPrefix(msg, "invalid_request:")) == "":
			public = "请填写 refresh_token"
		case strings.Contains(msg, "sync already running"):
			public = "该车正在同步"
		case strings.Contains(msg, "password"):
			public = "新密码至少 8 位"
		case strings.Contains(msg, "unknown table"):
			public = "没有这张表"
		case strings.Contains(msg, "nickname"):
			public = "备注名最多 32 字"
		case strings.Contains(msg, "ledger"):
			public = "记账数字不在合理范围"
		case strings.Contains(msg, "fill"):
			public = "充能记录数字不在合理范围"
		}
		WriteErr(r, http.StatusBadRequest, "invalid_request", public)
	case strings.HasPrefix(msg, "unauthorized"):
		WriteErr(r, http.StatusUnauthorized, "unauthorized", "请先绑定")
	case strings.HasPrefix(msg, "not_found"):
		WriteErr(r, http.StatusNotFound, "not_found", "绑定不存在")
	case errors.Is(err, neta.ErrTokenInvalid):
		WriteErr(r, http.StatusUnauthorized, "token_invalid", "请重新填写 refresh_token")
	case errors.Is(err, neta.ErrDecode):
		WriteErr(r, http.StatusBadGateway, "decode", "官方响应结构变了")
	default:
		WriteErr(r, http.StatusBadGateway, "upstream", "官方云暂不可用")
	}
}
