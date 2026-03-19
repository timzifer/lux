// Package fonts provides font loading, fallback chains, and the
// embedded fallback font (RFC §16, RFC-003 §3).
//
// The package embeds Noto Sans as the default fallback and provides
// LoadFile, LoadFS, and LoadBytes for loading additional TTF/OTF fonts.
// The 5×7 bitmap fallback is retained as the ultimate fallback.
package fonts

import (
	"bytes"
	"io/fs"
	"os"
	"strings"
	"unicode"

	gotextfont "github.com/go-text/typesetting/font"
	"golang.org/x/image/font/sfnt"
)

// FontStyle distinguishes normal, italic, and oblique faces.
type FontStyle uint8

const (
	StyleNormal  FontStyle = iota
	StyleItalic
	StyleOblique
)

// FontFaceKey identifies a specific face within a FontFamily.
type FontFaceKey struct {
	Weight int       // 100–900; 400 = Regular
	Style  FontStyle
}

// Font is a loaded TTF/OTF font. Immutable after Load.
// When sfnt is non-nil the font was loaded from a TTF/OTF file;
// when nil the legacy 5×7 bitmap fallback is used.
type Font struct {
	id         uint64 // unique ID, assigned at load time
	name       string
	sfnt       *sfnt.Font
	goTextFace *gotextfont.Face // go-text/typesetting face for OpenType shaping
	rawData    []byte           // retained for go-text parsing (Face holds reference)
}

var nextFontID uint64 = 1

// Name returns the font's display name.
func (f *Font) Name() string { return f.name }

// ID returns a unique identifier for this font instance (used for atlas caching).
func (f *Font) ID() uint64 { return f.id }

// SfntFont returns the parsed sfnt.Font, or nil for the bitmap fallback.
func (f *Font) SfntFont() *sfnt.Font { return f.sfnt }

// IsBitmap reports whether this font uses the legacy bitmap glyphs.
func (f *Font) IsBitmap() bool { return f.sfnt == nil }

// GoTextFace returns the go-text/typesetting Face for OpenType shaping,
// or nil if the font could not be parsed by go-text.
func (f *Font) GoTextFace() *gotextfont.Face { return f.goTextFace }

// HasGlyph reports whether this font contains a glyph for the given rune.
func (f *Font) HasGlyph(r rune) bool {
	if f.goTextFace == nil {
		return false
	}
	_, ok := f.goTextFace.NominalGlyph(r)
	return ok
}

// ── Loading API (RFC-003 §3.4) ──────────────────────────────────

// LoadBytes parses a TTF or OTF font from raw bytes.
func LoadBytes(data []byte) (*Font, error) {
	parsed, err := sfnt.Parse(data)
	if err != nil {
		return nil, err
	}
	id := nextFontID
	nextFontID++

	f := &Font{id: id, sfnt: parsed, rawData: data}

	// Also parse with go-text/typesetting for OpenType shaping.
	// Failure here is non-fatal — the font will still work with sfnt.
	if gtFace, err := gotextfont.ParseTTF(bytes.NewReader(data)); err == nil {
		f.goTextFace = gtFace
	}

	return f, nil
}

// LoadFile loads a TTF or OTF font from a file path.
func LoadFile(path string) (*Font, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadBytes(data)
}

// LoadFS loads a TTF or OTF font from an fs.FS.
func LoadFS(fsys fs.FS, path string) (*Font, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, err
	}
	return LoadBytes(data)
}

// FontFamily groups multiple faces and provides a fallback chain.
type FontFamily struct {
	Name     string
	Faces    map[FontFaceKey]*Font
	Fallback []*FontFamily
}

// FindGlyphFont searches this family's faces (preferring the given weight),
// then walks the Fallback chain recursively, returning the first Font that
// contains a glyph for the given rune. Returns nil if no font has the glyph.
func (ff *FontFamily) FindGlyphFont(r rune, weight int) *Font {
	if ff == nil {
		return nil
	}
	// Try exact weight first.
	key := FontFaceKey{Weight: weight, Style: StyleNormal}
	if f, ok := ff.Faces[key]; ok && f.HasGlyph(r) {
		return f
	}
	// Try regular weight.
	if weight != 400 {
		key.Weight = 400
		if f, ok := ff.Faces[key]; ok && f.HasGlyph(r) {
			return f
		}
	}
	// Try any face in this family.
	for _, f := range ff.Faces {
		if f.HasGlyph(r) {
			return f
		}
	}
	// Walk fallback chain.
	for _, fb := range ff.Fallback {
		if f := fb.FindGlyphFont(r, weight); f != nil {
			return f
		}
	}
	return nil
}

// ── Embedded 5×7 bitmap fallback ─────────────────────────────────

// BitmapGlyph returns the 5×7 bitmap rows for a rune, or the '?' glyph.
func BitmapGlyph(r rune) [7]string {
	g, ok := glyphs[unicode.ToUpper(r)]
	if !ok {
		g = glyphs['?']
	}
	return g
}

// BitmapCharWidth is the advance width of one bitmap character cell.
const BitmapCharWidth = 6

