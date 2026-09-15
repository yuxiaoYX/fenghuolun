package neta

import (
	"encoding/json"
	"fmt"
	"time"

	"fenghuolun/internal/clock"
)

type rawVehicleData struct {
	VehicleBasic struct {
		Soc          *float64 `json:"soc"`
		Mileage      *float64 `json:"mileage"`
		TotalVoltage *float64 `json:"totalVoltage"`
		TotalCurrent *float64 `json:"totalCurrent"`
		ChargeStatus *float64 `json:"chargeStatus"`
	} `json:"vehicleBasic"`
	EnduranceStatus struct {
		PowerPercentage      *float64 `json:"powerPercentage"`
		PowerResidueMileage  *float64 `json:"powerResidueMileage"`
		FuelMileageRemaining *float64 `json:"fuelMileageRemaining"`
		EstimateFuelPercent  *float64 `json:"estimateFuelPercent"`
	} `json:"enduranceStatus"`
	VehicleExtend struct {
		Battery12Voltage *float64        `json:"battery12Voltage"`
		EngineStatus     *float64        `json:"engineStatus"`
		Lng              json.RawMessage `json:"lng"`
		Lat              json.RawMessage `json:"lat"`
	} `json:"vehicleExtend"`
	VehicleConnection struct {
		Online     *bool  `json:"online"`
		ReportTime *int64 `json:"reportTime"`
	} `json:"vehicleConnection"`
	ChargingStatus struct {
		PowerChargeStatus *float64 `json:"powerChargeStatus"`
	} `json:"chargingStatus"`
	EnergyConsumption struct {
		TodayEnergyConsumption *float64 `json:"todayEnergyConsumption"`
	} `json:"energyConsumption"`
	LockStatus struct {
		DriverSizeDoorLockStatus   *float64 `json:"driverSizeDoorLockStatus"`
		CoDriverSizeDoorLockStatus *float64 `json:"coDriverSizeDoorLockStatus"`
		LeftAfterDoorLockStatus    *float64 `json:"leftAfterDoorLockStatus"`
		RightAfterDoorLockStatus   *float64 `json:"rightAfterDoorLockStatus"`
		CarBoarLockStatus          *float64 `json:"carBoarLockStatus"`
		CoverFrontLockStatus       *float64 `json:"coverFrontLockStatus"`
	} `json:"lockStatus"`
	DoorCoverStatus struct {
		DriverSizeDoorStatus   *float64 `json:"driverSizeDoorStatus"`
		CoDriverSizeDoorStatus *float64 `json:"coDriverSizeDoorStatus"`
		LeftAfterDoorStatus    *float64 `json:"leftAfterDoorStatus"`
		RightAfterDoorStatus   *float64 `json:"rightAfterDoorStatus"`
		CoverFrontStatus       *float64 `json:"coverFrontStatus"`
		CarBoarStatus          *float64 `json:"carBoarStatus"`
	} `json:"doorCoverStatus"`
	WindowStatus struct {
		DriverSizeWinStatus     *float64 `json:"driverSizeWinStatus"`
		CoDriverSizeWinStatus   *float64 `json:"coDriverSizeWinStatus"`
		LeftAfterWinStatus      *float64 `json:"leftAfterWinStatus"`
		RightAfterWinLockStatus *float64 `json:"rightAfterWinLockStatus"`
		SkyWindowStatus         *float64 `json:"skyWindowStatus"`
	} `json:"windowStatus"`
	TyreStatus struct {
		TireLeftFrontPress  *float64 `json:"tireLeftFrontPress"`
		TireLeftFrontTemp   *float64 `json:"tireLeftFrontTemp"`
		TireRightFrontPress *float64 `json:"tireRightFrontPress"`
		TireRightFrontTemp  *float64 `json:"tireRightFrontTemp"`
		TireLeftAfterPress  *float64 `json:"tireLeftAfterPress"`
		TireLeftAfterTemp   *float64 `json:"tireLeftAfterTemp"`
		TireRightAfterPress *float64 `json:"tireRightAfterPress"`
		TireRightAfterTemp  *float64 `json:"tireRightAfterTemp"`
	} `json:"tyreStatus"`
	AirconditionStatus struct {
		AirInTemp  *float64 `json:"airInTemp"`
		AirOutTemp *float64 `json:"airOutTemp"`
	} `json:"airconditionStatus"`
}

