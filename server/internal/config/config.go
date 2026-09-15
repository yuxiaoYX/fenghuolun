package config

import "os"

type Config struct {
	HTTPAddr       string
	SQLitePath     string
	TokenKEK       string
	AdminUser      string
	AdminPassword  string
	CORSOrigins    string
	ScaleCandidate bool
	CronSync       string
}

func Load() Config {
	_ = loadDotEnv(".env")
	scale := os.Getenv("FENGHUOLUN_NETA_SCALE")
	return Config{
		HTTPAddr:       getenv("FENGHUOLUN_HTTP_ADDR", ":8088"),
		SQLitePath:     getenv("FENGHUOLUN_SQLITE_PATH", "./data/fenghuolun.db"),
		TokenKEK:       os.Getenv("FENGHUOLUN_TOKEN_KEK"),
		AdminUser:      os.Getenv("FENGHUOLUN_ADMIN_BOOTSTRAP_USER"),
		AdminPassword:  os.Getenv("FENGHUOLUN_ADMIN_BOOTSTRAP_PASSWORD"),
		CORSOrigins:    getenv("FENGHUOLUN_CORS_ORIGINS", ""),
		ScaleCandidate: scale == "candidate",
		CronSync:       os.Getenv("FENGHUOLUN_CRON_SYNC"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