// BitmapCharHeight is the height of one bitmap character cell.
const BitmapCharHeight = 7

// Fallback is the embedded font family used when no other font is available.
// It contains Noto Sans (loaded at init) with the 5×7 bitmap as ultimate fallback.
var Fallback = &FontFamily{
	Name:  "Noto Sans",
	Faces: make(map[FontFaceKey]*Font),
}

// DefaultFont returns the embedded Noto Sans regular font, loading it
// on first access. Returns nil if the embedded font cannot be parsed.
func DefaultFont() *Font {
	initDefaultFont()
	return defaultFont
}

// ── glyph data ───────────────────────────────────────────────────

var glyphs = map[rune][7]string{
	'?': {"11111", "00001", "00010", "00100", "00100", "00000", "00100"},
	'!': {"00100", "00100", "00100", "00100", "00100", "00000", "00100"},
	'-': {"00000", "00000", "11111", "00000", "00000", "00000", "00000"},
	'_': {"00000", "00000", "00000", "00000", "00000", "00000", "11111"},
	'.': {"00000", "00000", "00000", "00000", "00000", "00110", "00110"},
	':': {"00000", "00110", "00110", "00000", "00110", "00110", "00000"},
	'+': {"00000", "00100", "00100", "11111", "00100", "00100", "00000"},
	'(': {"00010", "00100", "01000", "01000", "01000", "00100", "00010"},
	')': {"01000", "00100", "00010", "00010", "00010", "00100", "01000"},
	'0': {"01110", "10001", "10011", "10101", "11001", "10001", "01110"},
	'1': {"00100", "01100", "00100", "00100", "00100", "00100", "01110"},
	'2': {"01110", "10001", "00001", "00010", "00100", "01000", "11111"},
	'3': {"11110", "00001", "00001", "01110", "00001", "00001", "11110"},
	'4': {"00010", "00110", "01010", "10010", "11111", "00010", "00010"},
	'5': {"11111", "10000", "10000", "11110", "00001", "00001", "11110"},
	'6': {"01110", "10000", "10000", "11110", "10001", "10001", "01110"},
	'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
	'8': {"01110", "10001", "10001", "01110", "10001", "10001", "01110"},
	'9': {"01110", "10001", "10001", "01111", "00001", "00001", "01110"},
	'A': {"01110", "10001", "10001", "11111", "10001", "10001", "10001"},
	'B': {"11110", "10001", "10001", "11110", "10001", "10001", "11110"},
	'C': {"01110", "10001", "10000", "10000", "10000", "10001", "01110"},
	'D': {"11110", "10001", "10001", "10001", "10001", "10001", "11110"},
	'E': {"11111", "10000", "10000", "11110", "10000", "10000", "11111"},
	'F': {"11111", "10000", "10000", "11110", "10000", "10000", "10000"},
	'G': {"01110", "10001", "10000", "10111", "10001", "10001", "01110"},
	'H': {"10001", "10001", "10001", "11111", "10001", "10001", "10001"},
	'I': {"01110", "00100", "00100", "00100", "00100", "00100", "01110"},
	'J': {"00001", "00001", "00001", "00001", "10001", "10001", "01110"},
	'K': {"10001", "10010", "10100", "11000", "10100", "10010", "10001"},
	'L': {"10000", "10000", "10000", "10000", "10000", "10000", "11111"},
	'M': {"10001", "11011", "10101", "10101", "10001", "10001", "10001"},
	'N': {"10001", "10001", "11001", "10101", "10011", "10001", "10001"},
	'O': {"01110", "10001", "10001", "10001", "10001", "10001", "01110"},
	'P': {"11110", "10001", "10001", "11110", "10000", "10000", "10000"},
	'Q': {"01110", "10001", "10001", "10001", "10101", "10010", "01101"},
	'R': {"11110", "10001", "10001", "11110", "10100", "10010", "10001"},
	'S': {"01111", "10000", "10000", "01110", "00001", "00001", "11110"},
	'T': {"11111", "00100", "00100", "00100", "00100", "00100", "00100"},
	'U': {"10001", "10001", "10001", "10001", "10001", "10001", "01110"},
	'V': {"10001", "10001", "10001", "10001", "10001", "01010", "00100"},
	'W': {"10001", "10001", "10001", "10101", "10101", "10101", "01010"},
	'X': {"10001", "10001", "01010", "00100", "01010", "10001", "10001"},
	'Y': {"10001", "10001", "01010", "00100", "00100", "00100", "00100"},
	'Z': {"11111", "00001", "00010", "00100", "01000", "10000", "11111"},
}

func init() {
	for ch := 'a'; ch <= 'z'; ch++ {
		glyphs[ch] = glyphs[unicode.ToUpper(ch)]
	}
	for _, ch := range strings.Split("äöüßÄÖÜ", "") {
		if ch == "" {
			continue
		}
		glyphs[[]rune(ch)[0]] = glyphs['?']
	}

	// Load the embedded Noto Sans font into the Fallback family.
	initDefaultFont()

	// Load the embedded Phosphor icon font.
	initPhosphorFont()
}
