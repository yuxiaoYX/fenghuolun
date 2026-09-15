package boot

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/guid"

	"fenghuolun/internal/config"
	"fenghuolun/internal/neta"
)

type stubUpstream struct {
	dir string
}

func testdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "testdata", "neta"))
}

func (s stubUpstream) Refresh(refreshToken string) (neta.TokenPair, error) {
	if len(refreshToken) < 8 {
		return neta.TokenPair{}, neta.ErrTokenInvalid
	}
	return neta.TokenPair{AccessToken: "test-access", RefreshToken: refreshToken, ExpiresIn: 604799}, nil
}

func (s stubUpstream) GetCurrentVehicle(accessToken string) ([]byte, error) {
	return os.ReadFile(filepath.Join(s.dir, "getCurrentVehicle.sample.json"))
}

func (s stubUpstream) GetAppVehicleData(accessToken, vin string) ([]byte, error) {
	return os.ReadFile(filepath.Join(s.dir, "getAppVehicleData.sample.json"))
}

func (s stubUpstream) QueryEnergyByVin(accessToken, vin string, periodType int) ([]byte, error) {
	return os.ReadFile(filepath.Join(s.dir, "energyByVin.sample.json"))
}

func startOwner(t *testing.T) string {
	t.Helper()
	s := g.Server(guid.S())
	s.SetDumpRouterMap(false)
	s.SetAccessLogEnabled(false)
	s.SetErrorLogEnabled(false)
	svc := Register(s, config.Config{
		ScaleCandidate: true,
		SQLitePath:     ":memory:",
		TokenKEK:       "test-kek",
		AdminUser:      "admin",
		AdminPassword:  "secret-pass-xx",
	})
	svc.Client = stubUpstream{dir: testdataDir(t)}
	s.SetPort(0)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Shutdown() })
	return "http://127.0.0.1:" + strconv.Itoa(s.GetListenedPort())
}

func TestAdminSPA(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<!doctype html><title>admin-spa</title>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log(1)"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := g.Server(guid.S())
	s.SetDumpRouterMap(false)
	s.SetAccessLogEnabled(false)
	s.SetErrorLogEnabled(false)
	Register(s, config.Config{
		SQLitePath:    ":memory:",
		TokenKEK:      "test-kek",
		AdminUser:     "admin",
		AdminPassword: "secret-pass-xx",
		AdminDir:      dir,
	})
	s.SetPort(0)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Shutdown() })
	prefix := "http://127.0.0.1:" + strconv.Itoa(s.GetListenedPort())
	html, err := g.Client().Get(context.Background(), prefix+"/bindings")
	if err != nil {
		t.Fatal(err)
	}
	defer html.Close()
	htmlBody := html.ReadAllString()
	if html.StatusCode != http.StatusOK || !strings.Contains(htmlBody, "admin-spa") {
		t.Fatalf("spa %d %s", html.StatusCode, htmlBody)
	}
	js, err := g.Client().Get(context.Background(), prefix+"/assets/app.js")
	if err != nil {
		t.Fatal(err)
	}
	defer js.Close()
	jsBody := js.ReadAllString()
	if js.StatusCode != http.StatusOK || !strings.Contains(jsBody, "console.log") {
		t.Fatalf("asset %d %s", js.StatusCode, jsBody)
	}
	hz, err := g.Client().Get(context.Background(), prefix+"/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer hz.Close()
	hzBody := hz.ReadAllString()
	if hz.StatusCode != http.StatusOK || !strings.Contains(hzBody, `"phase":"2"`) {
		t.Fatalf("healthz %d %s", hz.StatusCode, hzBody)
	}
}

func TestHealthz(t *testing.T) {
	prefix := startOwner(t)
	resp, err := g.Client().Get(context.Background(), prefix+"/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	body := resp.ReadAllString()
	if strings.Contains(body, "fixture") {
		t.Fatalf("healthz must not advertise fixture: %s", body)
	}
}

func TestOwnerUnauthorized(t *testing.T) {
	prefix := startOwner(t)
	resp, err := g.Client().Get(context.Background(), prefix+"/api/v1/owner/vehicle")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d body %s", resp.StatusCode, resp.ReadAllString())
	}
}

