//go:build js && wasm

package wgpu

import (
	"fmt"
	"log"
	"syscall/js"
)

// wasmCanvas holds the canvas element reference, set by the web platform before renderer init.
var wasmCanvas js.Value

// SetWASMCanvas stores the canvas element for surface creation.
func SetWASMCanvas(canvas js.Value) {
	wasmCanvas = canvas
}

// awaitPromise blocks the current goroutine until a JS Promise resolves or rejects.
func awaitPromise(promise js.Value) (js.Value, error) {
	ch := make(chan js.Value, 1)
	errCh := make(chan error, 1)
	thenFn := js.FuncOf(func(_ js.Value, args []js.Value) any {
		ch <- args[0]
		return nil
	})
	catchFn := js.FuncOf(func(_ js.Value, args []js.Value) any {
		errCh <- fmt.Errorf("js promise rejected: %s", args[0].Call("toString").String())
		return nil
	})
	defer thenFn.Release()
	defer catchFn.Release()
	promise.Call("then", thenFn).Call("catch", catchFn)
	select {
	case v := <-ch:
		return v, nil
	case err := <-errCh:
		return js.Value{}, err
	}
}

// copyBytesToJS copies a Go byte slice into a new JS Uint8Array.
func copyBytesToJS(data []byte) js.Value {
	arr := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(arr, data)
	return arr
}

// --- CreateInstance ---

func CreateInstance() (Instance, error) {
	log.Println("wgpu/wasm: CreateInstance start")
	gpu := js.Global().Get("navigator").Get("gpu")
	if gpu.IsUndefined() || gpu.IsNull() {
		return nil, fmt.Errorf("wgpu/wasm: WebGPU not supported (navigator.gpu is undefined)")
	}
	preferredFormat := "bgra8unorm"
	pf := gpu.Call("getPreferredCanvasFormat")
	if !pf.IsUndefined() && !pf.IsNull() {
		preferredFormat = pf.String()
	}
	preferredCanvasFormat = preferredFormat
	log.Printf("wgpu/wasm: CreateInstance OK, preferredFormat=%s", preferredFormat)
	return &wasmInstance{gpu: gpu, preferredFormat: preferredFormat}, nil
}

// --- Instance ---

type wasmInstance struct {
	gpu             js.Value // navigator.gpu
	preferredFormat string   // from getPreferredCanvasFormat()
}

// preferredCanvasFormat stores the browser's preferred format, set during CreateInstance.
var preferredCanvasFormat = "bgra8unorm"

func (i *wasmInstance) CreateSurface(desc *SurfaceDescriptor) Surface {
	log.Println("wgpu/wasm: CreateSurface start")
	canvas := wasmCanvas
	if canvas.IsUndefined() || canvas.IsNull() {
		log.Println("wgpu/wasm: wasmCanvas not set, falling back to getElementById")
		canvas = js.Global().Get("document").Call("getElementById", "lux-canvas")
	}
	if canvas.IsUndefined() || canvas.IsNull() {
		log.Println("wgpu/wasm: ERROR canvas not found")
		return &wasmSurface{}
	}
	log.Printf("wgpu/wasm: canvas found, width=%v height=%v", canvas.Get("width"), canvas.Get("height"))
	ctx := canvas.Call("getContext", "webgpu")
	if ctx.IsUndefined() || ctx.IsNull() {
		log.Println("wgpu/wasm: ERROR getContext('webgpu') returned null")
		return &wasmSurface{}
	}
	log.Println("wgpu/wasm: CreateSurface OK")
	return &wasmSurface{ctx: ctx, canvas: canvas}
}

func (i *wasmInstance) RequestAdapter(opts *RequestAdapterOptions) (Adapter, error) {
	log.Println("wgpu/wasm: RequestAdapter start")
	jsOpts := js.Global().Get("Object").New()
	if opts != nil {
		switch opts.PowerPreference {
		case PowerPreferenceLowPower:
			jsOpts.Set("powerPreference", "low-power")
		case PowerPreferenceHighPerformance:
			jsOpts.Set("powerPreference", "high-performance")
		}
	}
	promise := i.gpu.Call("requestAdapter", jsOpts)
	result, err := awaitPromise(promise)
	if err != nil {
		return nil, fmt.Errorf("wgpu/wasm: requestAdapter: %w", err)
	}
	if result.IsNull() || result.IsUndefined() {
		return nil, fmt.Errorf("wgpu/wasm: requestAdapter returned null (no WebGPU adapter available)")
	}
	log.Println("wgpu/wasm: RequestAdapter OK")
	return &wasmAdapter{jsAdapter: result}, nil
}

func (i *wasmInstance) Destroy() {}

// --- Adapter ---

type wasmAdapter struct {
	jsAdapter js.Value
}

