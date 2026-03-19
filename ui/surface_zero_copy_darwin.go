//go:build darwin

package ui

func zeroCopyPlatform() ZeroCopyMode { return ZeroCopyIOSurface }
