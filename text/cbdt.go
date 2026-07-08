package text

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
)

// CBDTAtlas provides access to color bitmap glyphs stored in CBDT/CBLC
// tables of an OpenType font (used by Noto Color Emoji and similar fonts).
type CBDTAtlas struct {
	cblc []byte // raw CBLC table data
	cbdt []byte // raw CBDT table data

	// Parsed strike info (we pick the first/largest strike).
	idxSubTableArrayOff uint32
	numIdxSubTables     uint32
	ppem                uint8
}

// ParseCBDT creates a CBDTAtlas from raw font file bytes.
// Returns nil if the font does not contain CBDT/CBLC tables.
func ParseCBDT(fontData []byte) *CBDTAtlas {
	if len(fontData) < 12 {
		return nil
	}
	numTables := binary.BigEndian.Uint16(fontData[4:6])

	var cblcOff, cblcLen, cbdtOff, cbdtLen uint32
	for i := 0; i < int(numTables); i++ {
		rec := 12 + i*16
		if rec+16 > len(fontData) {
			return nil
		}
		tag := string(fontData[rec : rec+4])
		off := binary.BigEndian.Uint32(fontData[rec+8 : rec+12])
		ln := binary.BigEndian.Uint32(fontData[rec+12 : rec+16])
		switch tag {
		case "CBLC":
			cblcOff, cblcLen = off, ln
		case "CBDT":
			cbdtOff, cbdtLen = off, ln
		}
	}
	if cblcLen == 0 || cbdtLen == 0 {
		return nil
	}

	cblc := fontData[cblcOff : cblcOff+cblcLen]
	cbdt := fontData[cbdtOff : cbdtOff+cbdtLen]

	// Need at least the header + one BitmapSize record.
	if len(cblc) < 8+48 {
		return nil
	}

	numSizes := binary.BigEndian.Uint32(cblc[4:8])
	if numSizes == 0 {
		return nil
	}

	// Use the first strike (typically the only one / largest ppem).
	strikeOff := 8
	idxSubTableArrayOff := binary.BigEndian.Uint32(cblc[strikeOff : strikeOff+4])
	numIdxSubTables := binary.BigEndian.Uint32(cblc[strikeOff+8 : strikeOff+12])
	ppem := cblc[strikeOff+44]

	return &CBDTAtlas{
		cblc:                cblc,
		cbdt:                cbdt,
		idxSubTableArrayOff: idxSubTableArrayOff,
		numIdxSubTables:     numIdxSubTables,
		ppem:                ppem,
	}
}

// PPEM returns the pixels-per-em of the bitmap strike.
func (c *CBDTAtlas) PPEM() int {
	return int(c.ppem)
}

// ColorGlyphMetrics holds metrics for a color bitmap glyph.
type ColorGlyphMetrics struct {
	Width    int
	Height   int
	BearingX int
	BearingY int
	Advance  int
}

// RasterizeGlyph extracts the PNG bitmap for the given glyph ID.
// Returns nil if the glyph is not found in the CBDT tables.
func (c *CBDTAtlas) RasterizeGlyph(glyphID uint16) (*image.NRGBA, *ColorGlyphMetrics) {
	// Scan IndexSubTableArray entries (each 8 bytes).
	for i := uint32(0); i < c.numIdxSubTables; i++ {
		entryOff := int(c.idxSubTableArrayOff) + int(i)*8
		if entryOff+8 > len(c.cblc) {
			break
		}
		firstGlyph := binary.BigEndian.Uint16(c.cblc[entryOff : entryOff+2])
		lastGlyph := binary.BigEndian.Uint16(c.cblc[entryOff+2 : entryOff+4])

		if glyphID < firstGlyph || glyphID > lastGlyph {
			continue
		}

		addlOff := binary.BigEndian.Uint32(c.cblc[entryOff+4 : entryOff+8])
		hdrOff := int(c.idxSubTableArrayOff) + int(addlOff)
		if hdrOff+8 > len(c.cblc) {
			continue
		}

		idxFormat := binary.BigEndian.Uint16(c.cblc[hdrOff : hdrOff+2])
		imgFormat := binary.BigEndian.Uint16(c.cblc[hdrOff+2 : hdrOff+4])
		imgDataOff := binary.BigEndian.Uint32(c.cblc[hdrOff+4 : hdrOff+8])

		if imgFormat != 17 && imgFormat != 18 && imgFormat != 19 {
			continue // unsupported image format
		}

		var glyphDataOff uint32
		glyphIdx := int(glyphID - firstGlyph)

		switch idxFormat {
		case 1: // Offset32 array
			sbitOff := hdrOff + 8 + glyphIdx*4
			if sbitOff+8 > len(c.cblc) {
				continue
			}
			glyphDataOff = binary.BigEndian.Uint32(c.cblc[sbitOff : sbitOff+4])
		case 3: // Offset16 array
			sbitOff := hdrOff + 8 + glyphIdx*2
			if sbitOff+4 > len(c.cblc) {
				continue
			}
			glyphDataOff = uint32(binary.BigEndian.Uint16(c.cblc[sbitOff : sbitOff+2]))
		default:
			continue
		}

		absOff := int(imgDataOff + glyphDataOff)
		if absOff+9 > len(c.cbdt) {
			continue
		}

		var metrics ColorGlyphMetrics
		var pngStart int

		switch imgFormat {
		case 17: // SmallGlyphMetrics (5 bytes) + dataLen (uint32) + PNG
			metrics.Height = int(c.cbdt[absOff])
			metrics.Width = int(c.cbdt[absOff+1])
			metrics.BearingX = int(int8(c.cbdt[absOff+2]))
			metrics.BearingY = int(int8(c.cbdt[absOff+3]))
			metrics.Advance = int(c.cbdt[absOff+4])
			dataLen := binary.BigEndian.Uint32(c.cbdt[absOff+5 : absOff+9])
			pngStart = absOff + 9
			if pngStart+int(dataLen) > len(c.cbdt) {
				continue
			}
		default:
			continue
		}

		// Decode the PNG.
		img, err := png.Decode(bytes.NewReader(c.cbdt[pngStart:]))
		if err != nil {
			continue
		}

		// Convert to NRGBA if needed.
		bounds := img.Bounds()
		nrgba, ok := img.(*image.NRGBA)
		if !ok {
			nrgba = image.NewNRGBA(bounds)
			for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					r, g, b, a := img.At(x, y).RGBA()
					nrgba.SetNRGBA(x, y, color.NRGBA{
						R: uint8(r >> 8), G: uint8(g >> 8),
						B: uint8(b >> 8), A: uint8(a >> 8),
					})
				}
			}
		}

		return nrgba, &metrics
	}
	return nil, nil
}
