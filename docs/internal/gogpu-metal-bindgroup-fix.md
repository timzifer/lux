# fix(metal): use per-type sequential indices in SetBindGroup

## Problem

The Metal HAL's `SetBindGroup` on both `RenderPassEncoder` and `ComputePassEncoder` uses the WGSL `@binding(N)` number directly as the Metal argument table index for all resource types:

```go
slot := uintptr(entry.Binding)
// ...
_ = MsgSend(e.raw, Sel("setVertexBuffer:offset:atIndex:"), res.Buffer, offset, slot)
_ = MsgSend(e.raw, Sel("setFragmentTexture:atIndex:"), res.TextureView, slot)
_ = MsgSend(e.raw, Sel("setSamplerState:atIndex:"), res.Sampler, slot)
```

However, Metal's MSL uses **separate index spaces** for each resource type: `[[buffer(N)]]`, `[[texture(M)]]`, `[[sampler(K)]]`. The naga WGSL-to-MSL compiler (with `DefaultOptions()` / nil `PerEntryPointMap`) generates these indices **sequentially per type**, not from the WGSL binding number.

### Example

Given this WGSL:

```wgsl
@group(0) @binding(0) var<uniform> uniforms: Uniforms;
@group(0) @binding(1) var atlas_texture: texture_2d<f32>;
@group(0) @binding(2) var atlas_sampler: sampler;
```

Naga generates:

```metal
constant Uniforms& uniforms [[buffer(0)]]      // first buffer  -> 0
texture2d<float> atlas_texture [[texture(0)]]   // first texture -> 0
sampler atlas_sampler [[sampler(0)]]            // first sampler -> 0
```

But `SetBindGroup` currently binds:

| Resource | @binding | Current Metal slot | Expected Metal slot |
|---|---|---|---|
| uniforms | 0 | `setVertexBuffer atIndex:0` | `[[buffer(0)]]` **0** |
| atlas_texture | 1 | `setFragmentTexture atIndex:1` | `[[texture(0)]]` **0** |
| atlas_sampler | 2 | `setSamplerState atIndex:2` | `[[sampler(0)]]` **0** |

The texture and sampler are bound at the wrong Metal slots, so the shader reads uninitialized resources.

### Additional issue: vertex buffer conflict

Metal vertex buffers (`setVertexBuffer:offset:atIndex:`) and shader buffer arguments (`[[buffer(N)]]`) share the **same index space**. When `SetBindGroup` binds a uniform buffer at index 0, it overwrites a vertex buffer that was also bound at index 0 via `SetVertexBuffer(0, ...)` (or vice versa).

This does not affect DX12/Vulkan because those backends use separate descriptor heaps/binding spaces for vertex buffers and uniform buffers.

## Fix

Replace the single `slot` variable with three per-type counters:

```go
var bufferSlot, textureSlot, samplerSlot uintptr
for _, entry := range bg.entries {
    switch res := entry.Resource.(type) {
    case gputypes.BufferBinding:
        // ...
        _ = MsgSend(e.raw, Sel("setVertexBuffer:offset:atIndex:"), res.Buffer, offset, bufferSlot)
        _ = MsgSend(e.raw, Sel("setFragmentBuffer:offset:atIndex:"), res.Buffer, offset, bufferSlot)
        bufferSlot++
    case gputypes.TextureViewBinding:
        _ = MsgSend(e.raw, Sel("setVertexTexture:atIndex:"), res.TextureView, textureSlot)
        _ = MsgSend(e.raw, Sel("setFragmentTexture:atIndex:"), res.TextureView, textureSlot)
        textureSlot++
    case gputypes.SamplerBinding:
        _ = MsgSend(e.raw, Sel("setVertexSamplerState:atIndex:"), res.Sampler, samplerSlot)
        _ = MsgSend(e.raw, Sel("setFragmentSamplerState:atIndex:"), res.Sampler, samplerSlot)
        samplerSlot++
    }
}
```

This matches naga's MSL output where each resource type is numbered independently starting from 0.

The same fix is applied to both `RenderPassEncoder.SetBindGroup` and `ComputePassEncoder.SetBindGroup`.

## Affected files

- `hal/metal/encoder.go` — `RenderPassEncoder.SetBindGroup()` (~line 427)
- `hal/metal/encoder.go` — `ComputePassEncoder.SetBindGroup()` (~line 595)

## How to reproduce

Any wgpu application that uses bind groups with mixed resource types (buffers + textures + samplers) will have incorrect Metal bindings. Specifically:

1. Create a render pipeline with a shader that uses `@binding(0)` for a uniform, `@binding(1)` for a texture, and `@binding(2)` for a sampler
2. Bind vertex buffers at slot 0
3. Call `SetBindGroup` followed by draw calls
4. Result: textures and samplers are not visible; vertex buffer may be overwritten by the uniform buffer

## Note on vertex buffer slot conflict

The vertex buffer conflict (`[[buffer(N)]]` shared between vertex fetch and shader arguments) is a separate but related issue. It can be worked around on the application side by offsetting vertex buffer slots (e.g., starting at index 2+ instead of 0). However, a proper fix would involve configuring naga's `PerEntryPointMap` to offset buffer indices past the vertex buffer slots, or offsetting vertex descriptor buffer indices in `buildVertexDescriptor`. This PR addresses only the per-type sequential indexing issue.
