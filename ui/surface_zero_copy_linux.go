//go:build linux

package ui

func zeroCopyPlatform() ZeroCopyMode { return ZeroCopyDMABuf }
