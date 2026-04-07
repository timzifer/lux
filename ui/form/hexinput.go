package form

import (
	"fmt"
	"strings"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/osk"
)

// Layout constants for hex input.
const (
	hexCellSize = 32
	hexCellGap  = 4
	hexPrefixW  = 24
)

// HexInput is a hexadecimal value entry widget (RFC-004 §6.9).
type HexInput struct {
	ui.BaseElement

	// Value is the current numeric value.
	Value uint64

	// Digits is the fixed number of hex digits (e.g. 2, 4, 8). 0 = variable.
	Digits int

	// Prefix shows "0x" before the value.
	Prefix bool

	// Upper uses A-F instead of a-f.
	Upper bool

	// OnChange is called when the value changes.
	OnChange func(uint64)

	Disabled bool
}

// HexInputOption configures a HexInput element.
type HexInputOption func(*HexInput)

// WithHexDigits sets the fixed digit count.
func WithHexDigits(d int) HexInputOption {
	return func(h *HexInput) { h.Digits = d }
}

// WithHexPrefix enables "0x" prefix display.
func WithHexPrefix() HexInputOption {
	return func(h *HexInput) { h.Prefix = true }
}

// WithHexUpper uses uppercase A-F.
func WithHexUpper() HexInputOption {
	return func(h *HexInput) { h.Upper = true }
}

// WithOnHexChange sets the change callback.
func WithOnHexChange(fn func(uint64)) HexInputOption {
	return func(h *HexInput) { h.OnChange = fn }
}

// WithHexDisabled disables the widget.
func WithHexDisabled() HexInputOption {
	return func(h *HexInput) { h.Disabled = true }
}

// NewHexInput creates a hex input element.
func NewHexInput(value uint64, opts ...HexInputOption) ui.Element {
	el := HexInput{Value: value, Digits: 4, Prefix: true, Upper: true}
	for _, o := range opts {
		o(&el)
	}
	return el
}

// IsValidHexChar checks if a rune is a valid hex digit.
func IsValidHexChar(ch rune) bool {
	return (ch >= '0' && ch <= '9') ||
		(ch >= 'a' && ch <= 'f') ||
		(ch >= 'A' && ch <= 'F')
}

// FormatHex formats the value as a hex string.
func (h HexInput) FormatHex() string {
	digits := h.Digits
	if digits <= 0 {
		digits = 1
	}
	format := fmt.Sprintf("%%0%dX", digits)
	if !h.Upper {
		format = fmt.Sprintf("%%0%dx", digits)
	}
	return fmt.Sprintf(format, h.Value)
}

// OSKLayout implements osk.OSKRequester (RFC-004 §6.11).
func (h HexInput) OSKLayout() osk.OSKLayout { return osk.OSKLayoutHex }

