//go:build windows

package ui

func zeroCopyPlatform() ZeroCopyMode { return ZeroCopyDXGI }