func (a *wasmAdapter) RequestDevice(desc *DeviceDescriptor) (Device, error) {
	log.Println("wgpu/wasm: RequestDevice start")
	jsDesc := js.Global().Get("Object").New()
	if desc != nil && desc.Label != "" {
		jsDesc.Set("label", desc.Label)
	}
	promise := a.jsAdapter.Call("requestDevice", jsDesc)
	result, err := awaitPromise(promise)
	if err != nil {
		return nil, fmt.Errorf("wgpu/wasm: requestDevice: %w", err)
	}
	if result.IsNull() || result.IsUndefined() {
		return nil, fmt.Errorf("wgpu/wasm: requestDevice returned null")
	}
	errorHandler := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) > 0 {
			e := args[0]
			errObj := e.Get("error")
			msg := errObj.Get("message")
			log.Printf("wgpu/wasm: DEVICE ERROR: %s", msg)
			js.Global().Get("console").Call("error", "wgpu/wasm: device error:", msg)
		}
		return nil
	})
	result.Call("addEventListener", "uncapturederror", errorHandler)
	log.Println("wgpu/wasm: RequestDevice OK")
	return &wasmDevice{jsDevice: result}, nil
}

func (a *wasmAdapter) GetInfo() AdapterInfo {
	info := a.jsAdapter.Get("info")
	if info.IsUndefined() || info.IsNull() {
		return AdapterInfo{Name: "WebGPU", BackendType: "webgpu"}
	}
	return AdapterInfo{
		Name:        jsString(info, "device"),
		Vendor:      jsString(info, "vendor"),
		DriverInfo:  jsString(info, "description"),
		AdapterType: jsString(info, "architecture"),
		BackendType: "webgpu",
	}
}

// --- Device ---

type wasmDevice struct {
	jsDevice js.Value
}

func (d *wasmDevice) CreateShaderModule(desc *ShaderModuleDescriptor) ShaderModule {
	jsDesc := js.Global().Get("Object").New()
	if desc.Label != "" {
		jsDesc.Set("label", desc.Label)
	}
	jsDesc.Set("code", desc.Source)
	m := d.jsDevice.Call("createShaderModule", jsDesc)
	log.Printf("wgpu/wasm: CreateShaderModule %q", desc.Label)
	go func() {
		info, err := awaitPromise(m.Call("getCompilationInfo"))
		if err != nil {
			log.Printf("wgpu/wasm: shader %q compilationInfo error: %v", desc.Label, err)
			return
		}
		msgs := info.Get("messages")
		for i := 0; i < msgs.Length(); i++ {
			msg := msgs.Index(i)
			log.Printf("wgpu/wasm: shader %q %s: %s (line %d)", desc.Label,
				msg.Get("type").String(), msg.Get("message").String(), msg.Get("lineNum").Int())
		}
	}()
	return &wasmShaderModule{jsModule: m}
}

