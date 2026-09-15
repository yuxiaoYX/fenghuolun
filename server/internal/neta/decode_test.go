package neta

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"fenghuolun/internal/clock"
)

func testdata(t *testing.T, name string) []byte {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	p := filepath.Join(filepath.Dir(file), "..", "..", "..", "testdata", "neta", name)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestMaskVIN(t *testing.T) {
	if MaskVIN("") != "" {
		t.Fatal("empty")
	}
	if MaskVIN("ABCD") != "****" {
		t.Fatal("short")
	}
	if MaskVIN("TESTVIN0000000001") != "****0001" {
		t.Fatalf("got %q", MaskVIN("TESTVIN0000000001"))
	}
}

func TestDecodeCurrentVehicle(t *testing.T) {
	meta, err := DecodeCurrentVehicle(testdata(t, "getCurrentVehicle.sample.json"))
	if err != nil {
		t.Fatal(err)
	}
	if meta.ModelCode != "EP32" || meta.VinMasked != "****0001" {
		t.Fatalf("%+v", meta)
	}
	if meta.VIN == "" {
		t.Fatal("decoder keeps raw VIN internally")
	}
}

func TestDecodeHidesScaleByDefault(t *testing.T) {
	meta, _ := DecodeCurrentVehicle(testdata(t, "getCurrentVehicle.sample.json"))
	snap, err := DecodeVehicleData(testdata(t, "getAppVehicleData.sample.json"), meta, time.Unix(0, 0).UTC(), false)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Power.SocPct == nil || *snap.Power.SocPct != 50 {
		t.Fatalf("soc %+v", snap.Power.SocPct)
	}
	if snap.Power.EvRangeKm == nil || *snap.Power.EvRangeKm != 127 {
		t.Fatalf("ev %+v", snap.Power.EvRangeKm)
	}
	// 电压 ÷10 已证实（量纲唯一 + 冗余互验 + 国标惯例），默认即输出，不受 candidate 闸门控制。
	if snap.Battery12V.Volts == nil || *snap.Battery12V.Volts != 12.7 {
		t.Fatalf("12v %+v", snap.Battery12V.Volts)
	}
	if snap.Battery.PackVoltageV == nil || *snap.Battery.PackVoltageV != 354.5 {
		t.Fatalf("pack %+v", snap.Battery.PackVoltageV)
	}
	// 电流仍是强候选：输出 null，但 warning 标明 GB/T 32960 先验，等三态路试定案。
	if snap.Battery.CurrentA != nil {
		t.Fatalf("current must stay null until road-test: %+v", snap.Battery.CurrentA)
	}
	hasCurrentWarn := false
	for _, w := range snap.DecodeWarnings {
		if w == "packCurrentA:scale_candidate:gbt32960_offset_-1000_res_0.1" {
			hasCurrentWarn = true
		}
	}
	if !hasCurrentWarn {
		t.Fatalf("missing current candidate warning: %+v", snap.DecodeWarnings)
	}
	if snap.OdometerKm == nil || *snap.OdometerKm != 1234.5 {
		t.Fatalf("odo %+v", snap.OdometerKm)
	}
	// 胎压保留 2 位（截断）：官方 App 显示时截到 1 位，解码层不提前丢精度。
	if snap.Tires.FL.Bar == nil || *snap.Tires.FL.Bar != 2.45 {
		t.Fatalf("fl bar %+v", snap.Tires.FL.Bar)
	}
	if snap.Tires.FR.Bar == nil || *snap.Tires.FR.Bar != 2.47 {
		t.Fatalf("fr bar %+v", snap.Tires.FR.Bar)
	}
	if snap.Tires.RL.Bar == nil || *snap.Tires.RL.Bar != 2.41 {
		t.Fatalf("rl bar %+v", snap.Tires.RL.Bar)
	}
	if snap.Tires.FL.TempC == nil || *snap.Tires.FL.TempC != 23 {
		t.Fatalf("fl temp %+v", snap.Tires.FL.TempC)
	}
	if snap.Tires.RR.Bar == nil || *snap.Tires.RR.Bar != 2.5 {
		t.Fatalf("rr bar %+v", snap.Tires.RR.Bar)
	}
	if snap.Climate.CabinC == nil || *snap.Climate.CabinC != 23.0 {
		t.Fatalf("cabin %+v", snap.Climate.CabinC)
	}
	if snap.Climate.OutsideC == nil || *snap.Climate.OutsideC != 24.5 {
		t.Fatalf("outside %+v", snap.Climate.OutsideC)
	}
	if snap.ReportedAt == nil {
		t.Fatal("reportedAt")
	}
	if snap.Extender == nil || snap.Extender.FuelPct == nil || *snap.Extender.FuelPct != 24 {
		t.Fatalf("fuel %+v", snap.Extender)
	}
	if snap.Power.ChargeStatus != "unknown" {
		t.Fatalf("charge %+v", snap.Power.ChargeStatus)
	}
	if snap.Body.Locked != "closed" || snap.Body.Doors.FL != "closed" || snap.Body.Windows.FL != "closed" || snap.Body.Windows.Sunroof != "closed" {
		t.Fatalf("body %+v", snap.Body)
	}
	// 默认 sample 无经纬度，位置应为空。
	if snap.Location.Lng != nil || snap.Location.Lat != nil {
		t.Fatalf("location should be empty for sample without coords: %+v", snap.Location)
	}
}

func TestDecodeLocation(t *testing.T) {
	// 内联假坐标（非真实车主数据），验证 ÷1e6 解码与 reportedAt 关联。
	raw := []byte(`{"code":20000,"data":{"vehicleExtend":{"lng":116000000,"lat":40000000},"vehicleConnection":{"reportTime":1767225600000}}}`)
	meta := VehicleMeta{VinMasked: "****0001"}
	snap, err := DecodeVehicleData(raw, meta, time.Now().UTC(), false)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Location.Lng == nil || *snap.Location.Lng != 116.0 {
		t.Fatalf("lng %+v", snap.Location.Lng)
	}
	if snap.Location.Lat == nil || *snap.Location.Lat != 40.0 {
		t.Fatalf("lat %+v", snap.Location.Lat)
	}
	if snap.Location.ReportedAt == nil {
		t.Fatal("location reportedAt")
	}
	// 零坐标应被丢弃。
	zero := []byte(`{"code":20000,"data":{"vehicleExtend":{"lng":0,"lat":0}}}`)
	snap2, err := DecodeVehicleData(zero, meta, time.Now().UTC(), false)
	if err != nil {
		t.Fatal(err)
	}
	if snap2.Location.Lng != nil || snap2.Location.Lat != nil {
		t.Fatalf("zero coords should be dropped: %+v", snap2.Location)
	}
}

func TestDecodeCandidateScale(t *testing.T) {
	meta, _ := DecodeCurrentVehicle(testdata(t, "getCurrentVehicle.sample.json"))
	snap, err := DecodeVehicleData(testdata(t, "getAppVehicleData.sample.json"), meta, time.Now(), true)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Power.EvRangeKm == nil || *snap.Power.EvRangeKm != 127 {
		t.Fatalf("ev %+v", snap.Power.EvRangeKm)
	}
	if snap.Battery12V.Volts == nil || *snap.Battery12V.Volts != 12.7 {
		t.Fatalf("12v %+v", snap.Battery12V.Volts)
	}
	if snap.Battery.PackVoltageV == nil || *snap.Battery.PackVoltageV != 354.5 {
		t.Fatalf("pack %+v", snap.Battery.PackVoltageV)
	}
}

func TestDecodeEnergy(t *testing.T) {
	stat, err := DecodeEnergyDays(testdata(t, "energyByVin.sample.json"), 1, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(stat.Days) != 1 || stat.Days[0].CountTime != "2026-09-02" {
		t.Fatalf("%+v", stat)
	}
	if stat.TotalKwh == nil || *stat.TotalKwh != 12.3 {
		t.Fatalf("total %+v", stat.TotalKwh)
	}
}

func TestSnapshotJSONChinaWallClock(t *testing.T) {
	var snap Snapshot
	if err := json.Unmarshal([]byte(`{"fetchedAt":"2026-01-15T04:00:00Z"}`), &snap); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(snap)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, `"fetchedAt":"2026-01-15 12:00:00"`) {
		t.Fatalf("want China wall clock in JSON, got %s", s)
	}
	if strings.Contains(s, "T04:00:00Z") {
		t.Fatalf("must not emit UTC RFC3339: %s", s)
	}
}

func TestFuelLedger(t *testing.T) {
	a := Snapshot{FetchedAt: clock.Of(time.Unix(1, 0).UTC()), Extender: &Extender{FuelPct: ptr(24)}}
	b := Snapshot{FetchedAt: clock.Of(time.Unix(2, 0).UTC()), Extender: &Extender{FuelPct: ptr(21)}}
	c := Snapshot{FetchedAt: clock.Of(time.Unix(3, 0).UTC()), Extender: &Extender{FuelPct: ptr(39)}}
	d := FuelLedger([]Snapshot{a, b, c})
	if len(d) != 2 || d[0].Kind != "burn" || d[1].Kind != "refuel" {
		t.Fatalf("%+v", d)
	}
}

func TestDecodeTokenPair(t *testing.T) {
	raw := []byte(`{"message":"","success":true,"code":20000,"data":{"access_token":"a","token_type":"bearer","refresh_token":"r","expires_in":604799}}`)
	pair, err := DecodeTokenPair(raw)
	if err != nil || pair.AccessToken != "a" || pair.ExpiresIn != 604799 {
		t.Fatalf("%+v %v", pair, err)
	}
}

func TestDecodeTokenExpired(t *testing.T) {
	raw := []byte(`{"message":"登录状态已过期，请重新登录","success":false,"code":41141,"data":null}`)
	if _, err := DecodeTokenPair(raw); err != ErrTokenInvalid {
		t.Fatalf("err %v", err)
	}
}

func ptr(v float64) *float64 { return &v }
