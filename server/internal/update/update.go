package update

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	Mode            string `json:"mode,omitempty"`
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

	var rel Release
	var relErr error
	rel, relErr = LatestRelease(forceLatest)
	githubHint := ""
	if relErr != nil {
		githubHint = "查不到 GitHub Release：" + relErr.Error()
	} else {
		st.Latest = rel.Tag
		st.LatestURL = rel.URL
		st.Image = ImageRef(rel.Tag)
		st.UpdateAvailable = CompareTags(rel.Tag, st.Version) > 0
	}

	dataDir := sqliteDirHint()
	if flag, ok := readProgress(dataDir); ok {
		if flag.Target != "" && CompareTags(st.Version, flag.Target) >= 0 && IsReleaseTag(st.Version) {
			clearProgress("")
			clearUpdateError("")
		} else {
			st.Updating = true
			st.Target = flag.Target
		}
	}
	failure := ""
	if !st.Updating {
		if errFile, ok := readUpdateError(dataDir); ok {
			if errFile.Target != "" && CompareTags(st.Version, errFile.Target) >= 0 && IsReleaseTag(st.Version) {
				clearUpdateError("")
			} else {
				failure = "更新失败：" + errFile.Error
			}
		}
	}

	_, _, bundleReady := selectBundle(rel.Tag, rel.Assets, runtime.GOOS, runtime.GOARCH)
	selfOK := false
	if st.DockerAvailable && st.InDocker && !bundleReady {
		if _, err := inspectSelf(ctx); err == nil {
			selfOK = true
		}
	}
	can, mode, hint := decideApply(applyInput{
		Disabled:        strings.TrimSpace(os.Getenv("FENGHUOLUN_UPDATE_DISABLE")) == "1",
		UpdateAvailable: st.UpdateAvailable,
		Latest:          st.Latest,
		Image:           st.Image,
		BundleReady:     bundleReady && relErr == nil,
		ReleaseBuild:    IsReleaseTag(st.Version),
		InDocker:        st.InDocker,
		DockerReady:     st.DockerAvailable,
		SelfOK:          selfOK,
		Updating:        st.Updating,
		Target:          st.Target,
		Failure:         failure,
	})
	st.CanApply = can
	st.Mode = mode
	switch {
	case strings.TrimSpace(os.Getenv("FENGHUOLUN_UPDATE_DISABLE")) == "1" || st.Updating:
		st.Hint = hint
	case githubHint != "" && !st.UpdateAvailable:
		st.Hint = githubHint
	case hint != "":
		st.Hint = hint
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
	mode := st.Mode
	image := ImageRef(target)
	rel, err := LatestRelease(false)
	if err != nil && mode == "bundle" {
		applyMu.Lock()
		busy = false
		applyMu.Unlock()
		return ApplyResult{}, fmt.Errorf("github: %w", err)
	}
	if _, err := BackupSQLite(sqlitePath); err != nil {
		applyMu.Lock()
		busy = false
		applyMu.Unlock()
		return ApplyResult{}, fmt.Errorf("backup sqlite: %w", err)
	}
	clearUpdateError(sqlitePath)
	writeProgress(sqlitePath, target)

	go func() {
		defer func() {
			applyMu.Lock()
			busy = false
			applyMu.Unlock()
		}()
		bg, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
		defer cancel()
		var runErr error
		if mode == "bundle" {
			runErr = applyBundle(bg, dataDirFromSQLite(sqlitePath), rel, runtime.GOOS, runtime.GOARCH)
		} else {
			runErr = launchHelper(bg, image)
		}
		if runErr != nil {
			clearProgress(sqlitePath)
			writeUpdateError(sqlitePath, target, runErr.Error())
			fmt.Fprintf(os.Stderr, "[fenghuolun] update failed: %v\n", runErr)
			return
		}
		if mode == "bundle" {
			fmt.Fprintf(os.Stderr, "[fenghuolun] bundle %s installed, restarting\n", target)
			exitProcess(0)
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

func dataDirFromSQLite(sqlitePath string) string {
	if sqlitePath != "" && sqlitePath != ":memory:" {
		return filepath.Dir(sqlitePath)
	}
	return sqliteDirHint()
}

type updateErrorFile struct {
	Target string `json:"target"`
	Error  string `json:"error"`
	At     int64  `json:"at"`
}

func updateErrorPath(sqlitePath string) string {
	return filepath.Join(dataDirFromSQLite(sqlitePath), "update-last-error.json")
}

func writeUpdateError(sqlitePath, target, msg string) {
	msg = strings.TrimSpace(msg)
	if len(msg) > 300 {
		msg = msg[:300]
	}
	p := updateErrorPath(sqlitePath)
	_ = os.MkdirAll(filepath.Dir(p), 0o700)
	b, _ := json.Marshal(updateErrorFile{Target: target, Error: msg, At: time.Now().Unix()})
	_ = os.WriteFile(p, b, 0o600)
}

func clearUpdateError(sqlitePath string) {
	_ = os.Remove(updateErrorPath(sqlitePath))
}

func readUpdateError(dir string) (updateErrorFile, bool) {
	b, err := os.ReadFile(filepath.Join(dir, "update-last-error.json"))
	if err != nil {
		return updateErrorFile{}, false
	}
	var f updateErrorFile
	if json.Unmarshal(b, &f) != nil || f.Error == "" {
		return updateErrorFile{}, false
	}
	if f.At > 0 && time.Since(time.Unix(f.At, 0)) > 30*time.Minute {
		return updateErrorFile{}, false
	}
	return f, true
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