func (d *wasmDevice) CreateRenderPipeline(desc *RenderPipelineDescriptor) RenderPipeline {
	log.Printf("wgpu/wasm: CreateRenderPipeline %q", desc.Label)
	jsDesc := js.Global().Get("Object").New()
	if desc.Label != "" {
		jsDesc.Set("label", desc.Label)
	}

	// Vertex state
	jsVertex := js.Global().Get("Object").New()
	jsVertex.Set("module", desc.Vertex.Module.(*wasmShaderModule).jsModule)
	jsVertex.Set("entryPoint", desc.Vertex.EntryPoint)
	if len(desc.Vertex.Buffers) > 0 {
		jsBufs := js.Global().Get("Array").New(len(desc.Vertex.Buffers))
		for i, buf := range desc.Vertex.Buffers {
			if buf.ArrayStride == 0 && len(buf.Attributes) == 0 {
				jsBufs.SetIndex(i, js.Null())
				continue
			}
			jb := js.Global().Get("Object").New()
			jb.Set("arrayStride", int(buf.ArrayStride))
			jb.Set("stepMode", mapStepMode(buf.StepMode))
			if len(buf.Attributes) > 0 {
				attrs := js.Global().Get("Array").New(len(buf.Attributes))
				for j, attr := range buf.Attributes {
					ja := js.Global().Get("Object").New()
					ja.Set("format", mapVertexFormat(attr.Format))
					ja.Set("offset", int(attr.Offset))
					ja.Set("shaderLocation", int(attr.ShaderLocation))
					attrs.SetIndex(j, ja)
				}
				jb.Set("attributes", attrs)
			}
			jsBufs.SetIndex(i, jb)
		}
		jsVertex.Set("buffers", jsBufs)
	}
	jsDesc.Set("vertex", jsVertex)

	// Fragment state
	if desc.Fragment != nil {
		jsFrag := js.Global().Get("Object").New()
		jsFrag.Set("module", desc.Fragment.Module.(*wasmShaderModule).jsModule)
		jsFrag.Set("entryPoint", desc.Fragment.EntryPoint)
		if len(desc.Fragment.Targets) > 0 {
			targets := js.Global().Get("Array").New(len(desc.Fragment.Targets))
			for i, t := range desc.Fragment.Targets {
				jt := js.Global().Get("Object").New()
				fmt := mapTextureFormat(t.Format)
				if t.Format == TextureFormatBGRA8Unorm {
					fmt = preferredCanvasFormat
				}
				jt.Set("format", fmt)
				if t.Blend != nil {
					jb := js.Global().Get("Object").New()
					jb.Set("color", mapBlendComponent(t.Blend.Color))
					jb.Set("alpha", mapBlendComponent(t.Blend.Alpha))
					jt.Set("blend", jb)
				}
				targets.SetIndex(i, jt)
			}
			jsFrag.Set("targets", targets)
		}
		jsDesc.Set("fragment", jsFrag)
	}

	// Primitive state
	jsPrim := js.Global().Get("Object").New()
	jsPrim.Set("topology", mapPrimitiveTopology(desc.Primitive.Topology))
	jsPrim.Set("cullMode", mapCullMode(desc.Primitive.CullMode))
	jsPrim.Set("frontFace", mapFrontFace(desc.Primitive.FrontFace))
	jsDesc.Set("primitive", jsPrim)

	// Depth/stencil
	if desc.DepthStencil != nil {
		jsDS := js.Global().Get("Object").New()
		jsDS.Set("format", mapTextureFormat(desc.DepthStencil.Format))
		jsDS.Set("depthWriteEnabled", desc.DepthStencil.DepthWriteEnabled)
		jsDS.Set("depthCompare", mapCompareFunction(desc.DepthStencil.DepthCompare))
		jsDesc.Set("depthStencil", jsDS)
	}

	// Layout
	if len(desc.BindGroupLayouts) > 0 {
		jsLayouts := js.Global().Get("Array").New(len(desc.BindGroupLayouts))
		for i, l := range desc.BindGroupLayouts {
			jsLayouts.SetIndex(i, l.(*wasmBindGroupLayout).jsLayout)
		}
		layoutDesc := d.jsDevice.Call("createPipelineLayout", map[string]any{
			"bindGroupLayouts": jsLayouts,
		})
		jsDesc.Set("layout", layoutDesc)
	} else {
		jsDesc.Set("layout", "auto")
	}

	p := d.jsDevice.Call("createRenderPipeline", jsDesc)
	log.Printf("wgpu/wasm: CreateRenderPipeline %q OK", desc.Label)
	return &wasmRenderPipeline{jsPipeline: p}
}

func (d *wasmDevice) CreateBuffer(desc *BufferDescriptor) Buffer {
	jsDesc := js.Global().Get("Object").New()
	if desc.Label != "" {
		jsDesc.Set("label", desc.Label)
	}
	jsDesc.Set("size", int(desc.Size))
	jsDesc.Set("usage", int(mapBufferUsage(desc.Usage)))
	jsDesc.Set("mappedAtCreation", false)
	b := d.jsDevice.Call("createBuffer", jsDesc)
	return &wasmBuffer{jsBuffer: b, size: desc.Size}
}

func (d *wasmDevice) CreateTexture(desc *TextureDescriptor) Texture {
	jsDesc := js.Global().Get("Object").New()
	if desc.Label != "" {
		jsDesc.Set("label", desc.Label)
	}
	jsSize := js.Global().Get("Object").New()
	jsSize.Set("width", int(desc.Size.Width))
	jsSize.Set("height", int(desc.Size.Height))
	if desc.Size.DepthOrArrayLayers > 0 {
		jsSize.Set("depthOrArrayLayers", int(desc.Size.DepthOrArrayLayers))
	} else {
		jsSize.Set("depthOrArrayLayers", 1)
	}
	jsDesc.Set("size", jsSize)
	jsDesc.Set("format", mapTextureFormat(desc.Format))
	jsDesc.Set("usage", int(mapTextureUsage(desc.Usage)))
	t := d.jsDevice.Call("createTexture", jsDesc)
	return &wasmTexture{jsTexture: t, width: desc.Size.Width, height: desc.Size.Height, format: desc.Format}
}

