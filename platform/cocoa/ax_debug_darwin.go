//go:build darwin && cocoa && !nogui && arm64

package cocoa

import (
	"log"
	"os"
	"strings"
	"sync"
)

var (
	axDebugOnce    sync.Once
	axDebugEnabled bool
)

func axDebugf(format string, args ...any) {
	axDebugOnce.Do(func() {
		switch strings.ToLower(strings.TrimSpace(os.Getenv("LUX_AX_DEBUG"))) {
		case "1", "true", "yes", "on", "debug":
			axDebugEnabled = true
			log.Printf("[lux ax] debug logging enabled")
		}
	})
	if !axDebugEnabled {
		return
	}
	log.Printf("[lux ax] "+format, args...)
}
