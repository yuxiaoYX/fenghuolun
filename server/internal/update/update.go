package update

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fenghuolun/internal/version"
)

type Status struct {
	Version         string `json:"version"`
	Commit          string `json:"commit,omitempty"`
	Latest          string `json:"latest"`
	LatestURL       string `json:"latestUrl,omitempty"`
	UpdateAvailable bool   `json:"updateAvailable"`
	InDocker        bool   `json:"inDocker"`
	DockerAvailable bool   `json:"dockerAvailable"`
	CanApply        bool   `json:"canApply"`
	Updating        bool   `json:"updating"`
	Target          string `json:"target,omitempty"`
	Image           string `json:"image,omitempty"`
	Hint            string `json:"hint"`
}

type ApplyResult struct {
	Target string `json:"target"`
	Status string `json:"status"`
}

var (
	applyMu sync.Mutex
	busy    bool
)

func Probe(ctx context.Context, forceLatest bool) Status {
	st := Status{
		Version: version.Display(),
		Commit:  version.Commit,
		Hint:    "",
	}
	if strings.TrimSpace(os.Getenv("FENGHUOLUN_UPDATE_DISABLE")) == "1" {
		st.Hint = "已设置 FENGHUOLUN_UPDATE_DISABLE，后台更新关闭"
	}
	st.InDocker = InDocker()
	if err := DockerPing(ctx); err == nil {
		st.DockerAvailable = true
	}

	rel, err := LatestRelease(forceLatest)
	if err != nil {
		if st.Hint == "" {
			st.Hint = "查不到 GitHub Release：" + err.Error()
		}
	} else {
		st.Latest = rel.Tag
		st.LatestURL = rel.URL
		st.Image = ImageRef(rel.Tag)
		st.UpdateAvailable = CompareTags(rel.Tag, st.Version) > 0
	}

	if flag, ok := readProgress(sqliteDirHint()); ok {
		if flag.Target != "" && CompareTags(st.Version, flag.Target) >= 0 && IsReleaseTag(st.Version) {
			clearProgress("")
		} else {
			st.Updating = true
			st.Target = flag.Target
		}
	}

	switch {
	case st.Hint != "" && strings.Contains(st.Hint, "FENGHUOLUN_UPDATE_DISABLE"):
		st.CanApply = false
	case !st.UpdateAvailable:
		st.CanApply = false
		if st.Latest != "" && st.Hint == "" {
			st.Hint = "已是最新版 " + st.Latest
		}
	case !st.DockerAvailable:
		st.CanApply = false
		if st.Hint == "" {
			st.Hint = "未挂载 Docker 套接字，无法在后台一点更新。把 /var/run/docker.sock 挂进容器，或在宿主机执行：docker compose pull && docker compose up -d"
		}
	case !st.InDocker:
		st.CanApply = false
		if st.Hint == "" {
			st.Hint = "当前不是容器进程（本地 go run / 源码）。请用 Docker Compose 部署后再用后台更新，或在宿主机：docker compose pull && docker compose up -d"
		}
	default:
		if _, err := inspectSelf(ctx); err != nil {
			st.CanApply = false
			if st.Hint == "" {
				st.Hint = "Docker 可用，但找不到本容器。给容器名 fenghuolun，或设 FENGHUOLUN_CONTAINER_NAME。"
			}
		} else {
			st.CanApply = true
			if st.Hint == "" {
				st.Hint = "将备份数据库、拉取 " + st.Image + " 并重建本容器。页面会短暂不可用。"
			}
		}
	}
	if st.Updating {
		st.CanApply = false
		st.Hint = "正在更新到 " + st.Target + "，请稍候，不要重复点击。"
	}
	return st
}

func StartApply(ctx context.Context, sqlitePath string) (ApplyResult, error) {
	applyMu.Lock()
	if busy {
		applyMu.Unlock()
		return ApplyResult{}, fmt.Errorf("update already running")
	}
	st := Probe(ctx, true)
	if !st.CanApply {
		applyMu.Unlock()
		return ApplyResult{}, fmt.Errorf("%s", st.Hint)
	}
	busy = true
	applyMu.Unlock()

	target := st.Latest
	image := ImageRef(target)
	if _, err := BackupSQLite(sqlitePath); err != nil {
		applyMu.Lock()
		busy = false
		applyMu.Unlock()
		return ApplyResult{}, fmt.Errorf("backup sqlite: %w", err)
	}
	writeProgress(sqlitePath, target)

	go func() {
		defer func() {
			applyMu.Lock()
			busy = false
			applyMu.Unlock()
		}()
		bg, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
		defer cancel()
		if err := launchHelper(bg, image); err != nil {
			clearProgress(sqlitePath)
			fmt.Fprintf(os.Stderr, "[fenghuolun] update helper failed: %v\n", err)
		}
	}()

	return ApplyResult{Target: target, Status: "started"}, nil
}