type rawCurrentVehicle struct {
	VIN         string `json:"vin"`
	VehicleName string `json:"vehicleName"`
	ModelCode   string `json:"modelCode"`
	ModelName   string `json:"modelName"`
	MarketName  string `json:"marketName"`
	Config      string `json:"config"`
}

func DecodeCurrentVehicle(raw []byte) (VehicleMeta, error) {
	env, err := ParseEnvelope(raw)
	if err != nil {
		return VehicleMeta{}, err
	}
	if !env.OK() {
		return VehicleMeta{}, fmt.Errorf("%w: code %d", ErrUpstream, env.Code)
	}
	var cur rawCurrentVehicle
	if err := json.Unmarshal(env.Data, &cur); err != nil {
		return VehicleMeta{}, fmt.Errorf("%w: current vehicle: %v", ErrDecode, err)
	}
	name := cur.MarketName
	if name == "" {
		name = cur.ModelName
	}
	return VehicleMeta{
		VIN:        cur.VIN,
		VinMasked:  MaskVIN(cur.VIN),
		Nickname:   cur.VehicleName,
		ModelCode:  cur.ModelCode,
		ModelName:  name,
		Trim:       cur.Config,
		IsExtender: cur.ModelCode == "EP32" || cur.Config != "",
	}, nil
}

