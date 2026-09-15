package store

import "fenghuolun/internal/config"

func OpenFromConfig(cfg config.Config) (*SQLite, error) {
	kek, err := MustKEK(cfg.TokenKEK)
	if err != nil {
		return nil, err
	}
	path := cfg.SQLitePath
	if path == "" {
		path = "./data/fenghuolun.db"
	}
	st, err := OpenSQLite(path, kek)
	if err != nil {
		return nil, err
	}
	if err := st.EnsureAdmin(cfg.AdminUser, cfg.AdminPassword); err != nil {
		_ = st.Close()
		return nil, err
	}
	return st, nil
}
