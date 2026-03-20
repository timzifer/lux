//go:build darwin && cocoa && !nogui && arm64

package cocoa

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"github.com/go-webgpu/goffi/ffi"
	"github.com/go-webgpu/goffi/types"
)

// nsPoint mirrors CGPoint / NSPoint.
type nsPoint struct {
	X, Y float64
}

// nsSize mirrors CGSize / NSSize.
type nsSize struct {
	Width, Height float64
}

// nsRect mirrors CGRect / NSRect.
type nsRect struct {
	Origin nsPoint
	Size   nsSize
}

// ObjC FFI type descriptors for struct passing via objc_msgSend.
var (
	nsPointType = &types.TypeDescriptor{
		Kind: types.StructType,
		Members: []*types.TypeDescriptor{
			types.DoubleTypeDescriptor,
			types.DoubleTypeDescriptor,
		},
	}
	nsSizeType = &types.TypeDescriptor{
		Kind: types.StructType,
		Members: []*types.TypeDescriptor{
			types.DoubleTypeDescriptor,
			types.DoubleTypeDescriptor,
		},
	}
	nsRectType = &types.TypeDescriptor{
		Kind: types.StructType,
		Members: []*types.TypeDescriptor{nsPointType, nsSizeType},
	}
)

// objcRT holds cached ObjC runtime symbols.
type objcRT struct {
	libobjc    unsafe.Pointer
	foundation unsafe.Pointer
	appKit     unsafe.Pointer
	quartzCore unsafe.Pointer

	fnGetClass    unsafe.Pointer
	fnSelRegister unsafe.Pointer
	fnMsgSend     unsafe.Pointer

	cifStrToPtr types.CallInterface // (cstring) -> ptr
}

var (
	rtOnce sync.Once
	rt     *objcRT
	rtErr  error

	classCache sync.Map
	selCache   sync.Map
)

// ensureRT loads the ObjC runtime and resolves symbols (once).
func ensureRT() error {
	rtOnce.Do(func() {
		r := &objcRT{}
		var err error

		r.libobjc, err = ffi.LoadLibrary("/usr/lib/libobjc.A.dylib")
		if err != nil {
			rtErr = fmt.Errorf("cocoa: load libobjc: %w", err)
			return
		}
		r.foundation, err = ffi.LoadLibrary("/System/Library/Frameworks/Foundation.framework/Foundation")
		if err != nil {
			rtErr = fmt.Errorf("cocoa: load Foundation: %w", err)
			return
		}
		r.appKit, err = ffi.LoadLibrary("/System/Library/Frameworks/AppKit.framework/AppKit")
		if err != nil {
			rtErr = fmt.Errorf("cocoa: load AppKit: %w", err)
			return
		}
		r.quartzCore, err = ffi.LoadLibrary("/System/Library/Frameworks/QuartzCore.framework/QuartzCore")
		if err != nil {
			rtErr = fmt.Errorf("cocoa: load QuartzCore: %w", err)
			return
		}

		r.fnGetClass, err = ffi.GetSymbol(r.libobjc, "objc_getClass")
		if err != nil {
			rtErr = fmt.Errorf("cocoa: objc_getClass: %w", err)
			return
		}
		r.fnSelRegister, err = ffi.GetSymbol(r.libobjc, "sel_registerName")
		if err != nil {
			rtErr = fmt.Errorf("cocoa: sel_registerName: %w", err)
			return
		}
		r.fnMsgSend, err = ffi.GetSymbol(r.libobjc, "objc_msgSend")
		if err != nil {
			rtErr = fmt.Errorf("cocoa: objc_msgSend: %w", err)
			return
		}

		err = ffi.PrepareCallInterface(
			&r.cifStrToPtr,
			types.DefaultCall,
			types.PointerTypeDescriptor,
			[]*types.TypeDescriptor{types.PointerTypeDescriptor},
		)
		if err != nil {
			rtErr = fmt.Errorf("cocoa: prepare CIF: %w", err)
			return
		}

		rt = r
	})
	return rtErr
}

// getClass returns the ObjC class pointer for the given name.
func getClass(name string) uintptr {
	if v, ok := classCache.Load(name); ok {
		return v.(uintptr)
	}
	cname := append([]byte(name), 0)
	namePtr := unsafe.Pointer(&cname[0])
	var result uintptr
	_ = ffi.CallFunction(&rt.cifStrToPtr, rt.fnGetClass, unsafe.Pointer(&result), []unsafe.Pointer{unsafe.Pointer(&namePtr)})
	runtime.KeepAlive(cname)
	classCache.Store(name, result)
	return result
}