// applyCandidateScale 当前为 no-op：电压/12V 已证实（默认输出），电流仍始终输出 null
// （等三态路试）。保留该参数与 FENGHUOLUN_NETA_SCALE=candidate 开关，为未来其它候选缩放预留。
func DecodeVehicleData(raw []byte, meta VehicleMeta, fetchedAt time.Time, applyCandidateScale bool) (Snapshot, error) {
	_ = applyCandidateScale
	env, err := ParseEnvelope(raw)
	if err != nil {
		return Snapshot{}, err
	}
	if !env.OK() {
		return Snapshot{}, fmt.Errorf("%w: code %d", ErrUpstream, env.Code)
	}
	var src rawVehicleData
	if err := json.Unmarshal(env.Data, &src); err != nil {
		return Snapshot{}, fmt.Errorf("%w: vehicle data: %v", ErrDecode, err)
	}

	snap := Snapshot{
		FetchedAt:      clock.Of(fetchedAt),
		DecodeWarnings: nil,
		VinMasked:      meta.VinMasked,
		Nickname:       meta.Nickname,
		ModelCode:      meta.ModelCode,
		ModelName:      meta.ModelName,
		Trim:           meta.Trim,
		Power: Power{
			ChargeStatus: "unknown",
		},
		Body: Body{
			Locked: "unknown",
			Doors: BodyDoors{
				FL: "unknown", FR: "unknown", RL: "unknown", RR: "unknown",
				Hood: "unknown", Trunk: "unknown",
			},
			Windows: BodyWins{
				FL: "unknown", FR: "unknown", RL: "unknown", RR: "unknown",
				Sunroof: "unknown",
			},
		},
		Tires: Tires{Unit: "bar"},
	}
	if src.VehicleConnection.Online != nil {
		v := *src.VehicleConnection.Online
		snap.Online = &v
	}
	if src.VehicleConnection.ReportTime != nil && *src.VehicleConnection.ReportTime > 0 {
		t := clock.Of(time.UnixMilli(*src.VehicleConnection.ReportTime))
		snap.ReportedAt = &t
	}

	// 位置：WGS-84，原始值 ÷1e6。决策 D17 已修订——不再丢弃，保留最新一个点，
	// 车主与管理员均可见。仅当经纬度均为有限非哨兵且落在合理范围时才写入。
	if lng := finiteNonSentinel(rawFloat(src.VehicleExtend.Lng)); lng != nil {
		if lat := finiteNonSentinel(rawFloat(src.VehicleExtend.Lat)); lat != nil {
			lngDeg := *lng / 1e6
			latDeg := *lat / 1e6
			if lngDeg >= -180 && lngDeg <= 180 && latDeg >= -90 && latDeg <= 90 && (lngDeg != 0 || latDeg != 0) {
				snap.Location.Lng = &lngDeg
				snap.Location.Lat = &latDeg
				if snap.ReportedAt != nil {
					t := *snap.ReportedAt
					snap.Location.ReportedAt = &t
				}
			}
		}
	}

	if v := finiteNonSentinel(src.VehicleBasic.Soc); v != nil && *v >= 0 && *v <= 100 {
		snap.Power.SocPct = v
	}

	if v := finiteNonSentinel(src.EnduranceStatus.PowerResidueMileage); v != nil {
		km := *v / 10
		snap.Power.EvRangeKm = &km
	}
	if v := finiteNonSentinel(src.EnduranceStatus.FuelMileageRemaining); v != nil {
		km := *v / 10
		snap.Power.FuelRangeKm = &km
	}
	if snap.Power.EvRangeKm != nil && snap.Power.FuelRangeKm != nil {
		sum := *snap.Power.EvRangeKm + *snap.Power.FuelRangeKm
		snap.Power.TotalRangeKm = &sum
	}

	// 动力电池总电压：÷10 已证实。三重支撑——batVoltage[0] 与本字段冗余互验、
	// 量纲上只有 ÷10 落在 400V 级平台（÷1=3543V、÷100=35.4V 皆荒谬）、且与
	// GB/T 32960.3 电池总电压分辨率 0.1V 惯例一致。不再受 candidate 闸门控制。
	if v := finiteNonSentinel(src.VehicleBasic.TotalVoltage); v != nil {
		volts := *v / 10
		snap.Battery.PackVoltageV = &volts
	}
	// 总输出电流：强候选，仍输出 null。先验来自 GB/T 32960.3（分辨率 0.1A、
	// 偏移 −1000A，即 raw=10000 → 0.0A），但本协议族哨兵本就不统一
	// （65535/65534/255/10000 并存），10000 也可能是「未采集」哨兵；且不能用
	// 单样本未验证的 runMode/chargeStatus 去循环互证。等充电/行驶/能量回收三态
	// 路试定案（回收时 raw 应 <10000，可双向验证符号与分辨率）。
	if src.VehicleBasic.TotalCurrent != nil {
		snap.DecodeWarnings = append(snap.DecodeWarnings, "packCurrentA:scale_candidate:gbt32960_offset_-1000_res_0.1")
	}
	// 12V 小电瓶：÷10 已证实。量纲唯一可行（÷1=127V、÷100=1.27V 皆荒谬，
	// 只有 ÷10=12.7V 是正常静置电压）。不再受 candidate 闸门控制。
	if v := finiteNonSentinel(src.VehicleExtend.Battery12Voltage); v != nil {
		volts := *v / 10
		snap.Battery12V.Volts = &volts
	}

	ext := &Extender{}
	hasExt := false
	if v := finiteNonSentinel(src.EnduranceStatus.EstimateFuelPercent); v != nil {
		ext.FuelPct = v
		hasExt = true
	}
	if snap.Power.FuelRangeKm != nil {
		ext.FuelRangeKm = snap.Power.FuelRangeKm
		hasExt = true
	}
	if src.VehicleExtend.EngineStatus != nil {
		snap.DecodeWarnings = append(snap.DecodeWarnings, "extender.engineStatus:enum_unverified")
	}
	if hasExt || meta.IsExtender {
		snap.Extender = ext
	}

	if v := finiteNonSentinel(src.VehicleBasic.Mileage); v != nil && *v > 0 {
		km := *v / 10
		snap.OdometerKm = &km
	}

	decodeTire := func(press, temp *float64) TireCorner {
		c := TireCorner{}
		if p := finiteNonSentinel(press); p != nil {
			c.Bar = trunc2(*p / 55)
		}
		if t := finiteNonSentinel(temp); t != nil {
			deg := *t - 50
			c.TempC = &deg
		}
		return c
	}
	snap.Tires.FL = decodeTire(src.TyreStatus.TireLeftFrontPress, src.TyreStatus.TireLeftFrontTemp)
	snap.Tires.FR = decodeTire(src.TyreStatus.TireRightFrontPress, src.TyreStatus.TireRightFrontTemp)
	snap.Tires.RL = decodeTire(src.TyreStatus.TireLeftAfterPress, src.TyreStatus.TireLeftAfterTemp)
	snap.Tires.RR = decodeTire(src.TyreStatus.TireRightAfterPress, src.TyreStatus.TireRightAfterTemp)

	if v := finiteNonSentinel(src.AirconditionStatus.AirInTemp); v != nil {
		snap.Climate.CabinC = round1((*v - 110) / 2)
	}
	if v := finiteNonSentinel(src.AirconditionStatus.AirOutTemp); v != nil {
		snap.Climate.OutsideC = round1((*v - 110) / 2)
	}

	if src.VehicleBasic.ChargeStatus != nil || src.ChargingStatus.PowerChargeStatus != nil {
		snap.DecodeWarnings = append(snap.DecodeWarnings, "chargeStatus:enum_unverified")
	}

	snap.Body.Doors.FL = closedIfZero(src.DoorCoverStatus.DriverSizeDoorStatus)
	snap.Body.Doors.FR = closedIfZero(src.DoorCoverStatus.CoDriverSizeDoorStatus)
	snap.Body.Doors.RL = closedIfZero(src.DoorCoverStatus.LeftAfterDoorStatus)
	snap.Body.Doors.RR = closedIfZero(src.DoorCoverStatus.RightAfterDoorStatus)
	snap.Body.Doors.Hood = closedIfZero(src.DoorCoverStatus.CoverFrontStatus)
	snap.Body.Doors.Trunk = closedIfZero(src.DoorCoverStatus.CarBoarStatus)

	locks := []string{
		closedIfZero(src.LockStatus.DriverSizeDoorLockStatus),
		closedIfZero(src.LockStatus.CoDriverSizeDoorLockStatus),
		closedIfZero(src.LockStatus.LeftAfterDoorLockStatus),
		closedIfZero(src.LockStatus.RightAfterDoorLockStatus),
	}
	snap.Body.Locked = allClosedOrUnknown(locks)

	snap.Body.Windows.FL = closedIfWindow(src.WindowStatus.DriverSizeWinStatus)
	snap.Body.Windows.FR = closedIfWindow(src.WindowStatus.CoDriverSizeWinStatus)
	snap.Body.Windows.RL = closedIfWindow(src.WindowStatus.LeftAfterWinStatus)
	snap.Body.Windows.RR = closedIfWindow(src.WindowStatus.RightAfterWinLockStatus)
	snap.Body.Windows.Sunroof = closedIfZero(src.WindowStatus.SkyWindowStatus)
	if v := finiteNonSentinel(src.EnergyConsumption.TodayEnergyConsumption); v == nil && src.EnergyConsumption.TodayEnergyConsumption != nil {
		snap.DecodeWarnings = append(snap.DecodeWarnings, "todayEnergyConsumption:sentinel")
	}

	return snap, nil
}

