package neta

import (
	"math"
	"sort"

	"fenghuolun/internal/clock"
)

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

const (
	FillCharge = "charge"
	FillRefuel = "refuel"
	FillAuto   = "auto"
	FillManual = "manual"
	FillDraft  = "draft"
	FillRecorded = "recorded"
)

const (
	chargeSocMin  = 2.0
	capacitySocMin = 5.0
	refuelPctMin  = 0.5
)

type Fill struct {
	ID         string         `json:"id"`
	Kind       string         `json:"kind"`
	Source     string         `json:"source"`
	Status     string         `json:"status"`
	StartedAt  clock.Instant  `json:"startedAt"`
	FinishedAt clock.Instant  `json:"finishedAt"`
	FromFetched clock.Instant `json:"fromFetched"`
	ToFetched   clock.Instant `json:"toFetched"`
	OdoStart   *float64       `json:"odoStart"`
	OdoEnd     *float64       `json:"odoEnd"`
	SocStart   *float64       `json:"socStart"`
	SocEnd     *float64       `json:"socEnd"`
	FuelStart  *float64       `json:"fuelStart"`
	FuelEnd    *float64       `json:"fuelEnd"`
	EnergyKwh  *float64       `json:"energyKwh"`
	Liters     *float64       `json:"liters"`
	PaidCny    *float64       `json:"paidCny"`
	UnitCny    *float64       `json:"unitCny"`
	Note       string         `json:"note"`
}

type FillSpend struct {
	ChargePaidCny    *float64 `json:"chargePaidCny"`
	RefuelPaidCny    *float64 `json:"refuelPaidCny"`
	ElecCnyPerKwh    *float64 `json:"elecCnyPerKwh"`
	FuelCnyPerL      *float64 `json:"fuelCnyPerL"`
	ElecUseCostCny   *float64 `json:"elecUseCostCny"`
}

type PackCapacity struct {
	Kwh     *float64 `json:"kwh"`
	Samples int      `json:"samples"`
	Note    string   `json:"note"`
}

func DetectFills(snaps []Snapshot) []Fill {
	out := []Fill{}
	out = append(out, mergeSteps(snaps, isChargeStep, FillCharge)...)
	out = append(out, mergeSteps(snaps, isRefuelStep, FillRefuel)...)
	return out
}

func mergeSteps(snaps []Snapshot, step func(Snapshot, Snapshot) bool, kind string) []Fill {
	out := []Fill{}
	var cur *Fill
	for i := 1; i < len(snaps); i++ {
		prev, now := snaps[i-1], snaps[i]
		if !step(prev, now) {
			if cur != nil {
				out = append(out, *cur)
				cur = nil
			}
			continue
		}
		if cur == nil {
			f := startFill(kind, prev, now)
			cur = &f
			continue
		}
		extendFill(cur, now)
	}
	if cur != nil {
		out = append(out, *cur)
	}
	return out
}

func startFill(kind string, prev, now Snapshot) Fill {
	f := Fill{
		Kind:        kind,
		Source:      FillAuto,
		Status:      FillDraft,
		FromFetched: prev.FetchedAt,
		ToFetched:   now.FetchedAt,
		StartedAt:   snapTime(prev),
		FinishedAt:  snapTime(now),
		OdoStart:    prev.OdometerKm,
		OdoEnd:      now.OdometerKm,
		SocStart:    prev.Power.SocPct,
		SocEnd:      now.Power.SocPct,
		FuelStart:   fuelPct(prev),
		FuelEnd:     fuelPct(now),
	}
	return f
}

func extendFill(f *Fill, now Snapshot) {
	f.ToFetched = now.FetchedAt
	f.FinishedAt = snapTime(now)
	f.OdoEnd = now.OdometerKm
	f.SocEnd = now.Power.SocPct
	f.FuelEnd = fuelPct(now)
}

func isChargeStep(prev, now Snapshot) bool {
	if prev.Power.SocPct == nil || now.Power.SocPct == nil {
		return false
	}
	if *now.Power.SocPct-*prev.Power.SocPct < chargeSocMin {
		return false
	}
	return plugEvidence(prev) || plugEvidence(now)
}

func isRefuelStep(prev, now Snapshot) bool {
	a, b := fuelPct(prev), fuelPct(now)
	if a == nil || b == nil {
		return false
	}
	return *b-*a > refuelPctMin
}