func TestOwnerBindSnapshotEnergy(t *testing.T) {
	prefix := startOwner(t)
	resp, err := g.Client().ContentJson().Post(context.Background(), prefix+"/api/v1/owner/bind", `{"refresh_token":"live-token-xxxx"}`)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bind %d %s", resp.StatusCode, resp.ReadAllString())
	}
	var env struct {
		OK   bool `json:"ok"`
		Data struct {
			Session string `json:"session"`
			Fixture any    `json:"fixture"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.ReadAll(), &env); err != nil {
		t.Fatal(err)
	}
	if !env.OK || env.Data.Session == "" {
		t.Fatalf("%+v", env)
	}
	if env.Data.Fixture != nil {
		t.Fatal("bind must not return fixture flag")
	}
	cli := g.Client().SetHeader("Authorization", "Bearer "+env.Data.Session)
	snap, err := cli.Get(context.Background(), prefix+"/api/v1/owner/snapshot/latest")
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()
	if snap.StatusCode != http.StatusOK {
		t.Fatalf("snap %d %s", snap.StatusCode, snap.ReadAllString())
	}
	var shot struct {
		OK   bool `json:"ok"`
		Data struct {
			FetchedAt  string  `json:"fetchedAt"`
			ReportedAt *string `json:"reportedAt"`
			Stale      bool    `json:"stale"`
			Online     *bool   `json:"online"`
			Power      struct {
				SocPct *float64 `json:"socPct"`
			} `json:"power"`
		} `json:"data"`
	}
	if err := json.Unmarshal(snap.ReadAll(), &shot); err != nil {
		t.Fatal(err)
	}
	if shot.Data.Power.SocPct == nil || *shot.Data.Power.SocPct != 50 {
		t.Fatalf("soc %+v", shot)
	}
	if _, err := time.ParseInLocation("2006-01-02 15:04:05", shot.Data.FetchedAt, time.FixedZone("CST", 8*3600)); err != nil {
		t.Fatalf("fetchedAt must be China wall clock, got %q", shot.Data.FetchedAt)
	}
	if strings.Contains(shot.Data.FetchedAt, "T") || strings.HasSuffix(shot.Data.FetchedAt, "Z") {
		t.Fatalf("fetchedAt must not be RFC3339: %q", shot.Data.FetchedAt)
	}
	if shot.Data.ReportedAt != nil {
		if _, err := time.ParseInLocation("2006-01-02 15:04:05", *shot.Data.ReportedAt, time.FixedZone("CST", 8*3600)); err != nil {
			t.Fatalf("reportedAt must be China wall clock, got %q", *shot.Data.ReportedAt)
		}
	}
	en, err := cli.Get(context.Background(), prefix+"/api/v1/owner/energy")
	if err != nil {
		t.Fatal(err)
	}
	defer en.Close()
	if en.StatusCode != http.StatusOK {
		t.Fatalf("energy %d %s", en.StatusCode, en.ReadAllString())
	}
	hist, err := cli.Get(context.Background(), prefix+"/api/v1/owner/snapshots")
	if err != nil {
		t.Fatal(err)
	}
	defer hist.Close()
	histBody := hist.ReadAllString()
	if hist.StatusCode != http.StatusOK || !strings.Contains(histBody, `"total"`) {
		t.Fatalf("snapshots %d %s", hist.StatusCode, histBody)
	}
	put, err := cli.ContentJson().Put(context.Background(), prefix+"/api/v1/owner/vehicle", `{"nickname":"家里那辆"}`)
	if err != nil {
		t.Fatal(err)
	}
	defer put.Close()
	putBody := put.ReadAllString()
	if put.StatusCode != http.StatusOK || !strings.Contains(putBody, "家里那辆") {
		t.Fatalf("nickname %d %s", put.StatusCode, putBody)
	}
}

func TestBindShortToken(t *testing.T) {
	prefix := startOwner(t)
	resp, err := g.Client().ContentJson().Post(context.Background(), prefix+"/api/v1/owner/bind", `{"refresh_token":"x"}`)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d %s", resp.StatusCode, resp.ReadAllString())
	}
}

func TestAdminLoginAndBindings(t *testing.T) {
	prefix := startOwner(t)
	deny, err := g.Client().Get(context.Background(), prefix+"/api/v1/admin/bindings")
	if err != nil {
		t.Fatal(err)
	}
	defer deny.Close()
	if deny.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d %s", deny.StatusCode, deny.ReadAllString())
	}
	login, err := g.Client().ContentJson().Post(context.Background(), prefix+"/api/v1/admin/login", `{"username":"admin","password":"secret-pass-xx"}`)
	if err != nil {
		t.Fatal(err)
	}
	defer login.Close()
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login %d %s", login.StatusCode, login.ReadAllString())
	}
	var env struct {
		OK   bool `json:"ok"`
		Data struct {
			Session string `json:"session"`
		} `json:"data"`
	}
	if err := json.Unmarshal(login.ReadAll(), &env); err != nil {
		t.Fatal(err)
	}
	list, err := g.Client().SetHeader("Authorization", "Bearer "+env.Data.Session).Get(context.Background(), prefix+"/api/v1/admin/bindings")
	if err != nil {
		t.Fatal(err)
	}
	defer list.Close()
	body := list.ReadAllString()
	if list.StatusCode != http.StatusOK {
		t.Fatalf("list %d %s", list.StatusCode, body)
	}
	if strings.Contains(body, "refresh_token") || strings.Contains(body, "access_token") {
		t.Fatal("must not leak tokens")
	}
}

func TestAdminDisableBindingStopsOwnerSession(t *testing.T) {
	prefix := startOwner(t)
	bind, err := g.Client().ContentJson().Post(context.Background(), prefix+"/api/v1/owner/bind", `{"refresh_token":"live-token-xxxx"}`)
	if err != nil {
		t.Fatal(err)
	}
	defer bind.Close()
	var ownerEnv struct {
		OK   bool `json:"ok"`
		Data struct {
			Session string `json:"session"`
		} `json:"data"`
	}
	if err := json.Unmarshal(bind.ReadAll(), &ownerEnv); err != nil {
		t.Fatal(err)
	}
	login, err := g.Client().ContentJson().Post(context.Background(), prefix+"/api/v1/admin/login", `{"username":"admin","password":"secret-pass-xx"}`)
	if err != nil {
		t.Fatal(err)
	}
	defer login.Close()
	var adminEnv struct {
		OK   bool `json:"ok"`
		Data struct {
			Session string `json:"session"`
		} `json:"data"`
	}
	if err := json.Unmarshal(login.ReadAll(), &adminEnv); err != nil {
		t.Fatal(err)
	}
	admin := g.Client().SetHeader("Authorization", "Bearer "+adminEnv.Data.Session)
	list, err := admin.Get(context.Background(), prefix+"/api/v1/admin/bindings")
	if err != nil {
		t.Fatal(err)
	}
	defer list.Close()
	listBody := list.ReadAllString()
	if strings.Contains(listBody, "TESTVIN0000000001") {
		t.Fatal("admin list must not show raw VIN")
	}
	if !strings.Contains(listBody, "****0001") {
		t.Fatalf("masked vin missing: %s", listBody)
	}
	var listEnv struct {
		OK   bool `json:"ok"`
		Data struct {
			Items []struct {
				ID        string `json:"id"`
				VinMasked string `json:"vinMasked"`
				Disabled  bool   `json:"disabled"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(listBody), &listEnv); err != nil {
		t.Fatal(err)
	}
	if len(listEnv.Data.Items) != 1 || listEnv.Data.Items[0].ID == "" {
		t.Fatalf("%+v", listEnv)
	}
	id := listEnv.Data.Items[0].ID
	dis, err := admin.ContentJson().Post(context.Background(), prefix+"/api/v1/admin/bindings/"+id+"/disable", "{}")
	if err != nil {
		t.Fatal(err)
	}
	defer dis.Close()
	if dis.StatusCode != http.StatusOK {
		t.Fatalf("disable %d %s", dis.StatusCode, dis.ReadAllString())
	}
	veh, err := g.Client().SetHeader("Authorization", "Bearer "+ownerEnv.Data.Session).Get(context.Background(), prefix+"/api/v1/owner/vehicle")
	if err != nil {
		t.Fatal(err)
	}
	defer veh.Close()
	if veh.StatusCode != http.StatusUnauthorized {
		t.Fatalf("owner after disable %d %s", veh.StatusCode, veh.ReadAllString())
	}
}