func DecodeEnergyDays(raw []byte, periodType int, fetchedAt time.Time) (EnergyStat, error) {
	env, err := ParseEnvelope(raw)
	if err != nil {
		return EnergyStat{}, err
	}
	if !env.OK() {
		return EnergyStat{}, fmt.Errorf("%w: code %d", ErrUpstream, env.Code)
	}
	stat := EnergyStat{PeriodType: periodType, FetchedAt: clock.Of(fetchedAt), Days: []EnergyDay{}}

	var days []map[string]any
	if err := json.Unmarshal(env.Data, &days); err != nil {
		var one map[string]any
		if err2 := json.Unmarshal(env.Data, &one); err2 != nil {
			return EnergyStat{}, fmt.Errorf("%w: energy: %v", ErrDecode, err)
		}
		days = []map[string]any{one}
	}
	var sum float64
	var hasSum bool
	for _, row := range days {
		day := EnergyDay{
			CountTime:    asString(row["countTime"]),
			TotalKwh:     asFloat(row["totalConsumesEnergy"]),
			DrivingKwh:   asFloat(row["drivingConsumesEnergy"]),
			AcKwh:        asFloat(row["airConditionersConsumesEnergy"]),
			RecoveryKwh:  asFloat(row["recoveryConsumesEnergy"]),
			WindKwh:      asFloat(row["windResistanceConsumesEnergy"]),
			AccessoryKwh: asFloat(row["attachmentConsumesEnergy"]),
		}
		if day.CountTime == "" && day.TotalKwh == nil {
			continue
		}
		stat.Days = append(stat.Days, day)
		if day.TotalKwh != nil {
			sum += *day.TotalKwh
			hasSum = true
		}
	}
	if hasSum {
		stat.TotalKwh = &sum
	}
	return stat, nil
}

