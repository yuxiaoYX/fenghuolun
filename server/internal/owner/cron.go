package owner

import (
	"context"
	"strings"
	"time"

	"github.com/gogf/gf/v2/os/gcron"

	"fenghuolun/internal/config"
)

func StartCron(ctx context.Context, cfg config.Config, svc *Service) error {
	pattern := cronPattern(cfg.CronSync)
	if pattern == "" {
		return nil
	}
	_, err := gcron.AddSingleton(ctx, pattern, func(ctx context.Context) {
		svc.SyncAllActive()
	}, "fenghuolun-readonly-sync")
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
