package fonts

import (
	"embed"
	"sync"
)

//go:embed phosphor/Phosphor.ttf
var phosphorFS embed.FS

var (
	phosphorFontOnce sync.Once
	phosphorFont     *Font
)

// PhosphorFamily is the icon font family for Phosphor icons.
var PhosphorFamily = &FontFamily{
	Name:  "Phosphor",
	Faces: make(map[FontFaceKey]*Font),
}

func initPhosphorFont() {
	phosphorFontOnce.Do(func() {
		data, err := phosphorFS.ReadFile("phosphor/Phosphor.ttf")
		if err != nil {
			return
		}
		f, err := LoadBytes(data)
		if err != nil {
			return
		}
		f.name = "Phosphor"
		phosphorFont = f
		PhosphorFamily.Faces[FontFaceKey{Weight: 400, Style: StyleNormal}] = f
	})
}

// PhosphorFont returns the embedded Phosphor icon font.
func PhosphorFont() *Font {
	initPhosphorFont()
	return phosphorFont
}
