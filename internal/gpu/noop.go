package gpu

// NoopRenderer is a no-op GPU renderer for headless/CI environments.
type NoopRenderer struct{}

func (r *NoopRenderer) Init(cfg Config) error       { return nil }
func (r *NoopRenderer) Resize(width, height int)     {}
func (r *NoopRenderer) BeginFrame()                  {}
func (r *NoopRenderer) EndFrame()                    {}
func (r *NoopRenderer) Destroy()                     {}
