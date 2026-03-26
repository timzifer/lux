# State Persistence

lux can automatically save and restore the application model between restarts. The model is
serialised on shutdown and deserialised at startup before the first frame.

---

## Enabling Persistence

Pass `app.WithPersistence` to `app.Run`:

```go
import "encoding/json"

app.Run(model, update, view,
    app.WithPersistence(app.PersistenceConfig[Model]{
        Encode:     json.Marshal,
        Decode:     func(b []byte) (Model, error) {
            var m Model
            return m, json.Unmarshal(b, &m)
        },
        StorageKey: "my-app",
    }),
)
```

```go
type PersistenceConfig[M any] struct {
    Encode     func(M) ([]byte, error)
    Decode     func([]byte) (M, error)
    StorageKey string
}
```

- `Encode` — serialise the model to bytes. Any format works (JSON, gob, protobuf, etc.).
- `Decode` — deserialise bytes back to the model. Return the initial model on error.
- `StorageKey` — filename (without extension) within the platform storage directory.

---

## Storage Location

| Platform | Path |
|----------|------|
| Linux | `$XDG_DATA_HOME/lux/<StorageKey>` or `~/.local/share/lux/<StorageKey>` |
| macOS | `~/Library/Application Support/lux/<StorageKey>` |
| Windows | `%APPDATA%\lux\<StorageKey>` |

Override with `app.WithStoragePath`:

```go
app.Run(model, update, view,
    app.WithPersistence(cfg),
    app.WithStoragePath("/custom/path/state.json"),
)
```

---

## `ModelRestoredMsg`

When a persisted model is successfully loaded, `app.ModelRestoredMsg` is sent **once** before
the first user event. Handle it in `update` to re-apply any side effects that depend on the
restored state:

```go
func update(m Model, msg app.Msg) Model {
    switch msg.(type) {
    case app.ModelRestoredMsg:
        // The model is already restored. Re-apply side effects.
        app.Send(app.SetDarkModeMsg{Dark: m.Dark})
        app.Send(LoadRecentFilesMsg{Files: m.RecentFiles})
    }
    return m
}
```

`ModelRestoredMsg` is **not** sent if:
- No persisted file exists (first run).
- The decode step returns an error (the initial model is used instead).

---

## Model Evolution

As your model evolves across versions, the `Decode` function must handle older serialised
formats. A versioned approach using JSON:

```go
type modelV1 struct {
    Count int `json:"count"`
}

type modelV2 struct {
    Version int    `json:"v"`
    Count   int    `json:"count"`
    Dark    bool   `json:"dark"`
}

func decode(b []byte) (Model, error) {
    // Try v2 first.
    var v2 modelV2
    if err := json.Unmarshal(b, &v2); err == nil && v2.Version >= 2 {
        return Model{Count: v2.Count, Dark: v2.Dark}, nil
    }
    // Fall back to v1.
    var v1 modelV1
    if err := json.Unmarshal(b, &v1); err == nil {
        return Model{Count: v1.Count, Dark: false}, nil
    }
    // Unknown format — return zero model.
    return Model{}, nil
}
```

The decode function should **never** return a hard error for recoverable format mismatches —
returning the zero model is preferable to a startup failure.

---

## Security Note

The persisted file is plain bytes written to the user's application data directory. Do not
store secrets (passwords, tokens) in the persisted model. Use the platform keychain/credential
store for sensitive data.
