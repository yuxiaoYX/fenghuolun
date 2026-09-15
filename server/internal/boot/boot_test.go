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
