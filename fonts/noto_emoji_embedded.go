package fonts

import (
	"embed"
	"sync"
)

//go:embed noto/NotoColorEmoji.ttf
var notoEmojiFS embed.FS

var (
	notoEmojiFontOnce sync.Once
	notoEmojiFont     *Font
)

// NotoEmojiFamily is the color emoji font family used as fallback for emoji glyphs.
var NotoEmojiFamily = &FontFamily{
	Name:  "Noto Emoji",
	Faces: make(map[FontFaceKey]*Font),
}

func initNotoEmojiFont() {
	notoEmojiFontOnce.Do(func() {
		data, err := notoEmojiFS.ReadFile("noto/NotoColorEmoji.ttf")
		if err != nil {
			return
		}
		f, err := LoadBytes(data)
		if err != nil {
			return
		}
		f.name = "Noto Emoji"
		notoEmojiFont = f
		NotoEmojiFamily.Faces[FontFaceKey{Weight: 400, Style: StyleNormal}] = f
	})
}

// NotoEmojiFont returns the embedded Noto Color Emoji font.
func NotoEmojiFont() *Font {
	initNotoEmojiFont()
	return notoEmojiFont
}
