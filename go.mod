module github.com/timzifer/lux

go 1.25

replace github.com/gogpu/wgpu v0.21.3 => ./vendor_gogpu_wgpu

require (
	github.com/go-gl/gl v0.0.0-20231021071112-07e5d0ea2e71
	github.com/go-gl/glfw/v3.3/glfw v0.0.0-20250301202403-da16c1255728
	github.com/go-text/typesetting v0.3.4
	github.com/gogpu/wgpu v0.21.3
	github.com/pierrec/msdf v0.0.0-20260126203608-76b1ee18a962
	github.com/zzl/go-com v1.5.0
	github.com/zzl/go-webview2 v0.0.0-20230129130204-9df4a7d166d5
	github.com/zzl/go-win32api/v2 v2.0.1
	golang.org/x/image v0.29.0
	golang.org/x/text v0.27.0
)

require (
	github.com/go-webgpu/goffi v0.4.2 // indirect
	github.com/gogpu/gputypes v0.3.0 // indirect
	github.com/gogpu/naga v0.14.8 // indirect
	golang.org/x/sys v0.41.0 // indirect
)