func (d *wasmDevice) CreateBindGroupLayout(desc *BindGroupLayoutDescriptor) BindGroupLayout {
	jsDesc := js.Global().Get("Object").New()
	if desc.Label != "" {
		jsDesc.Set("label", desc.Label)
	}
	entries := js.Global().Get("Array").New(len(desc.Entries))
	for i, e := range desc.Entries {
		je := js.Global().Get("Object").New()
		je.Set("binding", int(e.Binding))
		je.Set("visibility", int(mapShaderStage(e.Visibility)))
		if e.Buffer != nil {
			jb := js.Global().Get("Object").New()
			jb.Set("type", mapBufferBindingType(e.Buffer.Type))
			je.Set("buffer", jb)
		}
		if e.Sampler != nil {
			js_ := js.Global().Get("Object").New()
			js_.Set("type", "filtering")
			je.Set("sampler", js_)
		}
		if e.Texture != nil {
			jt := js.Global().Get("Object").New()
			jt.Set("sampleType", "float")
			jt.Set("viewDimension", "2d")
			je.Set("texture", jt)
		}
		if e.StorageTexture != nil {
			jst := js.Global().Get("Object").New()
			jst.Set("access", mapStorageTextureAccess(e.StorageTexture.Access))
			jst.Set("format", mapTextureFormat(e.StorageTexture.Format))
			jst.Set("viewDimension", "2d")
			je.Set("storageTexture", jst)
		}
		entries.SetIndex(i, je)
	}
	jsDesc.Set("entries", entries)
	l := d.jsDevice.Call("createBindGroupLayout", jsDesc)
	return &wasmBindGroupLayout{jsLayout: l}
}

func (d *wasmDevice) CreateBindGroup(desc *BindGroupDescriptor) BindGroup {
	jsDesc := js.Global().Get("Object").New()
	if desc.Label != "" {
		jsDesc.Set("label", desc.Label)
	}
	jsDesc.Set("layout", desc.Layout.(*wasmBindGroupLayout).jsLayout)
	entries := js.Global().Get("Array").New(len(desc.Entries))
	for i, e := range desc.Entries {
		je := js.Global().Get("Object").New()
		je.Set("binding", int(e.Binding))
		if e.Buffer != nil {
			res := js.Global().Get("Object").New()
			res.Set("buffer", e.Buffer.(*wasmBuffer).jsBuffer)
			res.Set("offset", int(e.Offset))
			sz := e.Size
			if sz == 0 {
				sz = e.Buffer.(*wasmBuffer).size
			}
			res.Set("size", int(sz))
			je.Set("resource", res)
		} else if e.Sampler != nil {
			je.Set("resource", e.Sampler.(*wasmSampler).jsSampler)
		} else if e.Texture != nil {
			je.Set("resource", e.Texture.(*wasmTextureView).jsView)
		}
		entries.SetIndex(i, je)
	}
	jsDesc.Set("entries", entries)
	bg := d.jsDevice.Call("createBindGroup", jsDesc)
	return &wasmBindGroup{jsBindGroup: bg}
}

func (d *wasmDevice) CreateCommandEncoder() CommandEncoder {
	enc := d.jsDevice.Call("createCommandEncoder")
	return &wasmCommandEncoder{jsEncoder: enc}
}

func (d *wasmDevice) CreateSampler(desc *SamplerDescriptor) Sampler {
	jsDesc := js.Global().Get("Object").New()
	if desc != nil && desc.Label != "" {
		jsDesc.Set("label", desc.Label)
	}
	jsDesc.Set("magFilter", "linear")
	jsDesc.Set("minFilter", "linear")
	s := d.jsDevice.Call("createSampler", jsDesc)
	return &wasmSampler{jsSampler: s}
}

func (d *wasmDevice) CreateComputePipeline(desc *ComputePipelineDescriptor) ComputePipeline {
	jsDesc := js.Global().Get("Object").New()
	if desc.Label != "" {
		jsDesc.Set("label", desc.Label)
	}
	jsCompute := js.Global().Get("Object").New()
	jsCompute.Set("module", desc.Module.(*wasmShaderModule).jsModule)
	jsCompute.Set("entryPoint", desc.EntryPoint)
	jsDesc.Set("compute", jsCompute)
	if len(desc.BindGroupLayouts) > 0 {
		jsLayouts := js.Global().Get("Array").New(len(desc.BindGroupLayouts))
		for i, l := range desc.BindGroupLayouts {
			jsLayouts.SetIndex(i, l.(*wasmBindGroupLayout).jsLayout)
		}
		layoutDesc := d.jsDevice.Call("createPipelineLayout", map[string]any{
			"bindGroupLayouts": jsLayouts,
		})
		jsDesc.Set("layout", layoutDesc)
	} else {
		jsDesc.Set("layout", "auto")
	}
	p := d.jsDevice.Call("createComputePipeline", jsDesc)
	return &wasmComputePipeline{jsPipeline: p}
}

func (d *wasmDevice) GetQueue() Queue {
	q := d.jsDevice.Get("queue")
	return &wasmQueue{jsQueue: q}
}

func (d *wasmDevice) Destroy() {
	d.jsDevice.Call("destroy")
}

// --- Surface ---

type wasmSurface struct {
	ctx    js.Value // GPUCanvasContext
	canvas js.Value
	device js.Value // GPUDevice, stored from Configure
}

