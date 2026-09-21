package cmd

import (
	"context"

	"github.com/gogf/gf/v2/os/gcmd"

	"fenghuolun/internal/boot"
	"fenghuolun/internal/config"
)

var (
	Main = gcmd.Command{
		Name:  "fenghuolun",
		Usage: "fenghuolun",
		Brief: "风火轮 HTTP 服务",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {
			return boot.Run(config.Load())
		},
	}
)

func init() {
	if err := Main.AddCommand(&ApplyUpdate); err != nil {
		panic(err)
	}
}