// sel returns the ObjC selector for the given name.
func sel(name string) uintptr {
	if v, ok := selCache.Load(name); ok {
		return v.(uintptr)
	}
	cname := append([]byte(name), 0)
	namePtr := unsafe.Pointer(&cname[0])
	var result uintptr
	_ = ffi.CallFunction(&rt.cifStrToPtr, rt.fnSelRegister, unsafe.Pointer(&result), []unsafe.Pointer{unsafe.Pointer(&namePtr)})
	runtime.KeepAlive(cname)
	selCache.Store(name, result)
	return result
}

// objcArg represents a typed argument for objc_msgSend.
type objcArg struct {
	typ       *types.TypeDescriptor
	ptr       unsafe.Pointer
	keepAlive any
}

func argPtr(val uintptr) objcArg {
	v := val
	return objcArg{typ: types.PointerTypeDescriptor, ptr: unsafe.Pointer(&v), keepAlive: &v}
}

func argUInt64(val uint64) objcArg {
	v := val
	return objcArg{typ: types.UInt64TypeDescriptor, ptr: unsafe.Pointer(&v), keepAlive: &v}
}

func argInt64(val int64) objcArg {
	v := val
	return objcArg{typ: types.SInt64TypeDescriptor, ptr: unsafe.Pointer(&v), keepAlive: &v}
}

func argBool(val bool) objcArg {
	var v uint8
	if val {
		v = 1
	}
	return objcArg{typ: types.UInt8TypeDescriptor, ptr: unsafe.Pointer(&v), keepAlive: &v}
}

func argDouble(val float64) objcArg {
	v := val
	return objcArg{typ: types.DoubleTypeDescriptor, ptr: unsafe.Pointer(&v), keepAlive: &v}
}

func argInt32(val int32) objcArg {
	v := val
	return objcArg{typ: types.SInt32TypeDescriptor, ptr: unsafe.Pointer(&v), keepAlive: &v}
}

func argRect(r nsRect) objcArg {
	v := r
	return objcArg{typ: nsRectType, ptr: unsafe.Pointer(&v), keepAlive: &v}
}

func argSize(s nsSize) objcArg {
	v := s
	return objcArg{typ: nsSizeType, ptr: unsafe.Pointer(&v), keepAlive: &v}
}

func argPoint(p nsPoint) objcArg {
	v := p
	return objcArg{typ: nsPointType, ptr: unsafe.Pointer(&v), keepAlive: &v}
}

// msgSend calls objc_msgSend with arbitrary arguments and return type.
func msgSend(retType *types.TypeDescriptor, rvalue unsafe.Pointer, self, cmd uintptr, args ...objcArg) {
	argTypes := make([]*types.TypeDescriptor, 0, 2+len(args))
	argTypes = append(argTypes, types.PointerTypeDescriptor, types.PointerTypeDescriptor)
	for _, a := range args {
		argTypes = append(argTypes, a.typ)
	}

	var cif types.CallInterface
	_ = ffi.PrepareCallInterface(&cif, types.DefaultCall, retType, argTypes)

	selfVal := self
	cmdVal := cmd
	argPtrs := make([]unsafe.Pointer, 0, 2+len(args))
	argPtrs = append(argPtrs, unsafe.Pointer(&selfVal), unsafe.Pointer(&cmdVal))
	for _, a := range args {
		argPtrs = append(argPtrs, a.ptr)
	}

	_ = ffi.CallFunction(&cif, rt.fnMsgSend, rvalue, argPtrs)
	runtime.KeepAlive(args)
}

// Convenience wrappers for common return types.

func msgSendPtr(self, cmd uintptr, args ...objcArg) uintptr {
	var result uintptr
	msgSend(types.PointerTypeDescriptor, unsafe.Pointer(&result), self, cmd, args...)
	return result
}

func msgSendVoid(self, cmd uintptr, args ...objcArg) {
	msgSend(types.VoidTypeDescriptor, nil, self, cmd, args...)
}

func msgSendBool(self, cmd uintptr, args ...objcArg) bool {
	var result uint8
	msgSend(types.UInt8TypeDescriptor, unsafe.Pointer(&result), self, cmd, args...)
	return result != 0
}

func msgSendUInt64(self, cmd uintptr, args ...objcArg) uint64 {
	var result uint64
	msgSend(types.UInt64TypeDescriptor, unsafe.Pointer(&result), self, cmd, args...)
	return result
}

