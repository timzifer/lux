package fonts

import (
	"embed"
	"sync"
)

//go:embed noto/NotoSans-Regular.ttf
var notoFS embed.FS

var (
	defaultFontOnce sync.Once
	defaultFont     *Font
)

// initDefaultFont loads the embedded Noto Sans font and registers it
// as the default face in the Fallback family.
func initDefaultFont() {
	defaultFontOnce.Do(func() {
		data, err := notoFS.ReadFile("noto/NotoSans-Regular.ttf")
		if err != nil {
			// Embedded font must always be available.
			return
		}
		f, err := LoadBytes(data)
		if err != nil {
			return
		}
		f.name = "Noto Sans"
		defaultFont = f
		Fallback.Faces[FontFaceKey{Weight: 400, Style: StyleNormal}] = f
	})
}
