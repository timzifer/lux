package app

// PersistenceConfig configures state persistence between app restarts (RFC §3.4).
// Encode serializes the model, Decode deserializes it, and StorageKey
// identifies the file within the app's storage directory.
type PersistenceConfig[M any] struct {
	Encode     func(M) ([]byte, error)
	Decode     func([]byte) (M, error)
	StorageKey string
}

// persistenceHooks is a type-erased version of PersistenceConfig
// that can be stored in the non-generic options struct.
type persistenceHooks struct {
	encode func(any) ([]byte, error)
	decode func([]byte) (any, error)
	key    string
}

// WithPersistence enables state persistence (RFC §3.4).
// The model is saved on shutdown and restored on startup.
func WithPersistence[M any](cfg PersistenceConfig[M]) Option {
	return func(o *options) {
		o.persistence = &persistenceHooks{
			encode: func(v any) ([]byte, error) {
				return cfg.Encode(v.(M))
			},
			decode: func(data []byte) (any, error) {
				return cfg.Decode(data)
			},
			key: cfg.StorageKey,
		}
	}
}