func (s *wasmSurface) Configure(device Device, config *SurfaceConfiguration) {
	log.Printf("wgpu/wasm: Configure start, ctx valid=%v, format=%s, size=%dx%d",
		!s.ctx.IsUndefined() && !s.ctx.IsNull(), preferredCanvasFormat, config.Width, config.Height)
	if s.ctx.IsUndefined() || s.ctx.IsNull() {
		log.Println("wgpu/wasm: Configure SKIP — no context")
		return
	}
	dev := device.(*wasmDevice).jsDevice
	s.device = dev
	jsCfg := js.Global().Get("Object").New()
	jsCfg.Set("device", dev)
	jsCfg.Set("format", preferredCanvasFormat)
	jsCfg.Set("usage", int(mapTextureUsage(config.Usage)))
	jsCfg.Set("alphaMode", "opaque")
	s.ctx.Call("configure", jsCfg)
	log.Println("wgpu/wasm: Configure OK")
}

var getCurrentTextureCount int

func (s *wasmSurface) GetCurrentTexture() (TextureView, error) {
	if s.ctx.IsUndefined() || s.ctx.IsNull() {
		return nil, fmt.Errorf("wgpu/wasm: surface not configured")
	}
	tex := s.ctx.Call("getCurrentTexture")
	if tex.IsUndefined() || tex.IsNull() {
		return nil, fmt.Errorf("wgpu/wasm: getCurrentTexture returned null")
	}
	getCurrentTextureCount++
	if getCurrentTextureCount <= 3 {
		log.Printf("wgpu/wasm: GetCurrentTexture #%d: width=%v height=%v format=%v",
			getCurrentTextureCount, tex.Get("width"), tex.Get("height"), tex.Get("format"))
	}
	view := tex.Call("createView")
	return &wasmTextureView{jsView: view}, nil
}

func (s *wasmSurface) Present() {
	// Browser WebGPU presents automatically at the end of the frame.
}

func (s *wasmSurface) Destroy() {}

// --- RenderPipeline ---

type wasmRenderPipeline struct {
	jsPipeline js.Value
}

func (p *wasmRenderPipeline) Destroy() {}

// --- ComputePipeline ---

type wasmComputePipeline struct {
	jsPipeline js.Value
}

func (p *wasmComputePipeline) Destroy() {}

// --- ShaderModule ---

type wasmShaderModule struct {
	jsModule js.Value
}

func (m *wasmShaderModule) Destroy() {}

// --- Buffer ---

type wasmBuffer struct {
	jsBuffer js.Value
	size     uint64
}

func (b *wasmBuffer) Write(queue Queue, data []byte) {
	q := queue.(*wasmQueue)
	arr := copyBytesToJS(data)
	q.jsQueue.Call("writeBuffer", b.jsBuffer, 0, arr.Get("buffer"))
}

func (b *wasmBuffer) Destroy() {
	b.jsBuffer.Call("destroy")
}

// --- Texture ---

type wasmTexture struct {
	jsTexture js.Value
	width     uint32
	height    uint32
	format    TextureFormat
}

func (t *wasmTexture) CreateView() TextureView {
	v := t.jsTexture.Call("createView")
	return &wasmTextureView{jsView: v}
}

func (t *wasmTexture) Write(queue Queue, data []byte, bytesPerRow uint32) {
	q := queue.(*wasmQueue)
	arr := copyBytesToJS(data)
	jsDst := js.Global().Get("Object").New()
	jsDst.Set("texture", t.jsTexture)
	jsLayout := js.Global().Get("Object").New()
	jsLayout.Set("bytesPerRow", int(bytesPerRow))
	jsLayout.Set("rowsPerImage", int(t.height))
	jsSize := js.Global().Get("Object").New()
	jsSize.Set("width", int(t.width))
	jsSize.Set("height", int(t.height))
	q.jsQueue.Call("writeTexture", jsDst, arr.Get("buffer"), jsLayout, jsSize)
}

func (t *wasmTexture) Destroy() {
	t.jsTexture.Call("destroy")
}

// --- TextureView ---

type wasmTextureView struct {
	jsView js.Value
}

func (v *wasmTextureView) Destroy() {}

// --- CommandEncoder ---

type wasmCommandEncoder struct {
	jsEncoder js.Value
}

var renderPassCount int