func TestAdminConsoleAPIs(t *testing.T) {
	prefix := startOwner(t)
	bind, err := g.Client().ContentJson().Post(context.Background(), prefix+"/api/v1/owner/bind", `{"refresh_token":"live-token-xxxx"}`)
	if err != nil {
		t.Fatal(err)
	}
	defer bind.Close()
	login, err := g.Client().ContentJson().Post(context.Background(), prefix+"/api/v1/admin/login", `{"username":"admin","password":"secret-pass-xx"}`)
	if err != nil {
		t.Fatal(err)
	}
	defer login.Close()
	var adminEnv struct {
		OK   bool `json:"ok"`
		Data struct {
			Session string `json:"session"`
		} `json:"data"`
	}
	if err := json.Unmarshal(login.ReadAll(), &adminEnv); err != nil {
		t.Fatal(err)
	}
	admin := g.Client().SetHeader("Authorization", "Bearer "+adminEnv.Data.Session)
	health, err := admin.Get(context.Background(), prefix+"/api/v1/admin/health")
	if err != nil {
		t.Fatal(err)
	}
	defer health.Close()
	healthBody := health.ReadAllString()
	if health.StatusCode != http.StatusOK {
		t.Fatalf("health %d %s", health.StatusCode, healthBody)
	}
	if strings.Contains(healthBody, "TESTVIN") {
		t.Fatal("health must not show VIN")
	}
	list, err := admin.Get(context.Background(), prefix+"/api/v1/admin/bindings")
	if err != nil {
		t.Fatal(err)
	}
	defer list.Close()
	var listEnv struct {
		OK   bool `json:"ok"`
		Data struct {
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(list.ReadAllString()), &listEnv); err != nil {
		t.Fatal(err)
	}
	id := listEnv.Data.Items[0].ID
	detail, err := admin.Get(context.Background(), prefix+"/api/v1/admin/bindings/"+id)
	if err != nil {
		t.Fatal(err)
	}
	defer detail.Close()
	detailBody := detail.ReadAllString()
	if detail.StatusCode != http.StatusOK {
		t.Fatalf("detail %d %s", detail.StatusCode, detailBody)
	}
	if strings.Contains(detailBody, "TESTVIN0000000001") || strings.Contains(detailBody, "live-token") {
		t.Fatal("detail must not leak VIN or token")
	}
	sync, err := admin.ContentJson().Post(context.Background(), prefix+"/api/v1/admin/bindings/"+id+"/sync", "{}")
	if err != nil {
		t.Fatal(err)
	}
	defer sync.Close()
	if sync.StatusCode != http.StatusOK {
		t.Fatalf("sync %d %s", sync.StatusCode, sync.ReadAllString())
	}
	jobs, err := admin.Get(context.Background(), prefix+"/api/v1/admin/jobs?bindingId="+id)
	if err != nil {
		t.Fatal(err)
	}
	defer jobs.Close()
	jobsBody := jobs.ReadAllString()
	if jobs.StatusCode != http.StatusOK {
		t.Fatalf("jobs %d %s", jobs.StatusCode, jobsBody)
	}
	var jobsEnv struct {
		OK   bool `json:"ok"`
		Data struct {
			Total int `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(jobsBody), &jobsEnv); err != nil {
		t.Fatal(err)
	}
	if jobsEnv.Data.Total < 2 {
		t.Fatalf("job history want >=2 got %s", jobsBody)
	}
	table, err := admin.Get(context.Background(), prefix+"/api/v1/admin/tables/binding")
	if err != nil {
		t.Fatal(err)
	}
	defer table.Close()
	tableBody := table.ReadAllString()
	if table.StatusCode != http.StatusOK {
		t.Fatalf("table %d %s", table.StatusCode, tableBody)
	}
	if strings.Contains(tableBody, `"vin_cipher"`) || strings.Contains(tableBody, "TESTVIN0000000001") {
		t.Fatal("table must redact")
	}
	kick, err := admin.ContentJson().Post(context.Background(), prefix+"/api/v1/admin/bindings/"+id+"/kick", "{}")
	if err != nil {
		t.Fatal(err)
	}
	defer kick.Close()
	if kick.StatusCode != http.StatusOK {
		t.Fatalf("kick %d %s", kick.StatusCode, kick.ReadAllString())
	}
	put, err := admin.ContentJson().Put(context.Background(), prefix+"/api/v1/admin/settings", `{"cronSync":"off","corsOrigins":"http://127.0.0.1:5173","snapshotKeep":0,"staleAfterSec":7200}`)
	if err != nil {
		t.Fatal(err)
	}
	defer put.Close()
	if put.StatusCode != http.StatusOK {
		t.Fatalf("settings %d %s", put.StatusCode, put.ReadAllString())
	}
	acc, err := admin.Get(context.Background(), prefix+"/api/v1/admin/account")
	if err != nil {
		t.Fatal(err)
	}
	defer acc.Close()
	accBody := acc.ReadAllString()
	if acc.StatusCode != http.StatusOK || !strings.Contains(accBody, `"username":"admin"`) {
		t.Fatalf("account %d %s", acc.StatusCode, accBody)
	}
}
