package store

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"fenghuolun/internal/config"
	"fenghuolun/internal/model/do"
)

const (
	SettingCronSync     = "cron_sync"
	SettingCORS         = "cors_origins"
	SettingSnapshotKeep = "snapshot_keep"
	SettingStaleAfter   = "stale_after_sec"

	DefaultStaleAfterSec = 7200
)

type Settings struct {
	CronSync      string `json:"cronSync"`
	CORSOrigins   string `json:"corsOrigins"`
	SnapshotKeep  int    `json:"snapshotKeep"`
	StaleAfterSec int    `json:"staleAfterSec"`
}

func (s *SQLite) GetSetting(key string) string {
	if s == nil || key == "" {
		return ""
	}
	v, err := s.db.Model("app_settings").Ctx(s.ctx()).Where("key", key).Value("value")
	if err != nil || v.IsEmpty() {
		return ""
	}
	return v.String()
}

func (s *SQLite) SetSetting(key, value string) error {
	if key == "" {
		return fmt.Errorf("invalid_request: missing setting key")
	}
	_, err := s.db.Model("app_settings").Ctx(s.ctx()).Data(do.AppSetting{
		Key:   key,
		Value: value,
	}).OnConflict("key").Save()
	return err
}

func (s *SQLite) SeedSettings(cfg config.Config) error {
	defaults := map[string]string{
		SettingCronSync:     strings.TrimSpace(cfg.CronSync),
		SettingCORS:         strings.TrimSpace(cfg.CORSOrigins),
		SettingSnapshotKeep: "0",
		SettingStaleAfter:   strconv.Itoa(DefaultStaleAfterSec),
	}
	for key, val := range defaults {
		if s.GetSetting(key) != "" {
			continue
		}
		if err := s.SetSetting(key, val); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLite) Settings() Settings {
	stale := atoiDefault(s.GetSetting(SettingStaleAfter), DefaultStaleAfterSec)
	if stale < 60 {
		stale = DefaultStaleAfterSec
	}
	keep := atoiDefault(s.GetSetting(SettingSnapshotKeep), 0)
	if keep < 0 {
		keep = 0
	}
	return Settings{
		CronSync:      s.GetSetting(SettingCronSync),
		CORSOrigins:   s.GetSetting(SettingCORS),
		SnapshotKeep:  keep,
		StaleAfterSec: stale,
	}
}

func (s *SQLite) SaveSettings(in Settings) (Settings, error) {
	if in.SnapshotKeep < 0 {
		return Settings{}, fmt.Errorf("invalid_request: snapshotKeep")
	}
	stale := in.StaleAfterSec
	if stale == 0 {
		stale = DefaultStaleAfterSec
	}
	if stale < 60 {
		return Settings{}, fmt.Errorf("invalid_request: staleAfterSec")
	}
	pairs := map[string]string{
		SettingCronSync:     strings.TrimSpace(in.CronSync),
		SettingCORS:         strings.TrimSpace(in.CORSOrigins),
		SettingSnapshotKeep: strconv.Itoa(in.SnapshotKeep),
		SettingStaleAfter:   strconv.Itoa(stale),
	}
	for key, val := range pairs {
		if err := s.SetSetting(key, val); err != nil {
			return Settings{}, err
		}
	}
	return s.Settings(), nil
}

func (s *SQLite) SnapshotKeep() int {
	return s.Settings().SnapshotKeep
}

func (s *SQLite) StaleAfter() time.Duration {
	return time.Duration(s.Settings().StaleAfterSec) * time.Second
}

func atoiDefault(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}