func (e *wasmCommandEncoder) BeginRenderPass(desc *RenderPassDescriptor) RenderPass {
	renderPassCount++
	jsDesc := js.Global().Get("Object").New()

	colors := js.Global().Get("Array").New(len(desc.ColorAttachments))
	for i, ca := range desc.ColorAttachments {
		if renderPassCount <= 3 {
			log.Printf("wgpu/wasm: BeginRenderPass #%d: colorAttachment[%d] loadOp=%d storeOp=%d clear=(%.2f,%.2f,%.2f,%.2f)",
				renderPassCount, i, ca.LoadOp, ca.StoreOp, ca.ClearValue.R, ca.ClearValue.G, ca.ClearValue.B, ca.ClearValue.A)
		}
		jca := js.Global().Get("Object").New()
		jca.Set("view", ca.View.(*wasmTextureView).jsView)
		jca.Set("loadOp", mapLoadOp(ca.LoadOp))
		jca.Set("storeOp", mapStoreOp(ca.StoreOp))
		clearArr := js.Global().Get("Object").New()
		clearArr.Set("r", ca.ClearValue.R)
		clearArr.Set("g", ca.ClearValue.G)
		clearArr.Set("b", ca.ClearValue.B)
		clearArr.Set("a", ca.ClearValue.A)
		jca.Set("clearValue", clearArr)
		colors.SetIndex(i, jca)
	}
	jsDesc.Set("colorAttachments", colors)

	if desc.DepthStencilAttachment != nil {
		jds := js.Global().Get("Object").New()
		jds.Set("view", desc.DepthStencilAttachment.View.(*wasmTextureView).jsView)
		jds.Set("depthLoadOp", mapLoadOp(desc.DepthStencilAttachment.DepthLoadOp))
		jds.Set("depthStoreOp", mapStoreOp(desc.DepthStencilAttachment.DepthStoreOp))
		jds.Set("depthClearValue", desc.DepthStencilAttachment.DepthClearValue)
		jsDesc.Set("depthStencilAttachment", jds)
	}

	pass := e.jsEncoder.Call("beginRenderPass", jsDesc)
	return &wasmRenderPass{jsPass: pass}
}

func (e *wasmCommandEncoder) BeginComputePass() ComputePass {
	pass := e.jsEncoder.Call("beginComputePass")
	return &wasmComputePass{jsPass: pass}
}

func (e *wasmCommandEncoder) CopyTextureToTexture(src, dst *ImageCopyTexture, size Extent3D) {
	jsSrc := js.Global().Get("Object").New()
	jsSrc.Set("texture", src.Texture.(*wasmTexture).jsTexture)
	jsSrc.Set("mipLevel", int(src.MipLevel))
	jsDst := js.Global().Get("Object").New()
	jsDst.Set("texture", dst.Texture.(*wasmTexture).jsTexture)
	jsDst.Set("mipLevel", int(dst.MipLevel))
	jsSize := js.Global().Get("Object").New()
	jsSize.Set("width", int(size.Width))
	jsSize.Set("height", int(size.Height))
	jsSize.Set("depthOrArrayLayers", int(size.DepthOrArrayLayers))
	e.jsEncoder.Call("copyTextureToTexture", jsSrc, jsDst, jsSize)
}

func (e *wasmCommandEncoder) Finish() CommandBuffer {
	buf := e.jsEncoder.Call("finish")
	return &wasmCommandBuffer{jsCmdBuf: buf}
}

// --- CommandBuffer ---

type wasmCommandBuffer struct {
	jsCmdBuf js.Value
}

// --- RenderPass ---

type wasmRenderPass struct {
	jsPass js.Value
}

func (p *wasmRenderPass) SetPipeline(pipeline RenderPipeline) {
	p.jsPass.Call("setPipeline", pipeline.(*wasmRenderPipeline).jsPipeline)
}

func (p *wasmRenderPass) SetBindGroup(index uint32, group BindGroup) {
	p.jsPass.Call("setBindGroup", int(index), group.(*wasmBindGroup).jsBindGroup)
}

func (p *wasmRenderPass) SetVertexBuffer(slot uint32, buffer Buffer, offset, size uint64) {
	p.jsPass.Call("setVertexBuffer", int(slot), buffer.(*wasmBuffer).jsBuffer, int(offset), int(size))
}

func (p *wasmRenderPass) SetIndexBuffer(buffer Buffer, format IndexFormat, offset, size uint64) {
	p.jsPass.Call("setIndexBuffer", buffer.(*wasmBuffer).jsBuffer, mapIndexFormat(format), int(offset), int(size))
}

func (p *wasmRenderPass) Draw(vertexCount, instanceCount, firstVertex, firstInstance uint32) {
	p.jsPass.Call("draw", int(vertexCount), int(instanceCount), int(firstVertex), int(firstInstance))
}

func (p *wasmRenderPass) DrawInstanced(vertexCount, instanceCount, firstVertex, firstInstance uint32) {
	p.jsPass.Call("draw", int(vertexCount), int(instanceCount), int(firstVertex), int(firstInstance))
}

func (p *wasmRenderPass) DrawIndexed(indexCount, instanceCount, firstIndex, baseVertex int32, firstInstance uint32) {
	p.jsPass.Call("drawIndexed", int(indexCount), int(instanceCount), int(firstIndex), int(baseVertex), int(firstInstance))
}

func (p *wasmRenderPass) SetScissorRect(x, y, width, height uint32) {
	p.jsPass.Call("setScissorRect", int(x), int(y), int(width), int(height))
}

func (p *wasmRenderPass) End() {
	p.jsPass.Call("end")
}

// --- ComputePass ---

type wasmComputePass struct {
	jsPass js.Value
}

