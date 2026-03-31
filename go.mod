module github.com/timzifer/lux

go 1.25.0

replace github.com/gogpu/wgpu v0.21.3 => ./vendor_gogpu_wgpu

require (
	github.com/go-gl/gl v0.0.0-20231021071112-07e5d0ea2e71
	github.com/go-gl/glfw/v3.3/glfw v0.0.0-20250301202403-da16c1255728
	github.com/go-text/typesetting v0.3.4
	github.com/go-webgpu/goffi v0.4.2
	github.com/godbus/dbus/v5 v5.1.0
	github.com/gogpu/gputypes v0.3.0
	github.com/gogpu/wgpu v0.21.3
	github.com/pierrec/msdf v0.0.0-20260126203608-76b1ee18a962
	github.com/rivo/uniseg v0.4.7
	github.com/zzl/go-win32api/v2 v2.0.1
	golang.org/x/image v0.29.0
	golang.org/x/text v0.35.0
)

require (
	github.com/andybalholm/cascadia v1.3.3 // indirect
	github.com/gogpu/naga v0.14.8 // indirect
	github.com/tdewolff/parse/v2 v2.8.11 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
)
