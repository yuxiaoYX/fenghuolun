package update

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func BackupSQLite(sqlitePath string) (string, error) {
	if sqlitePath == "" || sqlitePath == ":memory:" {
		return "", nil
	}
	src, err := filepath.Abs(sqlitePath)
	if err != nil {
		src = sqlitePath
	}
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	dir := filepath.Join(filepath.Dir(src), "backups")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	stamp := time.Now().Format("20060102150405")
	dst := filepath.Join(dir, "pre-update-"+stamp+".db")
	if err := copyFile(src, dst); err != nil {
		return "", err
	}
	for _, suf := range []string{"-wal", "-shm"} {
		side := src + suf
		if _, err := os.Stat(side); err == nil {
			_ = copyFile(side, dst+suf)
		}
	}
	return dst, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Sync(); err != nil {
		return err
	}
	st, err := in.Stat()
	if err == nil && st.Size() == 0 {
		return fmt.Errorf("copied empty sqlite file")
	}
	return nil
}
