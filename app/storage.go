package app

import (
	"os"
	"path/filepath"
	"runtime"
)

// storagePath returns the platform-specific path for persisted state.
// If overridePath is non-empty, it is returned directly.
func storagePath(appName, key, overridePath string) string {
	if overridePath != "" {
		return overridePath
	}

	var dir string
	switch runtime.GOOS {
	case "windows":
		dir = filepath.Join(os.Getenv("APPDATA"), appName)
	case "darwin":
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, "Library", "Application Support", appName)
	default: // linux and others
		if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
			dir = filepath.Join(xdg, appName)
		} else {
			home, _ := os.UserHomeDir()
			dir = filepath.Join(home, ".local", "state", appName)
		}
	}
	return filepath.Join(dir, key+".bin")
}

// loadPersistedModel reads and decodes persisted state from disk.
func loadPersistedModel(hooks *persistenceHooks, path string) (any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return hooks.decode(data)
}

// savePersistedModel encodes and atomically writes state to disk.
func savePersistedModel(hooks *persistenceHooks, model any, path string) error {
	data, err := hooks.encode(model)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Atomic write: write to temp file then rename.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// WithStoragePath overrides the default platform-specific storage location.
func WithStoragePath(path string) Option {
	return func(o *options) { o.storagePath = path }
}
