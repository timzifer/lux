//go:build !darwin && !linux && !windows

package ui

func zeroCopyPlatform() ZeroCopyMode { return ZeroCopyNone }