// LayoutSelf implements ui.Layouter.
func (h HexInput) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	focus := ctx.Focus

	style := tokens.Typography.Body
	digits := h.Digits
	if digits <= 0 {
		digits = 4
	}

	prefixW := 0
	if h.Prefix {
		prefixW = hexPrefixW
	}
	totalW := prefixW + digits*hexCellSize + (digits-1)*hexCellGap
	if area.W < totalW {
		totalW = area.W
	}
	height := hexCellSize

	// Focus management.
	var focused bool
	var focusUID ui.UID
	if focus != nil && !h.Disabled {
		focusUID = focus.NextElementUID()
		focus.RegisterFocusable(focusUID, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(focusUID)
	}

	borderColor := tokens.Colors.Stroke.Border
	if h.Disabled {
		borderColor = ui.DisabledColor(borderColor, tokens.Colors.Surface.Base)
	}

	// Draw "0x" prefix.
	if h.Prefix {
		prefixColor := tokens.Colors.Text.Secondary
		if h.Disabled {
			prefixColor = tokens.Colors.Text.Disabled
		}
		prefixStyle := style
		canvas.DrawText("0x", draw.Pt(
			float32(area.X),
			float32(area.Y)+float32(height)/2-prefixStyle.Size/2,
		), prefixStyle, prefixColor)
	}

	// Format hex string.
	hexStr := h.FormatHex()
	hexRunes := []rune(hexStr)

	for i := 0; i < digits; i++ {
		cellX := area.X + prefixW + i*(hexCellSize+hexCellGap)
		cellRect := draw.R(float32(cellX), float32(area.Y), float32(hexCellSize), float32(height))

		// Cell background.
		fillColor := tokens.Colors.Surface.Elevated
		if h.Disabled {
			fillColor = ui.DisabledColor(fillColor, tokens.Colors.Surface.Base)
		}
		canvas.FillRoundRect(cellRect, tokens.Radii.Input, draw.SolidPaint(borderColor))
		canvas.FillRoundRect(
			draw.R(float32(cellX+1), float32(area.Y+1), float32(max(hexCellSize-2, 0)), float32(max(height-2, 0))),
			maxf(tokens.Radii.Input-1, 0), draw.SolidPaint(fillColor))

		// Draw hex digit.
		textColor := tokens.Colors.Text.Primary
		if h.Disabled {
			textColor = tokens.Colors.Text.Disabled
		}

		var ch string
		if i < len(hexRunes) {
			ch = string(hexRunes[i])
		} else {
			ch = "_"
			textColor = tokens.Colors.Text.Secondary
		}

		m := canvas.MeasureText(ch, style)
		textX := float32(cellX) + float32(hexCellSize)/2 - m.Width/2
		textY := float32(area.Y) + float32(height)/2 - style.Size/2
		canvas.DrawText(ch, draw.Pt(textX, textY), style, textColor)
	}

	wholeRect := draw.R(float32(area.X), float32(area.Y), float32(totalW), float32(height))

	// InputState: connect keyboard/OSK input to the widget.
	if focused && focus != nil {
		hexStr := h.FormatHex()
		cursorOff := len(hexStr)
		if focus.Input != nil && focus.Input.FocusUID == focusUID {
			cursorOff = focus.Input.CursorOffset
			if cursorOff > len(hexStr) {
				cursorOff = len(hexStr)
			}
		}
		onChange := h.OnChange
		upper := h.Upper
		maxDigits := digits
		focus.Input = &ui.InputState{
			Value: hexStr,
			OnChange: func(newVal string) {
				filtered := filterHexChars(newVal, upper, maxDigits)
				if onChange != nil {
					if v, ok := ParseHex(filtered); ok {
						onChange(v)
					}
				}
			},
			FocusUID:       focusUID,
			CursorOffset:   cursorOff,
			SelectionStart: -1,
		}
	}

	// Hit target for focus acquisition.
	if focus != nil && !h.Disabled {
		uid := focusUID
		fm := focus
		ix.RegisterHit(wholeRect, func() { fm.SetFocusedUID(uid) })
	}

	if focused {
		ui.DrawFocusRing(canvas, wholeRect, tokens.Radii.Input, tokens)
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: totalW, H: height}
}

// TreeEqual implements ui.TreeEqualizer.
func (h HexInput) TreeEqual(other ui.Element) bool {
	hb, ok := other.(HexInput)
	return ok && h.Value == hb.Value && h.Digits == hb.Digits && h.Upper == hb.Upper
}

// ResolveChildren implements ui.ChildResolver. HexInput is a leaf.
func (h HexInput) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return h
}

// WalkAccess implements ui.AccessWalker.
func (h HexInput) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	val := h.FormatHex()
	if h.Prefix {
		val = "0x" + val
	}
	b.AddNode(a11y.AccessNode{
		Role:   a11y.RoleTextInput,
		Value:  val,
		States: a11y.AccessStates{Disabled: h.Disabled},
	}, parentIdx, a11y.Rect{})
}

// ParseHex parses a hex string (with optional "0x" prefix) to uint64.
func ParseHex(s string) (uint64, bool) {
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	if s == "" {
		return 0, false
	}
	var val uint64
	for _, ch := range s {
		var d uint64
		switch {
		case ch >= '0' && ch <= '9':
			d = uint64(ch - '0')
		case ch >= 'a' && ch <= 'f':
			d = uint64(ch-'a') + 10
		case ch >= 'A' && ch <= 'F':
			d = uint64(ch-'A') + 10
		default:
			return 0, false
		}
		val = val*16 + d
	}
	return val, true
}

// filterHexChars filters a string to only valid hex characters, truncated to maxDigits.
func filterHexChars(s string, upper bool, maxDigits int) string {
	var b strings.Builder
	count := 0
	for _, r := range s {
		if count >= maxDigits {
			break
		}
		if IsValidHexChar(r) {
			if upper && r >= 'a' && r <= 'f' {
				r = r - 'a' + 'A'
			} else if !upper && r >= 'A' && r <= 'F' {
				r = r - 'A' + 'a'
			}
			b.WriteRune(r)
			count++
		}
	}
	return b.String()
}

// Compile-time interface checks.
var _ osk.OSKRequester = HexInput{}