func (p *wasmComputePass) SetPipeline(pipeline ComputePipeline) {
	p.jsPass.Call("setPipeline", pipeline.(*wasmComputePipeline).jsPipeline)
}

func (p *wasmComputePass) SetBindGroup(index uint32, group BindGroup) {
	p.jsPass.Call("setBindGroup", int(index), group.(*wasmBindGroup).jsBindGroup)
}

func (p *wasmComputePass) Dispatch(x, y, z uint32) {
	p.jsPass.Call("dispatchWorkgroups", int(x), int(y), int(z))
}

func (p *wasmComputePass) End() {
	p.jsPass.Call("end")
}

// --- Queue ---

type wasmQueue struct {
	jsQueue js.Value
}

var submitCount int

func (q *wasmQueue) Submit(buffers ...CommandBuffer) {
	arr := js.Global().Get("Array").New(len(buffers))
	for i, b := range buffers {
		arr.SetIndex(i, b.(*wasmCommandBuffer).jsCmdBuf)
	}
	q.jsQueue.Call("submit", arr)
	submitCount++
	if submitCount <= 3 {
		log.Printf("wgpu/wasm: Queue.Submit #%d (%d command buffers)", submitCount, len(buffers))
	}
}

func (q *wasmQueue) WriteBuffer(buffer Buffer, offset uint64, data []byte) {
	arr := copyBytesToJS(data)
	q.jsQueue.Call("writeBuffer", buffer.(*wasmBuffer).jsBuffer, int(offset), arr.Get("buffer"))
}

func (q *wasmQueue) WriteTexture(dst *ImageCopyTexture, data []byte, layout *TextureDataLayout, size Extent3D) {
	arr := copyBytesToJS(data)
	jsDst := js.Global().Get("Object").New()
	jsDst.Set("texture", dst.Texture.(*wasmTexture).jsTexture)
	jsDst.Set("mipLevel", int(dst.MipLevel))
	jsLayout := js.Global().Get("Object").New()
	jsLayout.Set("offset", int(layout.Offset))
	jsLayout.Set("bytesPerRow", int(layout.BytesPerRow))
	jsLayout.Set("rowsPerImage", int(layout.RowsPerImage))
	jsSize := js.Global().Get("Object").New()
	jsSize.Set("width", int(size.Width))
	jsSize.Set("height", int(size.Height))
	if size.DepthOrArrayLayers > 0 {
		jsSize.Set("depthOrArrayLayers", int(size.DepthOrArrayLayers))
	}
	q.jsQueue.Call("writeTexture", jsDst, arr.Get("buffer"), jsLayout, jsSize)
}

// --- BindGroup ---

type wasmBindGroup struct {
	jsBindGroup js.Value
}

func (g *wasmBindGroup) Destroy() {}

// --- BindGroupLayout ---

type wasmBindGroupLayout struct {
	jsLayout js.Value
}

func (l *wasmBindGroupLayout) Destroy() {}

// --- Sampler ---

type wasmSampler struct {
	jsSampler js.Value
}

func (s *wasmSampler) Destroy() {}

// ──────────────────────────────────────────────────────────────────────────────
// Enum mapping helpers: Go wgpu constants → JS WebGPU strings/flags
// ──────────────────────────────────────────────────────────────────────────────

func mapTextureFormat(f TextureFormat) string {
	switch f {
	case TextureFormatBGRA8Unorm:
		return "bgra8unorm"
	case TextureFormatRGBA8Unorm:
		return "rgba8unorm"
	case TextureFormatR8Unorm:
		return "r8unorm"
	case TextureFormatDepth24Plus:
		return "depth24plus"
	default:
		return "bgra8unorm"
	}
}

func mapPrimitiveTopology(t PrimitiveTopology) string {
	switch t {
	case PrimitiveTopologyTriangleList:
		return "triangle-list"
	case PrimitiveTopologyTriangleStrip:
		return "triangle-strip"
	case PrimitiveTopologyLineList:
		return "line-list"
	case PrimitiveTopologyPointList:
		return "point-list"
	default:
		return "triangle-list"
	}
}

func mapCullMode(m CullMode) string {
	switch m {
	case CullModeNone:
		return "none"
	case CullModeFront:
		return "front"
	case CullModeBack:
		return "back"
	default:
		return "none"
	}
}

func mapFrontFace(f FrontFace) string {
	switch f {
	case FrontFaceCCW:
		return "ccw"
	case FrontFaceCW:
		return "cw"
	default:
		return "ccw"
	}
}

func mapVertexFormat(f VertexFormat) string {
	switch f {
	case VertexFormatFloat32x2:
		return "float32x2"
	case VertexFormatFloat32x4:
		return "float32x4"
	case VertexFormatFloat32:
		return "float32"
	case VertexFormatFloat32x3:
		return "float32x3"
	default:
		return "float32x2"
	}
}

