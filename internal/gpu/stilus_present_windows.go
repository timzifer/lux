//go:build windows && !nogui

package gpu

import (
	"image"
	"unsafe"
)

// presentGDI copies a premultiplied RGBA frame into the window with
// SetDIBitsToDevice (top-down 32-bit BGRA DIB).
type presentGDI struct {
	hwnd uintptr
	bgra []byte
}

func (p *presentGDI) present(img *image.RGBA) {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	if p.hwnd == 0 || w <= 0 || h <= 0 {
		return
	}
	n := w * h * 4
	if cap(p.bgra) < n {
		p.bgra = make([]byte, n)
	}
	dst := p.bgra[:n]
	for y := 0; y < h; y++ {
		src := img.Pix[y*img.Stride : y*img.Stride+w*4]
		row := dst[y*w*4 : (y+1)*w*4]
		for x := 0; x < w*4; x += 4 {
			row[x], row[x+1], row[x+2], row[x+3] = src[x+2], src[x+1], src[x], src[x+3]
		}
	}
	hdc, _, _ := procGetDC.Call(p.hwnd)
	if hdc == 0 {
		return
	}
	defer procReleaseDC.Call(p.hwnd, hdc)
	bmi := bitmapInfoHeader{
		Size:     uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		Width:    int32(w),
		Height:   -int32(h), // top-down
		Planes:   1,
		BitCount: 32,
	}
	procSetDIBitsToDevice.Call(hdc, 0, 0, uintptr(w), uintptr(h), 0, 0, 0, uintptr(h),
		uintptr(unsafe.Pointer(&dst[0])), uintptr(unsafe.Pointer(&bmi)), 0)
}

// windowPresenter returns a presenter for the native window, or nil.
func windowPresenter(cfg Config) func(*image.RGBA) {
	if cfg.NativeHandle == 0 {
		return nil
	}
	p := &presentGDI{hwnd: cfg.NativeHandle}
	return p.present
}
