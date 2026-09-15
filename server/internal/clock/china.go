package clock

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "time/tzdata"
)

// Layout 对外 JSON / 界面用的北京时间，不含时区后缀。本产品只面向中国用户。
const Layout = "2006-01-02 15:04:05"

// China 是 Asia/Shanghai（UTC+8，含历史 DST 规则）。LoadLocation 失败时退回固定 +8。
var China *time.Location

func init() {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	China = loc
}

// Instant 表示一个绝对时刻（内部按 UTC 存）。JSON 编成北京时间 "YYYY-MM-DD HH:mm:ss"；
// 零值编成 null。反序列化同时接受该格式和旧的 RFC3339（含 Z），方便读历史快照 payload。
type Instant time.Time

func Of(t time.Time) Instant {
	if t.IsZero() {
		return Instant{}
	}
	return Instant(t.UTC())
}

func Ptr(t *time.Time) *Instant {
	if t == nil {
		return nil
	}
	v := Of(*t)
	return &v
}

func (i Instant) Time() time.Time {
	return time.Time(i)
}

func (i Instant) IsZero() bool {
	return time.Time(i).IsZero()
}

func (i Instant) Unix() int64 {
	return time.Time(i).Unix()
}

func (i Instant) UTC() time.Time {
	return time.Time(i).UTC()
}

func (i Instant) MarshalJSON() ([]byte, error) {
	t := time.Time(i)
	if t.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.In(China).Format(Layout))
}

func (i *Instant) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*i = Instant{}
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	t, err := Parse(s)
	if err != nil {
		return err
	}
	*i = Of(t)
	return nil
}

func Parse(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "null" || s == "unknown" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.ParseInLocation(Layout, s, China); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04", s, China); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("clock: cannot parse %q", s)
}
