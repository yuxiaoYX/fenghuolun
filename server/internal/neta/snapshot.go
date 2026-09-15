package neta

import (
	"time"

	"fenghuolun/internal/clock"
)

// StaleAfter：车辆上报时刻距现在超过此时长，快照 API 标 stale。
// 用来区分「刚刚同步成功」和「车已经很久没上报」。无 reportedAt 不标陈旧。
const StaleAfter = 2 * time.Hour

func (s Snapshot) Stale(now time.Time) bool {
	return s.StaleSince(now, StaleAfter)
}

func (s Snapshot) StaleSince(now time.Time, d time.Duration) bool {
	if d <= 0 {
		d = StaleAfter
	}
	if s.ReportedAt == nil || s.ReportedAt.IsZero() {
		return false
	}
	return now.Sub(s.ReportedAt.Time()) > d
}

type Snapshot struct {
	FetchedAt      clock.Instant  `json:"fetchedAt"`
	ReportedAt     *clock.Instant `json:"reportedAt"`
	DecodeWarnings []string       `json:"decodeWarnings"`
	VinMasked      string         `json:"vinMasked"`
	Nickname       string         `json:"nickname"`
	ModelCode      string         `json:"modelCode"`
	ModelName      string         `json:"modelName"`
	Trim           string         `json:"trim"`
	Power          Power          `json:"power"`
	Extender       *Extender      `json:"extender"`
	Body           Body           `json:"body"`
	Tires          Tires          `json:"tires"`
	Climate        Climate        `json:"climate"`
	Battery12V     Battery12V     `json:"battery12v"`
	Battery        Pack           `json:"battery"`
	Online         *bool          `json:"online"`
	OdometerKm     *float64       `json:"odometerKm"`
	Location       Location       `json:"location"`
}

// Location 车辆最新上报位置（WGS-84，÷1e6）。决策 D17 已修订：不再丢弃，
// 仅保留最新一个点，车主与管理员均可见。无有效坐标时 Lng/Lat 为 nil。
type Location struct {
	Lng        *float64       `json:"lng"`
	Lat        *float64       `json:"lat"`
	ReportedAt *clock.Instant `json:"reportedAt"`
}

type Power struct {
	SocPct        *float64 `json:"socPct"`
	EvRangeKm     *float64 `json:"evRangeKm"`
	FuelRangeKm   *float64 `json:"fuelRangeKm"`
	TotalRangeKm  *float64 `json:"totalRangeKm"`
	ChargeStatus  string   `json:"chargeStatus"`
	PluggedIn     *bool    `json:"pluggedIn"`
	ChargePowerKw *float64 `json:"chargePowerKw"`
}

type Extender struct {
	FuelPct     *float64 `json:"fuelPct"`
	FuelRangeKm *float64 `json:"fuelRangeKm"`
	Enabled     *bool    `json:"enabled"`
	Generating  *bool    `json:"generating"`
}

type Body struct {
	Locked  string    `json:"locked"`
	Doors   BodyDoors `json:"doors"`
	Windows BodyWins  `json:"windows"`
}

type BodyDoors struct {
	FL    string `json:"fl"`
	FR    string `json:"fr"`
	RL    string `json:"rl"`
	RR    string `json:"rr"`
	Hood  string `json:"hood"`
	Trunk string `json:"trunk"`
}

type BodyWins struct {
	FL      string `json:"fl"`
	FR      string `json:"fr"`
	RL      string `json:"rl"`
	RR      string `json:"rr"`
	Sunroof string `json:"sunroof"`
}

type TireCorner struct {
	Bar   *float64 `json:"bar"`
	TempC *float64 `json:"tempC"`
}

type Tires struct {
	Unit string     `json:"unit"`
	FL   TireCorner `json:"fl"`
	FR   TireCorner `json:"fr"`
	RL   TireCorner `json:"rl"`
	RR   TireCorner `json:"rr"`
}

type Climate struct {
	CabinC   *float64 `json:"cabinC"`
	OutsideC *float64 `json:"outsideC"`
}

type Battery12V struct {
	Volts *float64 `json:"volts"`
}

type Pack struct {
	PackVoltageV *float64 `json:"packVoltageV"`
	CurrentA     *float64 `json:"currentA"`
}

type VehicleMeta struct {
	VIN        string
	VinMasked  string
	Nickname   string
	ModelCode  string
	ModelName  string
	Trim       string
	IsExtender bool
}

type EnergyDay struct {
	CountTime    string   `json:"countTime"`
	TotalKwh     *float64 `json:"totalKwh"`
	DrivingKwh   *float64 `json:"drivingKwh"`
	AcKwh        *float64 `json:"acKwh"`
	RecoveryKwh  *float64 `json:"recoveryKwh"`
	WindKwh      *float64 `json:"windKwh"`
	AccessoryKwh *float64 `json:"accessoryKwh"`
}

type EnergyStat struct {
	PeriodType int           `json:"periodType"`
	Days       []EnergyDay   `json:"days"`
	TotalKwh   *float64      `json:"totalKwh"`
	FetchedAt  clock.Instant `json:"fetchedAt"`
}

type FuelDelta struct {
	FromAt       clock.Instant `json:"fromAt"`
	ToAt         clock.Instant `json:"toAt"`
	FuelPctFrom  *float64      `json:"fuelPctFrom"`
	FuelPctTo    *float64      `json:"fuelPctTo"`
	FuelPctDelta *float64      `json:"fuelPctDelta"`
	Kind         string        `json:"kind"`
	LitersEst    *float64      `json:"litersEst"`
}