func launchHelper(ctx context.Context, image string) error {
	cli := newDocker()
	self, err := inspectSelf(ctx)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "[fenghuolun] pulling %s\n", image)
	if err := cli.pull(ctx, image); err != nil {
		return err
	}
	repo, _ := splitImage(image)
	_ = cli.tag(ctx, image, repo, "latest")

	helper := updaterName()
	_ = cli.remove(ctx, helper, true)
	body, err := buildHelperBody(image, self.NameTrim(), image, sockHostPath())
	if err != nil {
		return err
	}
	id, err := cli.create(ctx, helper, body)
	if err != nil {
		return err
	}
	if err := cli.start(ctx, id); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "[fenghuolun] updater %s started\n", helper)
	return nil
}

// ApplyInHelper runs inside the one-shot updater container.
func ApplyInHelper(ctx context.Context, containerName, image string) error {
	containerName = strings.TrimPrefix(strings.TrimSpace(containerName), "/")
	image = strings.TrimSpace(image)
	if containerName == "" || image == "" {
		return fmt.Errorf("apply-update requires --container and --image")
	}
	cli := newDocker()
	old, err := cli.inspect(ctx, containerName)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "[fenghuolun] apply %s -> %s\n", old.NameTrim(), image)
	if err := cli.pull(ctx, image); err != nil {
		return err
	}
	repo, _ := splitImage(image)
	_ = cli.tag(ctx, image, repo, "latest")

	freshName := nextName(old.NameTrim())
	_ = cli.remove(ctx, freshName, true)
	body, err := buildCreateBody(old, image)
	if err != nil {
		return err
	}
	newID, err := cli.create(ctx, freshName, body)
	if err != nil {
		return err
	}

	if err := cli.stop(ctx, old.ID, 20); err != nil {
		_ = cli.remove(ctx, newID, true)
		return fmt.Errorf("stop old: %w", err)
	}
	if err := cli.start(ctx, newID); err != nil {
		_ = cli.start(ctx, old.ID)
		_ = cli.remove(ctx, newID, true)
		return fmt.Errorf("start new: %w", err)
	}
	if err := waitRunning(ctx, cli, newID, 45*time.Second); err != nil {
		_ = cli.stop(ctx, newID, 5)
		_ = cli.remove(ctx, newID, true)
		_ = cli.start(ctx, old.ID)
		return err
	}
	if err := cli.remove(ctx, old.ID, true); err != nil {
		fmt.Fprintf(os.Stderr, "[fenghuolun] remove old: %v\n", err)
	}
	if err := cli.rename(ctx, newID, old.NameTrim()); err != nil {
		fmt.Fprintf(os.Stderr, "[fenghuolun] rename to %s failed (running as %s): %v\n", old.NameTrim(), freshName, err)
	}
	fmt.Fprintf(os.Stderr, "[fenghuolun] update ok %s\n", image)
	return nil
}

type progressFile struct {
	Target    string `json:"target"`
	StartedAt int64  `json:"startedAt"`
}

func progressPath(sqlitePath string) string {
	dir := sqliteDirHint()
	if sqlitePath != "" && sqlitePath != ":memory:" {
		dir = filepath.Dir(sqlitePath)
	}
	return filepath.Join(dir, "update-in-progress.json")
}

func sqliteDirHint() string {
	if v := strings.TrimSpace(os.Getenv("FENGHUOLUN_SQLITE_PATH")); v != "" && v != ":memory:" {
		return filepath.Dir(v)
	}
	return "/var/lib/fenghuolun"
}

func writeProgress(sqlitePath, target string) {
	p := progressPath(sqlitePath)
	_ = os.MkdirAll(filepath.Dir(p), 0o700)
	b, _ := json.Marshal(progressFile{Target: target, StartedAt: time.Now().Unix()})
	_ = os.WriteFile(p, b, 0o600)
}

func clearProgress(sqlitePath string) {
	_ = os.Remove(progressPath(sqlitePath))
}

func readProgress(dir string) (progressFile, bool) {
	p := filepath.Join(dir, "update-in-progress.json")
	b, err := os.ReadFile(p)
	if err != nil {
		return progressFile{}, false
	}
	var f progressFile
	if json.Unmarshal(b, &f) != nil {
		return progressFile{}, false
	}
	if f.StartedAt > 0 && time.Since(time.Unix(f.StartedAt, 0)) > 25*time.Minute {
		return progressFile{}, false
	}
	return f, f.Target != ""
}
