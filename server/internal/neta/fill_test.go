package neta

import (
	"testing"
	"time"

	"fenghuolun/internal/clock"
)

func TestDetectFillsChargeNeedsPlug(t *testing.T) {
	t0 := clock.Of(time.Unix(1, 0).UTC())
	t1 := clock.Of(time.Unix(2, 0).UTC())
	a := Snapshot{FetchedAt: t0, Power: Power{SocPct: ptr(40)}}
	b := Snapshot{FetchedAt: t1, Power: Power{SocPct: ptr(70)}}
	if n := DetectFills([]Snapshot{a, b}); len(n) != 0 {
		t.Fatalf("soc jump without plug: %+v", n)
	}
	yes := true
	b.Power.PluggedIn = &yes
	got := DetectFills([]Snapshot{a, b})
	if len(got) != 1 || got[0].Kind != FillCharge {
		t.Fatalf("%+v", got)
	}
}

func TestDetectFillsMergeChargeAndRefuel(t *testing.T) {
	yes := true
	s := []Snapshot{
		{FetchedAt: clock.Of(time.Unix(1, 0).UTC()), Power: Power{SocPct: ptr(40), PluggedIn: &yes}, Extender: &Extender{FuelPct: ptr(20)}},
		{FetchedAt: clock.Of(time.Unix(2, 0).UTC()), Power: Power{SocPct: ptr(55), PluggedIn: &yes}, Extender: &Extender{FuelPct: ptr(20)}},
		{FetchedAt: clock.Of(time.Unix(3, 0).UTC()), Power: Power{SocPct: ptr(80), ChargeStatus: "complete"}, Extender: &Extender{FuelPct: ptr(20)}},
		{FetchedAt: clock.Of(time.Unix(4, 0).UTC()), Power: Power{SocPct: ptr(78)}, Extender: &Extender{FuelPct: ptr(45)}},
	}
	got := DetectFills(s)
	var ch, rf int
	for _, f := range got {
		if f.Kind == FillCharge {
			ch++
			if f.SocStart == nil || *f.SocStart != 40 || f.SocEnd == nil || *f.SocEnd != 80 {
				t.Fatalf("merged charge %+v", f)
			}
		}
		if f.Kind == FillRefuel {
			rf++
		}
	}
	if ch != 1 || rf != 1 {
		t.Fatalf("ch=%d rf=%d %+v", ch, rf, got)
	}
}

func TestPackCapacityAndSpend(t *testing.T) {
	paid := 48.0
	kwh := 80.0
	fills := []Fill{
		AnnotateFill(Fill{Kind: FillCharge, PaidCny: &paid, EnergyKwh: &kwh, SocStart: ptr(10), SocEnd: ptr(90)}),
	}
	cap := PackCapacityOf(fills)
	if cap.Kwh == nil || *cap.Kwh != 100 {
		t.Fatalf("cap %+v", cap)
	}
	off := 10.0
	sp := FillSpendOf(fills, &off)
	if sp.ChargePaidCny == nil || *sp.ChargePaidCny != 48 {
		t.Fatalf("spend %+v", sp)
	}
	if sp.ElecCnyPerKwh == nil || *sp.ElecCnyPerKwh != 0.6 {
		t.Fatalf("unit %+v", sp)
	}
	if sp.ElecUseCostCny == nil || *sp.ElecUseCostCny != 6 {
		t.Fatalf("use %+v", sp)
	}
}
