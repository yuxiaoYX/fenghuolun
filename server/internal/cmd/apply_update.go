package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gogf/gf/v2/os/gcmd"

	"fenghuolun/internal/update"
)

var ApplyUpdate = gcmd.Command{
	Name:  "apply-update",
	Usage: "fenghuolun apply-update --container NAME --image IMAGE",
	Brief: "内部命令：把指定 Docker 容器换成新镜像。由后台更新拉起，不要手跑。",
	Arguments: []gcmd.Argument{
		{Name: "container", Short: "c", Brief: "要替换的容器名"},
		{Name: "image", Short: "i", Brief: "目标镜像，如 ghcr.io/yuxiaoyx/fenghuolun:v0.1.1"},
	},
	Func: func(ctx context.Context, parser *gcmd.Parser) error {
		container := strings.TrimSpace(parser.GetOpt("container", "").String())
		image := strings.TrimSpace(parser.GetOpt("image", "").String())
		if container == "" || image == "" {
			return fmt.Errorf("apply-update requires --container and --image")
		}
		if err := update.ApplyInHelper(ctx, container, image); err != nil {
			fmt.Fprintf(os.Stderr, "[fenghuolun] apply-update failed: %v\n", err)
			return err
		}
		return nil
	},
}
