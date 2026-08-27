//go:build embedjk

package embedjk

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Extract writes the bundled jk binary to a per-version cache path and
// returns it, ready to exec. Idempotent: if that exact version is already
// there, it's reused as-is rather than rewritten every startup.
func Extract() (string, error) {
	if len(jkBinary) == 0 {
		return "", fmt.Errorf("no bundled jk for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("finding cache dir: %w", err)
	}
	dir := filepath.Join(cacheDir, "lazyjenkins")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating %s: %w", dir, err)
	}

	path := filepath.Join(dir, "jk-"+Version)
	if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() && info.Size() == int64(len(jkBinary)) {
		return path, nil
	}

	// Write to a temp file and rename into place, so a crash or a second
	// lazyjenkins starting concurrently never leaves a half-written binary
	// at the path callers are about to exec.
	tmp := path + fmt.Sprintf(".tmp-%d", os.Getpid())
	if err := os.WriteFile(tmp, jkBinary, 0o755); err != nil {
		return "", fmt.Errorf("writing %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("installing %s: %w", path, err)
	}
	return path, nil
}