func msgSendDouble(self, cmd uintptr, args ...objcArg) float64 {
	var result float64
	msgSend(types.DoubleTypeDescriptor, unsafe.Pointer(&result), self, cmd, args...)
	return result
}

func msgSendRect(self, cmd uintptr, args ...objcArg) nsRect {
	var result nsRect
	msgSend(nsRectType, unsafe.Pointer(&result), self, cmd, args...)
	return result
}

func msgSendPoint(self, cmd uintptr, args ...objcArg) nsPoint {
	var result nsPoint
	msgSend(nsPointType, unsafe.Pointer(&result), self, cmd, args...)
	return result
}

// NSString helpers.

func newNSString(s string) uintptr {
	cls := getClass("NSString")
	obj := msgSendPtr(cls, sel("alloc"))
	bytes := append([]byte(s), 0)
	ptr := unsafe.Pointer(&bytes[0])
	obj = msgSendPtr(obj, sel("initWithUTF8String:"), argPtr(uintptr(ptr)))
	runtime.KeepAlive(bytes)
	return obj
}

func goString(nsStr uintptr) string {
	if nsStr == 0 {
		return ""
	}
	ptr := msgSendPtr(nsStr, sel("UTF8String"))
	return cStringToGo(ptr)
}

func cStringToGo(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	// Convert C pointer (uintptr) to unsafe.Pointer via indirection
	// to avoid go vet's misuse-of-unsafe.Pointer false positive.
	p := *(*unsafe.Pointer)(unsafe.Pointer(&ptr))
	var length int
	for {
		if *(*byte)(unsafe.Add(p, length)) == 0 {
			break
		}
		length++
	}
	return unsafe.String((*byte)(p), length)
}

// ObjC runtime class creation helpers.

// viewMetalLayers maps NSView pointers to their CAMetalLayer pointers.
// Used by the makeBackingLayer callback to return the correct layer per view.
var viewMetalLayers sync.Map

// registerLuxViewClassHooks is a list of hooks invoked just before
// objc_registerClassPair in registerLuxViewClass. Each hook can add
// additional ObjC methods to the LuxMetalView class (e.g. accessibility overrides).
var registerLuxViewClassHooks []func(cls uintptr, fnAddMethod unsafe.Pointer, cifAddMethod *types.CallInterface)

