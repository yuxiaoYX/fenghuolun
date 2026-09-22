package update

// applyInput is the pure part of Probe. Docker and GitHub I/O stay outside.
type applyInput struct {
	Disabled        bool
	UpdateAvailable bool
	Latest          string
	Image           string
	BundleReady     bool
	ReleaseBuild    bool
	InDocker        bool
	DockerReady     bool
	SelfOK          bool
	Updating        bool
	Target          string
	Failure         string
}

func decideApply(in applyInput) (can bool, mode, hint string) {
	if in.Disabled {
		return false, "", "已设置 FENGHUOLUN_UPDATE_DISABLE，后台更新关闭"
	}
	if in.Updating {
		return false, "", "正在更新到 " + in.Target + "，请稍候，不要重复点击。"
	}
	if !in.UpdateAvailable {
		if in.Failure != "" {
			return false, "", in.Failure
		}
		if in.Latest != "" {
			return false, "", "已是最新版 " + in.Latest
		}
		return false, "", ""
	}
	if !in.InDocker {
		return false, "", "当前不是容器进程（本地 go run / 源码）。请用 Docker Compose 部署后再用后台更新，或在宿主机：docker compose pull && docker compose up -d"
	}
	if !in.ReleaseBuild {
		return false, "", "当前是开发构建，不能后台更新。请换用带版本号的 Release 镜像，或在宿主机：docker compose pull && docker compose up -d"
	}
	if in.BundleReady && in.InDocker {
		hint = "将备份数据库，下载程序包到数据目录并重启。页面会短暂不可用。"
		if in.Failure != "" {
			hint = in.Failure + " 可以再试一次。"
		}
		return true, "bundle", hint
	}
	if in.DockerReady && in.InDocker && in.SelfOK {
		hint = "将备份数据库、拉取 " + in.Image + " 并重建本容器。页面会短暂不可用。"
		if in.Failure != "" {
			hint = in.Failure + " 可以再试一次。"
		}
		return true, "docker", hint
	}
	if in.DockerReady && in.InDocker && !in.SelfOK {
		return false, "", "Docker 可用，但找不到本容器。给容器名 fenghuolun，或设 FENGHUOLUN_CONTAINER_NAME。"
	}
	return false, "", "这个版本还没有可下载的程序包，且未挂载 Docker 套接字。在宿主机执行：docker compose pull && docker compose up -d"
}
