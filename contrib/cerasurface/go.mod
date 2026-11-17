module github.com/timzifer/lux/contrib/cerasurface

go 1.27.1

replace (
	// lux builds gogpu/wgpu from its vendored fork; replaces of a
	// dependency do not apply here, so repeat it.
	github.com/gogpu/wgpu v0.21.3 => ../../vendor_gogpu_wgpu
	// Develop against the lux checkout this module lives in.
	github.com/timzifer/lux => ../..
)

require (
	github.com/timzifer/cera v0.13.0
	github.com/timzifer/lux v0.0.0-00010101000000-000000000000
)

require (
	github.com/andybalholm/brotli v1.2.6 // indirect
	github.com/go-gl/gl v0.0.0-20231021071112-07e5d0ea2e71 // indirect
	github.com/go-images/jpeg v0.3.0 // indirect
	github.com/go-images/jpeg2000 v0.14.0 // indirect
	github.com/go-opentype/fonts v0.12.0 // indirect
	github.com/go-opentype/opentype v0.15.0 // indirect
	github.com/go-simd/highway v0.0.13 // indirect
	github.com/go-text/typesetting v0.3.4 // indirect
	github.com/go-webgpu/goffi v0.4.2 // indirect
	github.com/gogpu/gputypes v0.3.0 // indirect
	github.com/gogpu/naga v0.14.8 // indirect
	github.com/gogpu/wgpu v0.21.3 // indirect
	github.com/pierrec/msdf v0.0.0-20260126203608-76b1ee18a962 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/tannevaled/gobig2 v0.2.0 // indirect
	github.com/timzifer/stilus v0.20.0 // indirect
	golang.org/x/image v0.29.0 // indirect
	golang.org/x/sys v0.44.0 // indirect
	golang.org/x/text v0.42.0 // indirect
)
