package update

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackupSQLite(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "fenghuolun.db")
	if err := os.WriteFile(src, []byte("sqlite"), 0o600); err != nil {
		t.Fatal(err)
	}
	dst, err := BackupSQLite(src)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "sqlite" {
		t.Fatalf("dst %s", b)
	}
}