func plugEvidence(s Snapshot) bool {
	if s.Power.PluggedIn != nil && *s.Power.PluggedIn {
		return true
	}
	switch s.Power.ChargeStatus {
	case "charging", "complete", "idle":
		return true
	}
	return false
}

func fuelPct(s Snapshot) *float64 {
	if s.Extender == nil {
		return nil
	}
	return s.Extender.FuelPct
}

func snapTime(s Snapshot) clock.Instant {
	if s.ReportedAt != nil && !s.ReportedAt.IsZero() {
		return *s.ReportedAt
	}
	return s.FetchedAt
}

func AnnotateFill(f Fill) Fill {
	if f.PaidCny != nil && f.Kind == FillCharge && f.EnergyKwh != nil && *f.EnergyKwh > 0 {
		v := round2(*f.PaidCny / *f.EnergyKwh)
		f.UnitCny = &v
	}
	if f.PaidCny != nil && f.Kind == FillRefuel && f.Liters != nil && *f.Liters > 0 {
		v := round2(*f.PaidCny / *f.Liters)
		f.UnitCny = &v
	}
	if f.PaidCny != nil {
		f.Status = FillRecorded
	} else if f.Status == "" {
		f.Status = FillDraft
	}
	return f
}

func FillSpendOf(fills []Fill, officialKwh *float64) FillSpend {
	var charge, refuel, kwh, liters float64
	var nCharge, nRefuel, nKwh, nL int
	for _, f := range fills {
		if f.PaidCny == nil {
			continue
		}
		switch f.Kind {
		case FillCharge:
			charge += *f.PaidCny
			nCharge++
			if f.EnergyKwh != nil && *f.EnergyKwh > 0 {
				kwh += *f.EnergyKwh
				nKwh++
			}
		case FillRefuel:
			refuel += *f.PaidCny
			nRefuel++
			if f.Liters != nil && *f.Liters > 0 {
				liters += *f.Liters
				nL++
			}
		}
	}
	out := FillSpend{}
	if nCharge > 0 {
		v := round2(charge)
		out.ChargePaidCny = &v
	}
	if nRefuel > 0 {
		v := round2(refuel)
		out.RefuelPaidCny = &v
	}
	if nKwh > 0 && kwh > 0 {
		paidWithKwh := 0.0
		for _, f := range fills {
			if f.Kind == FillCharge && f.PaidCny != nil && f.EnergyKwh != nil && *f.EnergyKwh > 0 {
				paidWithKwh += *f.PaidCny
			}
		}
		v := round2(paidWithKwh / kwh)
		out.ElecCnyPerKwh = &v
		if officialKwh != nil && *officialKwh > 0 {
			c := round2(*officialKwh * v)
			out.ElecUseCostCny = &c
		}
	}
	if nL > 0 && liters > 0 {
		paid := 0.0
		for _, f := range fills {
			if f.Kind == FillRefuel && f.PaidCny != nil && f.Liters != nil && *f.Liters > 0 {
				paid += *f.PaidCny
			}
		}
		v := round2(paid / liters)
		out.FuelCnyPerL = &v
	}
	return out
}

func PackCapacityOf(fills []Fill) PackCapacity {
	samples := []float64{}
	for _, f := range fills {
		if f.Kind != FillCharge || f.EnergyKwh == nil || *f.EnergyKwh <= 0 {
			continue
		}
		if f.SocStart == nil || f.SocEnd == nil {
			continue
		}
		d := *f.SocEnd - *f.SocStart
		if d < capacitySocMin {
			continue
		}
		samples = append(samples, *f.EnergyKwh/d*100)
	}
	out := PackCapacity{Samples: len(samples), Note: "没有填度数的充电不算容量。枪上度数含损耗，会比电池包略偏大。"}
	if len(samples) == 0 {
		out.Note = "有充电记录并填了度数、起止电量差≥5% 才反推。"
		return out
	}
	sort.Float64s(samples)
	v := round2(median(samples))
	out.Kwh = &v
	out.Note = "约 " + trimCap(v) + " kWh（根据 " + itoa(len(samples)) + " 次你填的充电电量反推）。枪上度数含损耗。"
	return out
}

func median(s []float64) float64 {
	n := len(s)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func trimCap(v float64) string {
	s := format1(v)
	return s
}

func format1(v float64) string {
	r := math.Round(v*10) / 10
	ip := int(r)
	frac := int(math.Round((r - float64(ip)) * 10))
	if frac < 0 {
		frac = -frac
	}
	return itoa(ip) + "." + itoa(frac)
}