// registerLuxViewClass creates a custom NSView subclass "LuxMetalView" that
// overrides makeBackingLayer to return a CAMetalLayer and wantsUpdateLayer
// to return YES. This is equivalent to the CGo LuxView in cocoa.m.
// Returns the class pointer. Safe to call multiple times (idempotent).
//
// Before creating a view instance, call viewMetalLayers.Store(viewPtr, layerPtr)
// so the makeBackingLayer callback can find the correct layer.
func registerLuxViewClass(metalLayerPtr *uintptr) uintptr {
	className := append([]byte("LuxMetalView"), 0)

	// Check if already registered.
	existing := getClass("LuxMetalView")
	if existing != 0 {
		return existing
	}

	// Load runtime functions.
	fnAllocClassPair, _ := ffi.GetSymbol(rt.libobjc, "objc_allocateClassPair")
	fnRegisterClassPair, _ := ffi.GetSymbol(rt.libobjc, "objc_registerClassPair")
	fnAddMethod, _ := ffi.GetSymbol(rt.libobjc, "class_addMethod")

	nsViewClass := getClass("NSView")

	// objc_allocateClassPair(NSView, "LuxMetalView", 0) -> Class
	var cifAllocClass types.CallInterface
	_ = ffi.PrepareCallInterface(&cifAllocClass, types.DefaultCall, types.PointerTypeDescriptor,
		[]*types.TypeDescriptor{types.PointerTypeDescriptor, types.PointerTypeDescriptor, types.UInt64TypeDescriptor})
	namePtr := unsafe.Pointer(&className[0])
	superClass := nsViewClass
	var extraBytes uint64
	var newClass uintptr
	_ = ffi.CallFunction(&cifAllocClass, fnAllocClassPair, unsafe.Pointer(&newClass),
		[]unsafe.Pointer{unsafe.Pointer(&superClass), unsafe.Pointer(&namePtr), unsafe.Pointer(&extraBytes)})
	runtime.KeepAlive(className)

	if newClass == 0 {
		return 0
	}

	// class_addMethod(cls, SEL, IMP, types) -> BOOL
	var cifAddMethod types.CallInterface
	_ = ffi.PrepareCallInterface(&cifAddMethod, types.DefaultCall, types.UInt8TypeDescriptor,
		[]*types.TypeDescriptor{types.PointerTypeDescriptor, types.PointerTypeDescriptor,
			types.PointerTypeDescriptor, types.PointerTypeDescriptor})

	// makeBackingLayer: looks up the CAMetalLayer for this view instance.
	// ObjC signature: -(CALayer*)makeBackingLayer { return metalLayer; }
	makeBackingLayerIMP := ffi.NewCallback(func(self, _cmd uintptr) uintptr {
		if layer, ok := viewMetalLayers.Load(self); ok {
			return layer.(uintptr)
		}
		// Fallback for the primary window (set during Init before alloc).
		return *metalLayerPtr
	})
	makeBackingLayerSel := sel("makeBackingLayer")
	makeBackingLayerTypes := append([]byte("@@:"), 0) // returns id, takes (id self, SEL _cmd)
	makeBackingLayerTypesPtr := unsafe.Pointer(&makeBackingLayerTypes[0])
	var addResult1 uint8
	_ = ffi.CallFunction(&cifAddMethod, fnAddMethod, unsafe.Pointer(&addResult1),
		[]unsafe.Pointer{unsafe.Pointer(&newClass), unsafe.Pointer(&makeBackingLayerSel),
			unsafe.Pointer(&makeBackingLayerIMP), unsafe.Pointer(&makeBackingLayerTypesPtr)})
	runtime.KeepAlive(makeBackingLayerTypes)

	// wantsUpdateLayer: returns YES.
	// ObjC signature: -(BOOL)wantsUpdateLayer { return YES; }
	wantsUpdateLayerIMP := ffi.NewCallback(func(self, _cmd uintptr) uintptr {
		return 1 // YES
	})
	wantsUpdateLayerSel := sel("wantsUpdateLayer")
	wantsUpdateLayerTypes := append([]byte("B@:"), 0) // returns BOOL, takes (id, SEL)
	wantsUpdateLayerTypesPtr := unsafe.Pointer(&wantsUpdateLayerTypes[0])
	var addResult2 uint8
	_ = ffi.CallFunction(&cifAddMethod, fnAddMethod, unsafe.Pointer(&addResult2),
		[]unsafe.Pointer{unsafe.Pointer(&newClass), unsafe.Pointer(&wantsUpdateLayerSel),
			unsafe.Pointer(&wantsUpdateLayerIMP), unsafe.Pointer(&wantsUpdateLayerTypesPtr)})
	runtime.KeepAlive(wantsUpdateLayerTypes)

	// acceptsFirstResponder: returns YES (for keyboard events).
	acceptsFirstResponderIMP := ffi.NewCallback(func(self, _cmd uintptr) uintptr {
		return 1 // YES
	})
	acceptsFirstResponderSel := sel("acceptsFirstResponder")
	acceptsFirstResponderTypes := append([]byte("B@:"), 0)
	acceptsFirstResponderTypesPtr := unsafe.Pointer(&acceptsFirstResponderTypes[0])
	var addResult3 uint8
	_ = ffi.CallFunction(&cifAddMethod, fnAddMethod, unsafe.Pointer(&addResult3),
		[]unsafe.Pointer{unsafe.Pointer(&newClass), unsafe.Pointer(&acceptsFirstResponderSel),
			unsafe.Pointer(&acceptsFirstResponderIMP), unsafe.Pointer(&acceptsFirstResponderTypesPtr)})
	runtime.KeepAlive(acceptsFirstResponderTypes)

	// Invoke hooks to add additional methods (e.g. accessibility overrides).
	for _, hook := range registerLuxViewClassHooks {
		hook(newClass, fnAddMethod, &cifAddMethod)
	}

	// objc_registerClassPair(cls)
	var cifRegister types.CallInterface
	_ = ffi.PrepareCallInterface(&cifRegister, types.DefaultCall, types.VoidTypeDescriptor,
		[]*types.TypeDescriptor{types.PointerTypeDescriptor})
	_ = ffi.CallFunction(&cifRegister, fnRegisterClassPair, nil,
		[]unsafe.Pointer{unsafe.Pointer(&newClass)})

	classCache.Store("LuxMetalView", newClass)
	return newClass
}

// Autorelease pool helpers.

func newAutoreleasePool() uintptr {
	cls := getClass("NSAutoreleasePool")
	return msgSendPtr(cls, sel("new"))
}

func drainPool(pool uintptr) {
	msgSendVoid(pool, sel("drain"))
}
