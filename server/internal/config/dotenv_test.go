package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvSkipsExisting(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".env")
	if err := os.WriteFile(p, []byte("FENGHUOLUN_DOTENV_TEST_A=fromfile\nFENGHUOLUN_DOTENV_TEST_B=fileb\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FENGHUOLUN_DOTENV_TEST_A", "already")
	_ = os.Unsetenv("FENGHUOLUN_DOTENV_TEST_B")
	if err := loadDotEnv(p); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("FENGHUOLUN_DOTENV_TEST_A") != "already" {
		t.Fatal("must not override process env")
	}
	if os.Getenv("FENGHUOLUN_DOTENV_TEST_B") != "fileb" {
		t.Fatal("should set missing key")
	}
}

func TestLoadDotEnvMissingOK(t *testing.T) {
	if err := loadDotEnv(filepath.Join(t.TempDir(), "nope.env")); err != nil {
		t.Fatal(err)
	}
}
