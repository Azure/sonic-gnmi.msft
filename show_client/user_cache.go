package show_client

import (
	"os"
	"path/filepath"
	"strconv"
)

// UserCache is a small helper mirroring utilities_common.cli.UserCache from sonic-utilities.
// It creates a per-user cache directory under /tmp/cache/<app>/<uid[-tag]> and exposes
// helpers to retrieve and remove the directory.
type UserCache struct {
	appName              string
	tag                  string
	uid                  int
	cacheDir             string
	cacheDirectoryApp    string
	cacheDirectorySuffix string
	cacheDirectory       string
}

// NewUserCache creates the cache directories if they don't exist and returns the helper.
// If appName is empty, it falls back to the current process name. Tag, if provided,
// is appended to the UID as "<uid>-<tag>".
func NewUserCache(appName, tag string) *UserCache {
	if appName == "" {
		// Use a stable default rather than argv[0] to avoid surprises in daemon contexts
		appName = "app"
	}
	uid := os.Getuid()

	uc := &UserCache{
		appName:  appName,
		tag:      tag,
		uid:      uid,
		cacheDir: "/tmp/cache",
	}

	uc.cacheDirectoryApp = filepath.Join(uc.cacheDir, uc.appName)
	suffix := strconv.Itoa(uid)
	if tag != "" {
		suffix = suffix + "-" + tag
	}
	uc.cacheDirectorySuffix = suffix
	uc.cacheDirectory = filepath.Join(uc.cacheDirectoryApp, uc.cacheDirectorySuffix)

	// Ensure directories exist
	_ = os.MkdirAll(uc.cacheDirectoryApp, 0755)
	_ = os.MkdirAll(uc.cacheDirectory, 0755)

	return uc
}

// GetDirectory returns the full path to the cache directory.
func (u *UserCache) GetDirectory() string {
	return u.cacheDirectory
}

// Remove deletes the per-user cache directory recursively.
func (u *UserCache) Remove() error {
	return os.RemoveAll(u.cacheDirectory)
}

// RemoveAll deletes the entire application cache directory recursively.
func (u *UserCache) RemoveAll() error {
	return os.RemoveAll(u.cacheDirectoryApp)
}

// Join returns a path joined under the cache directory.
func (u *UserCache) Join(names ...string) string {
	parts := append([]string{u.cacheDirectory}, names...)
	return filepath.Join(parts...)
}
