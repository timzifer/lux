//go:build nogui

package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type persistModel struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func testPersistenceConfig() PersistenceConfig[persistModel] {
	return PersistenceConfig[persistModel]{
		Encode: func(m persistModel) ([]byte, error) {
			return json.Marshal(m)
		},
		Decode: func(data []byte) (persistModel, error) {
			var m persistModel
			err := json.Unmarshal(data, &m)
			return m, err
		},
		StorageKey: "test-state",
	}
}

func TestPersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.bin")

	cfg := testPersistenceConfig()
	hooks := &persistenceHooks{
		encode: func(v any) ([]byte, error) {
			return cfg.Encode(v.(persistModel))
		},
		decode: func(data []byte) (any, error) {
			return cfg.Decode(data)
		},
		key: cfg.StorageKey,
	}

	original := persistModel{Name: "alice", Count: 42}
	if err := savePersistedModel(hooks, original, path); err != nil {
		t.Fatalf("save: %v", err)
	}

	restored, err := loadPersistedModel(hooks, path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	rm := restored.(persistModel)
	if rm.Name != "alice" || rm.Count != 42 {
		t.Errorf("restored = %+v, want {alice 42}", rm)
	}
}

func TestPersistenceDecodeError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.bin")

	// Write invalid JSON.
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := testPersistenceConfig()
	hooks := &persistenceHooks{
		encode: func(v any) ([]byte, error) {
			return cfg.Encode(v.(persistModel))
		},
		decode: func(data []byte) (any, error) {
			return cfg.Decode(data)
		},
		key: cfg.StorageKey,
	}

	_, err := loadPersistedModel(hooks, path)
	if err == nil {
		t.Error("expected decode error for invalid data")
	}
}

func TestPersistenceNoFileWithoutConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.bin")

	hooks := &persistenceHooks{
		encode: func(v any) ([]byte, error) {
			return nil, nil
		},
		decode: func(data []byte) (any, error) {
			return nil, nil
		},
		key: "test",
	}

	// Loading from non-existent file should return an error, not panic.
	_, err := loadPersistedModel(hooks, path)
	if err == nil {
		t.Error("expected error when file does not exist")
	}
}

func TestStoragePathPlatformDefault(t *testing.T) {
	// With no override, storagePath should return a non-empty path.
	p := storagePath("testapp", "state", "")
	if p == "" {
		t.Error("storagePath returned empty string")
	}
}

func TestStoragePathOverride(t *testing.T) {
	p := storagePath("testapp", "state", "/custom/path.bin")
	if p != "/custom/path.bin" {
		t.Errorf("storagePath with override = %q, want /custom/path.bin", p)
	}
}

func TestWithPersistenceOption(t *testing.T) {
	cfg := testPersistenceConfig()
	opt := WithPersistence(cfg)

	o := defaultOptions()
	opt(&o)

	if o.persistence == nil {
		t.Fatal("persistence should be set")
	}
	if o.persistence.key != "test-state" {
		t.Errorf("key = %q, want test-state", o.persistence.key)
	}
}