func mapStepMode(m VertexStepMode) string {
	switch m {
	case VertexStepModeVertex:
		return "vertex"
	case VertexStepModeInstance:
		return "instance"
	default:
		return "vertex"
	}
}

func mapBlendFactor(f BlendFactor) string {
	switch f {
	case BlendFactorZero:
		return "zero"
	case BlendFactorOne:
		return "one"
	case BlendFactorSrcAlpha:
		return "src-alpha"
	case BlendFactorOneMinusSrcAlpha:
		return "one-minus-src-alpha"
	default:
		return "one"
	}
}

func mapBlendOp(o BlendOperation) string {
	switch o {
	case BlendOperationAdd:
		return "add"
	default:
		return "add"
	}
}

func mapBlendComponent(c BlendComponent) js.Value {
	obj := js.Global().Get("Object").New()
	obj.Set("srcFactor", mapBlendFactor(c.SrcFactor))
	obj.Set("dstFactor", mapBlendFactor(c.DstFactor))
	obj.Set("operation", mapBlendOp(c.Operation))
	return obj
}

func mapLoadOp(op LoadOp) string {
	switch op {
	case LoadOpClear:
		return "clear"
	case LoadOpLoad:
		return "load"
	default:
		return "clear"
	}
}

func mapStoreOp(op StoreOp) string {
	switch op {
	case StoreOpStore:
		return "store"
	case StoreOpDiscard:
		return "discard"
	default:
		return "store"
	}
}

func mapCompareFunction(f CompareFunction) string {
	switch f {
	case CompareFunctionNever:
		return "never"
	case CompareFunctionLess:
		return "less"
	case CompareFunctionEqual:
		return "equal"
	case CompareFunctionLessEqual:
		return "less-equal"
	case CompareFunctionGreater:
		return "greater"
	case CompareFunctionNotEqual:
		return "not-equal"
	case CompareFunctionGreaterEqual:
		return "greater-equal"
	case CompareFunctionAlways:
		return "always"
	default:
		return "always"
	}
}

func mapIndexFormat(f IndexFormat) string {
	switch f {
	case IndexFormatUint16:
		return "uint16"
	case IndexFormatUint32:
		return "uint32"
	default:
		return "uint32"
	}
}

func mapBufferUsage(u BufferUsage) uint32 {
	var js uint32
	if u&BufferUsageVertex != 0 {
		js |= 0x0020 // GPUBufferUsage.VERTEX
	}
	if u&BufferUsageIndex != 0 {
		js |= 0x0010 // GPUBufferUsage.INDEX
	}
	if u&BufferUsageUniform != 0 {
		js |= 0x0040 // GPUBufferUsage.UNIFORM
	}
	if u&BufferUsageCopySrc != 0 {
		js |= 0x0004 // GPUBufferUsage.COPY_SRC
	}
	if u&BufferUsageCopyDst != 0 {
		js |= 0x0008 // GPUBufferUsage.COPY_DST
	}
	return js
}

func mapTextureUsage(u TextureUsage) uint32 {
	var js uint32
	if u&TextureUsageCopySrc != 0 {
		js |= 0x01 // GPUTextureUsage.COPY_SRC
	}
	if u&TextureUsageCopyDst != 0 {
		js |= 0x02 // GPUTextureUsage.COPY_DST
	}
	if u&TextureUsageTextureBinding != 0 {
		js |= 0x04 // GPUTextureUsage.TEXTURE_BINDING
	}
	if u&TextureUsageRenderAttachment != 0 {
		js |= 0x10 // GPUTextureUsage.RENDER_ATTACHMENT
	}
	if u&TextureUsageStorageBinding != 0 {
		js |= 0x08 // GPUTextureUsage.STORAGE_BINDING
	}
	return js
}

func mapShaderStage(s ShaderStage) uint32 {
	var js uint32
	if s&ShaderStageVertex != 0 {
		js |= 0x1 // GPUShaderStage.VERTEX
	}
	if s&ShaderStageFragment != 0 {
		js |= 0x2 // GPUShaderStage.FRAGMENT
	}
	if s&ShaderStageCompute != 0 {
		js |= 0x4 // GPUShaderStage.COMPUTE
	}
	return js
}

func mapBufferBindingType(t BufferBindingType) string {
	switch t {
	case BufferBindingTypeUniform:
		return "uniform"
	case BufferBindingTypeStorage:
		return "read-only-storage"
	default:
		return "uniform"
	}
}

func mapStorageTextureAccess(a StorageTextureAccess) string {
	switch a {
	case StorageTextureAccessWriteOnly:
		return "write-only"
	case StorageTextureAccessReadOnly:
		return "read-only"
	case StorageTextureAccessReadWrite:
		return "read-write"
	default:
		return "write-only"
	}
}

// jsString safely reads a string property from a JS object.
func jsString(obj js.Value, key string) string {
	v := obj.Get(key)
	if v.IsUndefined() || v.IsNull() {
		return ""
	}
	return v.String()
}
