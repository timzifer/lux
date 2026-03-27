package ui

// ZeroCopyMode describes the texture sharing mechanism for a surface (RFC §8.2).
type ZeroCopyMode int

const (
	ZeroCopyNone      ZeroCopyMode = iota // CPU copy fallback (OSR → Upload)
	ZeroCopyIOSurface                     // macOS: IOSurface → wgpu shared texture
	ZeroCopyDMABuf                        // Linux: DMA-buf → wgpu external memory
	ZeroCopyDXGI                          // Windows: DXGI shared handle
)

// PreferredZeroCopyMode returns the best available zero-copy mode for the
// current platform. Falls back to ZeroCopyNone if no shared-memory path
// is available.
func PreferredZeroCopyMode() ZeroCopyMode {
	return zeroCopyPlatform()
}
