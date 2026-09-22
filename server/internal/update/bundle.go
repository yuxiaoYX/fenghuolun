package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	checksumAssetName = "SHA256SUMS"
	maxBundleBytes    = 200 << 20
	maxBundleFile     = 80 << 20
)

// exitProcess is replaced by tests. A successful bundle update exits so
// Docker's restart policy (or systemd) starts the new files.
var exitProcess = os.Exit

var checkDownloadURL = validateDownloadURL

var bundleHTTP = &http.Client{
	Timeout: 10 * time.Minute,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		return checkDownloadURL(req.URL.String())
	},
}

func bundleAssetName(tag, goos, goarch string) string {
	return "fenghuolun_" + NormalizeTag(tag) + "_" + goos + "_" + goarch + ".tar.gz"
}

func selectBundle(tag string, assets []ReleaseAsset, goos, goarch string) (file, sums ReleaseAsset, ok bool) {
	want := bundleAssetName(tag, goos, goarch)
	var fileOK, sumOK bool
	for _, a := range assets {
		if a.Name == want && a.URL != "" {
			file = a
			fileOK = true
		}
		if a.Name == checksumAssetName && a.URL != "" {
			sums = a
			sumOK = true
		}
	}
	return file, sums, fileOK && sumOK
}

func validateDownloadURL(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if !strings.HasPrefix(rawURL, "https://") {
		return fmt.Errorf("refusing non-https download")
	}
	host := rawURL[len("https://"):]
	if i := strings.IndexAny(host, "/?#"); i >= 0 {
		host = host[:i]
	}
	if strings.Contains(host, "@") {
		return fmt.Errorf("refusing download url with userinfo")
	}
	if i := strings.Index(host, ":"); i >= 0 {
		host = host[:i]
	}
	host = strings.ToLower(host)
	if host == "github.com" || strings.HasSuffix(host, ".github.com") ||
		host == "githubusercontent.com" || strings.HasSuffix(host, ".githubusercontent.com") {
		return nil
	}
	return fmt.Errorf("refusing download host %s", host)
}

func downloadReleaseFile(ctx context.Context, rawURL, dest string) error {
	if err := checkDownloadURL(rawURL); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "fenghuolun")
	req.Header.Set("Accept", "application/octet-stream")
	resp, err := bundleHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("download %s: %s", resp.Status, truncate(string(body), 180))
	}
	if resp.ContentLength > maxBundleBytes {
		return fmt.Errorf("download larger than %d bytes", maxBundleBytes)
	}
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(f, io.LimitReader(resp.Body, maxBundleBytes+1))
	closeErr := f.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if n > maxBundleBytes {
		return fmt.Errorf("download larger than %d bytes", maxBundleBytes)
	}
	return nil
}

func verifySHA256File(path, sumsText, filename string) error {
	want, ok := checksumFor(sumsText, filename)
	if !ok {
		return fmt.Errorf("checksums do not list %s", filename)
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("checksum mismatch for %s", filename)
	}
	return nil
}

func checksumFor(sumsText, filename string) (string, bool) {
	filename = filepath.Base(filename)
	for _, line := range strings.Split(sumsText, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[len(fields)-1], "*")
		if filepath.Base(name) != filename {
			continue
		}
		sum := fields[0]
		if len(sum) != 64 {
			return "", false
		}
		if _, err := hex.DecodeString(sum); err != nil {
			return "", false
		}
		return strings.ToLower(sum), true
	}
	return "", false
}

func extractBundle(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var total int64
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name, err := safeBundlePath(hdr.Name)
		if err != nil {
			return err
		}
		if name == "" || name == "." {
			continue
		}
		target := filepath.Join(dest, name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if hdr.Size < 0 || hdr.Size > maxBundleFile {
				return fmt.Errorf("file too large: %s", name)
			}
			total += hdr.Size
			if total > maxBundleBytes {
				return fmt.Errorf("bundle too large")
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
			if err != nil {
				return err
			}
			n, copyErr := io.Copy(out, io.LimitReader(tr, hdr.Size))
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
			if n != hdr.Size {
				return fmt.Errorf("short read %s", name)
			}
		default:
			return fmt.Errorf("unsupported tar entry %s", name)
		}
	}
	return nil
}

func safeBundlePath(name string) (string, error) {
	name = strings.TrimPrefix(filepath.ToSlash(name), "./")
	if name == "" || name == "." {
		return "", nil
	}
	if strings.HasPrefix(name, "/") || strings.Contains(name, "\x00") {
		return "", fmt.Errorf("unsafe path %s", name)
	}
	clean := filepath.Clean(filepath.FromSlash(name))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) || filepath.IsAbs(clean) {
		return "", fmt.Errorf("unsafe path %s", name)
	}
	return clean, nil
}

func bundleLayoutOK(dir string) error {
	bin := filepath.Join(dir, "fenghuolun")
	info, err := os.Stat(bin)
	if err != nil || info.IsDir() {
		return fmt.Errorf("bundle missing fenghuolun")
	}
	for _, p := range []string{
		filepath.Join(dir, "admin", "index.html"),
		filepath.Join(dir, "owner", "index.html"),
	} {
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("bundle missing %s", filepath.Base(filepath.Dir(p))+"/index.html")
		}
	}
	return nil
}

func installBundleDir(dataDir, staging string) error {
	root := filepath.Join(dataDir, "app")
	current := filepath.Join(root, "current")
	previous := filepath.Join(root, "previous")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	_ = os.RemoveAll(previous)
	hadCurrent := false
	if _, err := os.Stat(current); err == nil {
		hadCurrent = true
		if err := os.Rename(current, previous); err != nil {
			return fmt.Errorf("move current aside: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(staging, current); err != nil {
		if hadCurrent {
			_ = os.Rename(previous, current)
		}
		return fmt.Errorf("activate bundle: %w", err)
	}
	return nil
}

func applyBundle(ctx context.Context, dataDir string, rel Release, goos, goarch string) error {
	file, sums, ok := selectBundle(rel.Tag, rel.Assets, goos, goarch)
	if !ok {
		return fmt.Errorf("release %s has no %s", rel.Tag, bundleAssetName(rel.Tag, goos, goarch))
	}
	work := filepath.Join(dataDir, "app", ".download")
	if err := os.MkdirAll(work, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(work)
	archivePath := filepath.Join(work, file.Name)
	sumsPath := filepath.Join(work, checksumAssetName)
	if err := downloadReleaseFile(ctx, file.URL, archivePath); err != nil {
		return fmt.Errorf("download bundle: %w", err)
	}
	if err := downloadReleaseFile(ctx, sums.URL, sumsPath); err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}
	sumText, err := os.ReadFile(sumsPath)
	if err != nil {
		return err
	}
	if err := verifySHA256File(archivePath, string(sumText), file.Name); err != nil {
		return err
	}
	staging := filepath.Join(dataDir, "app", ".staging")
	_ = os.RemoveAll(staging)
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return err
	}
	if err := extractBundle(archivePath, staging); err != nil {
		_ = os.RemoveAll(staging)
		return fmt.Errorf("extract: %w", err)
	}
	if err := bundleLayoutOK(staging); err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	if err := os.Chmod(filepath.Join(staging, "fenghuolun"), 0o755); err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	if err := installBundleDir(dataDir, staging); err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	return nil
}