func FuelLedger(snaps []Snapshot) []FuelDelta {
	out := []FuelDelta{}
	for i := 1; i < len(snaps); i++ {
		prev := snaps[i-1]
		cur := snaps[i]
		var fromPct, toPct *float64
		if prev.Extender != nil {
			fromPct = prev.Extender.FuelPct
		}
		if cur.Extender != nil {
			toPct = cur.Extender.FuelPct
		}
		kind := "unknown"
		var delta *float64
		if fromPct != nil && toPct != nil {
			d := *toPct - *fromPct
			delta = &d
			if d > 0.5 {
				kind = "refuel"
			} else if d < -0.5 {
				kind = "burn"
			} else {
				kind = "flat"
			}
		}
		out = append(out, FuelDelta{
			FromAt:       prev.FetchedAt,
			ToAt:         cur.FetchedAt,
			FuelPctFrom:  fromPct,
			FuelPctTo:    toPct,
			FuelPctDelta: delta,
			Kind:         kind,
		})
	}
	return out
}

func MaskVIN(vin string) string {
	if vin == "" {
		return ""
	}
	if len(vin) <= 4 {
		return "****"
	}
	return "****" + vin[len(vin)-4:]
}

func round1(v float64) *float64 {
	x := float64(int(v*10+0.5)) / 10
	return &x
}

// trunc2 截断到 2 位小数。官方 App 显示胎压时截断到 1 位
// （135→2.4、136→2.4），解码层保留多一位精度，不做四舍五入。
func trunc2(v float64) *float64 {
	x := float64(int(v*100)) / 100
	return &x
}

func closedIfZero(v *float64) string {
	if v == nil {
		return "unknown"
	}
	if *v == 0 {
		return "closed"
	}
	return "unknown"
}

func closedIfWindow(v *float64) string {
	if v == nil {
		return "unknown"
	}
	if *v == 16 {
		return "closed"
	}
	return "unknown"
}

func allClosedOrUnknown(vals []string) string {
	if len(vals) == 0 {
		return "unknown"
	}
	for _, s := range vals {
		if s != "closed" {
			return "unknown"
		}
	}
	return "closed"
}

func finiteNonSentinel(v *float64) *float64 {
	if v == nil {
		return nil
	}
	n := *v
	if n == 65535 || n == 65534 || n == 255 {
		return nil
	}
	out := n
	return &out
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func asFloat(v any) *float64 {
	switch n := v.(type) {
	case float64:
		x := n
		return &x
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return nil
		}
		return &f
	case int:
		x := float64(n)
		return &x
	default:
		return nil
	}
}

// rawFloat 从 json.RawMessage 解出一个有限浮点数；非数字或解析失败返回 nil。
func rawFloat(raw json.RawMessage) *float64 {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil
	}
	return &f
}
