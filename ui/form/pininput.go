package form

import (
	"regexp"
	"strings"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/osk"
)

// Layout constants for PIN input.
const (
	pinCellSize = 40
	pinCellGap  = 8
	pinCellPad  = 8
)

// DefaultPinChars matches digits only.
var DefaultPinChars = regexp.MustCompile(`^[0-9]$`)

// PinInput is a fixed-length PIN/code entry widget (RFC-004 §6.5).
type PinInput struct {
	ui.BaseElement

	// Length is the number of digit positions.
	Length int

	// Value is the currently entered string (len <= Length).
	Value string

	// Masked hides entered digits with dots.
	Masked bool

	// OnComplete is called when all positions are filled.
	OnComplete func(pin string)

	// OnChange is called on every character entry.
	OnChange func(partial string)

	// AllowedChars restricts input characters. Default: digits only.
	AllowedChars *regexp.Regexp

	Disabled bool
}

// PinInputOption configures a PinInput element.
type PinInputOption func(*PinInput)

// WithPinMasked enables masked (dot) display.
func WithPinMasked() PinInputOption {
	return func(p *PinInput) { p.Masked = true }
}

// WithOnPinComplete sets the completion callback.
func WithOnPinComplete(fn func(string)) PinInputOption {
	return func(p *PinInput) { p.OnComplete = fn }
}

// WithOnPinChange sets the change callback.
func WithOnPinChange(fn func(string)) PinInputOption {
	return func(p *PinInput) { p.OnChange = fn }
}

// WithPinAllowedChars sets the allowed character regex.
func WithPinAllowedChars(re *regexp.Regexp) PinInputOption {
	return func(p *PinInput) { p.AllowedChars = re }
}

// WithPinDisabled disables the widget.
func WithPinDisabled() PinInputOption {
	return func(p *PinInput) { p.Disabled = true }
}

// NewPinInput creates a PIN input with the given length.
func NewPinInput(length int, value string, opts ...PinInputOption) ui.Element {
	el := PinInput{Length: length, Value: value, AllowedChars: DefaultPinChars}
	for _, o := range opts {
		o(&el)
	}
	return el
}

// IsValidChar checks if a character is allowed for this PIN input.
func (p PinInput) IsValidChar(ch rune) bool {
	re := p.AllowedChars
	if re == nil {
		re = DefaultPinChars
	}
	return re.MatchString(string(ch))
}

// OSKLayout implements osk.OSKRequester (RFC-004 §6.11).
func (p PinInput) OSKLayout() osk.OSKLayout { return osk.OSKLayoutPin }

