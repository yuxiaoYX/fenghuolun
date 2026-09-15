package owner

import (
	"context"
	"strings"
	"time"

	"github.com/gogf/gf/v2/os/gcron"

	"fenghuolun/internal/config"
	"fenghuolun/internal/store"
)

const cronEntry = "fenghuolun-readonly-sync"

func StartCron(ctx context.Context, cfg config.Config, svc *Service) error {
	raw := cfg.CronSync
	if svc != nil && svc.Store != nil {
		if v := svc.Store.GetSetting(store.SettingCronSync); v != "" {
			raw = v
		}
	}
	return svc.ApplyCron(ctx, raw)
}

func (s *Service) ApplyCron(ctx context.Context, raw string) error {
	gcron.Remove(cronEntry)
	pattern := cronPattern(raw)
	if pattern == "" {
		return nil
	}
	_, err := gcron.AddSingleton(ctx, pattern, func(ctx context.Context) {
		s.SyncAllActive()
	}, cronEntry)
	return err
}

func cronPattern(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "0" || strings.EqualFold(raw, "off") {
		return ""
	}
	if strings.HasPrefix(raw, "@") {
		return raw
	}
	if _, err := time.ParseDuration(raw); err == nil {
		return "@every " + raw
	}
	return raw
}
