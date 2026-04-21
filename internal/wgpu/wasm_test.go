//go:build js && wasm

package wgpu

import "testing"

func TestMapTextureFormat(t *testing.T) {
	tests := []struct {
		in   TextureFormat
		want string
	}{
		{TextureFormatBGRA8Unorm, "bgra8unorm"},
		{TextureFormatRGBA8Unorm, "rgba8unorm"},
		{TextureFormatR8Unorm, "r8unorm"},
		{TextureFormatDepth24Plus, "depth24plus"},
	}
	for _, tt := range tests {
		if got := mapTextureFormat(tt.in); got != tt.want {
			t.Errorf("mapTextureFormat(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestMapPrimitiveTopology(t *testing.T) {
	tests := []struct {
		in   PrimitiveTopology
		want string
	}{
		{PrimitiveTopologyTriangleList, "triangle-list"},
		{PrimitiveTopologyTriangleStrip, "triangle-strip"},
		{PrimitiveTopologyLineList, "line-list"},
		{PrimitiveTopologyPointList, "point-list"},
	}
	for _, tt := range tests {
		if got := mapPrimitiveTopology(tt.in); got != tt.want {
			t.Errorf("mapPrimitiveTopology(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestMapVertexFormat(t *testing.T) {
	tests := []struct {
		in   VertexFormat
		want string
	}{
		{VertexFormatFloat32x2, "float32x2"},
		{VertexFormatFloat32x4, "float32x4"},
		{VertexFormatFloat32, "float32"},
		{VertexFormatFloat32x3, "float32x3"},
	}
	for _, tt := range tests {
		if got := mapVertexFormat(tt.in); got != tt.want {
			t.Errorf("mapVertexFormat(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestMapBlendFactor(t *testing.T) {
	tests := []struct {
		in   BlendFactor
		want string
	}{
		{BlendFactorZero, "zero"},
		{BlendFactorOne, "one"},
		{BlendFactorSrcAlpha, "src-alpha"},
		{BlendFactorOneMinusSrcAlpha, "one-minus-src-alpha"},
	}
	for _, tt := range tests {
		if got := mapBlendFactor(tt.in); got != tt.want {
			t.Errorf("mapBlendFactor(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestMapBufferUsage(t *testing.T) {
	tests := []struct {
		in   BufferUsage
		want uint32
	}{
		{BufferUsageVertex, 0x0020},
		{BufferUsageIndex, 0x0010},
		{BufferUsageUniform, 0x0040},
		{BufferUsageCopySrc, 0x0004},
		{BufferUsageCopyDst, 0x0008},
		{BufferUsageVertex | BufferUsageCopyDst, 0x0028},
		{BufferUsageUniform | BufferUsageCopyDst, 0x0048},
	}
	for _, tt := range tests {
		if got := mapBufferUsage(tt.in); got != tt.want {
			t.Errorf("mapBufferUsage(%d) = 0x%04x, want 0x%04x", tt.in, got, tt.want)
		}
	}
}

func TestMapTextureUsage(t *testing.T) {
	tests := []struct {
		in   TextureUsage
		want uint32
	}{
		{TextureUsageCopySrc, 0x01},
		{TextureUsageCopyDst, 0x02},
		{TextureUsageTextureBinding, 0x04},
		{TextureUsageRenderAttachment, 0x10},
		{TextureUsageStorageBinding, 0x08},
		{TextureUsageRenderAttachment | TextureUsageCopyDst, 0x12},
	}
	for _, tt := range tests {
		if got := mapTextureUsage(tt.in); got != tt.want {
			t.Errorf("mapTextureUsage(%d) = 0x%02x, want 0x%02x", tt.in, got, tt.want)
		}
	}
}

func TestMapShaderStage(t *testing.T) {
	tests := []struct {
		in   ShaderStage
		want uint32
	}{
		{ShaderStageVertex, 0x1},
		{ShaderStageFragment, 0x2},
		{ShaderStageCompute, 0x4},
		{ShaderStageVertex | ShaderStageFragment, 0x3},
	}
	for _, tt := range tests {
		if got := mapShaderStage(tt.in); got != tt.want {
			t.Errorf("mapShaderStage(%d) = 0x%x, want 0x%x", tt.in, got, tt.want)
		}
	}
}

func TestMapCompareFunction(t *testing.T) {
	tests := []struct {
		in   CompareFunction
		want string
	}{
		{CompareFunctionNever, "never"},
		{CompareFunctionLess, "less"},
		{CompareFunctionEqual, "equal"},
		{CompareFunctionLessEqual, "less-equal"},
		{CompareFunctionGreater, "greater"},
		{CompareFunctionNotEqual, "not-equal"},
		{CompareFunctionGreaterEqual, "greater-equal"},
		{CompareFunctionAlways, "always"},
	}
	for _, tt := range tests {
		if got := mapCompareFunction(tt.in); got != tt.want {
			t.Errorf("mapCompareFunction(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestMapLoadStoreOp(t *testing.T) {
	if got := mapLoadOp(LoadOpClear); got != "clear" {
		t.Errorf("mapLoadOp(Clear) = %q", got)
	}
	if got := mapLoadOp(LoadOpLoad); got != "load" {
		t.Errorf("mapLoadOp(Load) = %q", got)
	}
	if got := mapStoreOp(StoreOpStore); got != "store" {
		t.Errorf("mapStoreOp(Store) = %q", got)
	}
	if got := mapStoreOp(StoreOpDiscard); got != "discard" {
		t.Errorf("mapStoreOp(Discard) = %q", got)
	}
}

func TestCreateInstance(t *testing.T) {
	// In a browser environment with WebGPU, this should succeed.
	// In Node.js without WebGPU, navigator.gpu is undefined and this returns an error.
	inst, err := CreateInstance()
	if err != nil {
		t.Skipf("WebGPU not available: %v", err)
	}
	if inst == nil {
		t.Fatal("CreateInstance returned nil without error")
	}
	inst.Destroy()
}