// LayoutSelf implements ui.Layouter.
func (p PinInput) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	focus := ctx.Focus

	style := tokens.Typography.Body
	length := p.Length
	if length < 1 {
		length = 4
	}

	totalW := length*pinCellSize + (length-1)*pinCellGap
	if area.W < totalW {
		totalW = area.W
	}
	h := pinCellSize

	// Focus management.
	var focused bool
	var focusUID ui.UID
	if focus != nil && !p.Disabled {
		focusUID = focus.NextElementUID()
		focus.RegisterFocusable(focusUID, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(focusUID)
	}

	borderColor := tokens.Colors.Stroke.Border
	if p.Disabled {
		borderColor = ui.DisabledColor(borderColor, tokens.Colors.Surface.Base)
	}

	runes := []rune(p.Value)

	for i := 0; i < length; i++ {
		cellX := area.X + i*(pinCellSize+pinCellGap)
		cellRect := draw.R(float32(cellX), float32(area.Y), float32(pinCellSize), float32(h))

		// Cell background.
		fillColor := tokens.Colors.Surface.Elevated
		if p.Disabled {
			fillColor = ui.DisabledColor(fillColor, tokens.Colors.Surface.Base)
		}
		canvas.FillRoundRect(cellRect, tokens.Radii.Input, draw.SolidPaint(borderColor))
		canvas.FillRoundRect(
			draw.R(float32(cellX+1), float32(area.Y+1), float32(max(pinCellSize-2, 0)), float32(max(h-2, 0))),
			maxf(tokens.Radii.Input-1, 0), draw.SolidPaint(fillColor))

		// Active cell highlight.
		isActiveCell := focused && i == len(runes)
		if isActiveCell {
			ui.DrawFocusRing(canvas, cellRect, tokens.Radii.Input, tokens)
		}

		// Draw character or placeholder.
		if i < len(runes) {
			textColor := tokens.Colors.Text.Primary
			if p.Disabled {
				textColor = tokens.Colors.Text.Disabled
			}

			var ch string
			if p.Masked {
				ch = "\u25CF" // ●
			} else {
				ch = string(runes[i])
			}

			m := canvas.MeasureText(ch, style)
			textX := float32(cellX) + float32(pinCellSize)/2 - m.Width/2
			textY := float32(area.Y) + float32(h)/2 - style.Size/2
			canvas.DrawText(ch, draw.Pt(textX, textY), style, textColor)
		} else if isActiveCell {
			// Cursor indicator.
			cursorX := float32(cellX) + float32(pinCellSize)/2
			cursorY1 := float32(area.Y) + float32(pinCellPad)
			cursorY2 := float32(area.Y) + float32(h-pinCellPad)
			canvas.FillRect(
				draw.R(cursorX-0.5, cursorY1, 1, cursorY2-cursorY1),
				draw.SolidPaint(tokens.Colors.Accent.Primary))
		}
	}

	wholeRect := draw.R(float32(area.X), float32(area.Y), float32(totalW), float32(h))

	// InputState: connect keyboard/OSK input to the widget (like passwordfield.go:187-204).
	if focused && focus != nil {
		cursorOff := len(p.Value)
		if focus.Input != nil && focus.Input.FocusUID == focusUID {
			cursorOff = focus.Input.CursorOffset
			if cursorOff > len(p.Value) {
				cursorOff = len(p.Value)
			}
		}
		onChange := p.OnChange
		onComplete := p.OnComplete
		allowedChars := p.AllowedChars
		if allowedChars == nil {
			allowedChars = DefaultPinChars
		}
		maxLen := length
		focus.Input = &ui.InputState{
			Value: p.Value,
			OnChange: func(newVal string) {
				filtered := filterPinChars(newVal, allowedChars, maxLen)
				if onChange != nil {
					onChange(filtered)
				}
				if len([]rune(filtered)) == maxLen && onComplete != nil {
					onComplete(filtered)
				}
			},
			FocusUID:       focusUID,
			CursorOffset:   cursorOff,
			SelectionStart: -1,
		}
		focus.FocusedBounds = &wholeRect
	}

	// Hit target for focus acquisition (like passwordfield.go:207-212).
	if focus != nil && !p.Disabled {
		uid := focusUID
		fm := focus
		ix.RegisterHit(wholeRect, func() { fm.SetFocusedUID(uid) })
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: totalW, H: h}
}

// TreeEqual implements ui.TreeEqualizer.
func (p PinInput) TreeEqual(other ui.Element) bool {
	pb, ok := other.(PinInput)
	return ok && p.Length == pb.Length && p.Value == pb.Value && p.Masked == pb.Masked
}

// ResolveChildren implements ui.ChildResolver. PinInput is a leaf.
func (p PinInput) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return p
}

// WalkAccess implements ui.AccessWalker.
func (p PinInput) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	value := p.Value
	if p.Masked {
		value = strings.Repeat("\u25CF", len([]rune(p.Value)))
	}
	b.AddNode(a11y.AccessNode{
		Role:   a11y.RoleTextInput,
		Value:  value,
		States: a11y.AccessStates{Disabled: p.Disabled},
	}, parentIdx, a11y.Rect{})
}

// filterPinChars filters a string to only contain allowed characters,
// truncated to maxLen runes.
func filterPinChars(s string, allowed *regexp.Regexp, maxLen int) string {
	var b strings.Builder
	count := 0
	for _, r := range s {
		if count >= maxLen {
			break
		}
		if allowed.MatchString(string(r)) {
			b.WriteRune(r)
			count++
		}
	}
	return b.String()
}

// Compile-time interface checks.
var _ osk.OSKRequester = PinInput{}
