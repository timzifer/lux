package form

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/osk"
)

// Layout constants for IP input.
const (
	ipSegmentW  = 48
	ipSegmentH  = 32
	ipSepW      = 12
)

// IPVersion selects IPv4 or IPv6 mode (RFC-004 §6.6).
type IPVersion uint8

const (
	IPVersionAuto IPVersion = iota // Detect from input
	IPVersion4                     // 4 segments (0-255)
	IPVersion6                     // 8 hex segments
)

// IPInput is an IPv4/IPv6 address entry widget (RFC-004 §6.6).
type IPInput struct {
	ui.BaseElement

	// Value is the IP address string (e.g. "192.168.1.1" or "::1").
	Value string

	// Version selects the IP version.
	Version IPVersion

	// OnChange is called on every segment change.
	OnChange func(string)

	// OnCommit is called when the IP is complete and valid.
	OnCommit func(string)

	Disabled bool
}

// IPInputOption configures an IPInput element.
type IPInputOption func(*IPInput)

// WithIPVersion sets the IP version.
func WithIPVersion(v IPVersion) IPInputOption {
	return func(ip *IPInput) { ip.Version = v }
}

// WithOnIPChange sets the change callback.
func WithOnIPChange(fn func(string)) IPInputOption {
	return func(ip *IPInput) { ip.OnChange = fn }
}

// WithOnIPCommit sets the commit callback.
func WithOnIPCommit(fn func(string)) IPInputOption {
	return func(ip *IPInput) { ip.OnCommit = fn }
}

// WithIPDisabled disables the widget.
func WithIPDisabled() IPInputOption {
	return func(ip *IPInput) { ip.Disabled = true }
}

// NewIPInput creates an IP input element.
func NewIPInput(value string, opts ...IPInputOption) ui.Element {
	el := IPInput{Value: value, Version: IPVersion4}
	for _, o := range opts {
		o(&el)
	}
	return el
}

// SegmentCount returns 4 for IPv4, 8 for IPv6.
func (ip IPInput) SegmentCount() int {
	v := ip.effectiveVersion()
	if v == IPVersion6 {
		return 8
	}
	return 4
}

func (ip IPInput) effectiveVersion() IPVersion {
	if ip.Version != IPVersionAuto {
		return ip.Version
	}
	if strings.Contains(ip.Value, ":") {
		return IPVersion6
	}
	return IPVersion4
}

// Segments splits the value into segment strings.
func (ip IPInput) Segments() []string {
	count := ip.SegmentCount()
	v := ip.effectiveVersion()

	var sep string
	if v == IPVersion6 {
		sep = ":"
	} else {
		sep = "."
	}

	parts := strings.Split(ip.Value, sep)
	// Pad to expected count.
	for len(parts) < count {
		parts = append(parts, "")
	}
	return parts[:count]
}

// ValidateSegment checks if a segment value is valid for the IP version.
func (ip IPInput) ValidateSegment(seg string, version IPVersion) bool {
	if seg == "" {
		return true // empty is allowed during editing
	}
	if version == IPVersion6 {
		if len(seg) > 4 {
			return false
		}
		for _, ch := range seg {
			if !IsValidHexChar(ch) {
				return false
			}
		}
		return true
	}
	// IPv4: 0-255
	n, err := strconv.Atoi(seg)
	return err == nil && n >= 0 && n <= 255
}

// separator returns "." for IPv4, ":" for IPv6.
func (ip IPInput) separator() string {
	if ip.effectiveVersion() == IPVersion6 {
		return ":"
	}
	return "."
}

// OSKLayout implements osk.OSKRequester (RFC-004 §6.11).
func (ip IPInput) OSKLayout() osk.OSKLayout { return osk.OSKLayoutIP }

// LayoutSelf implements ui.Layouter.
func (ip IPInput) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	focus := ctx.Focus

	style := tokens.Typography.Body
	segCount := ip.SegmentCount()
	sep := ip.separator()
	segments := ip.Segments()

	totalW := segCount*ipSegmentW + (segCount-1)*ipSepW
	if area.W < totalW {
		totalW = area.W
	}
	h := ipSegmentH

	// Focus management.
	var focused bool
	var focusUID ui.UID
	if focus != nil && !ip.Disabled {
		focusUID = focus.NextElementUID()
		focus.RegisterFocusable(focusUID, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(focusUID)
	}

	borderColor := tokens.Colors.Stroke.Border
	if ip.Disabled {
		borderColor = ui.DisabledColor(borderColor, tokens.Colors.Surface.Base)
	}

	for i := 0; i < segCount; i++ {
		segX := area.X + i*(ipSegmentW+ipSepW)
		segRect := draw.R(float32(segX), float32(area.Y), float32(ipSegmentW), float32(h))

		// Segment background.
		fillColor := tokens.Colors.Surface.Elevated
		if ip.Disabled {
			fillColor = ui.DisabledColor(fillColor, tokens.Colors.Surface.Base)
		}
		canvas.FillRoundRect(segRect, tokens.Radii.Input, draw.SolidPaint(borderColor))
		canvas.FillRoundRect(
			draw.R(float32(segX+1), float32(area.Y+1), float32(max(ipSegmentW-2, 0)), float32(max(h-2, 0))),
			maxf(tokens.Radii.Input-1, 0), draw.SolidPaint(fillColor))

		// Segment text.
		seg := ""
		if i < len(segments) {
			seg = segments[i]
		}
		textColor := tokens.Colors.Text.Primary
		if ip.Disabled {
			textColor = tokens.Colors.Text.Disabled
		}
		if seg == "" {
			seg = "___"
			textColor = tokens.Colors.Text.Secondary
		}
		m := canvas.MeasureText(seg, style)
		textX := float32(segX) + float32(ipSegmentW)/2 - m.Width/2
		textY := float32(area.Y) + float32(h)/2 - style.Size/2
		canvas.DrawText(seg, draw.Pt(textX, textY), style, textColor)

		// Separator.
		if i < segCount-1 {
			sepX := segX + ipSegmentW
			sepColor := tokens.Colors.Text.Secondary
			if ip.Disabled {
				sepColor = tokens.Colors.Text.Disabled
			}
			sm := canvas.MeasureText(sep, style)
			sepTextX := float32(sepX) + float32(ipSepW)/2 - sm.Width/2
			sepTextY := float32(area.Y) + float32(h)/2 - style.Size/2
			canvas.DrawText(sep, draw.Pt(sepTextX, sepTextY), style, sepColor)
		}
	}

	outerRect := draw.R(float32(area.X), float32(area.Y), float32(totalW), float32(h))

	// InputState: connect keyboard/OSK input to the widget.
	if focused && focus != nil {
		cursorOff := len(ip.Value)
		if focus.Input != nil && focus.Input.FocusUID == focusUID {
			cursorOff = focus.Input.CursorOffset
			if cursorOff > len(ip.Value) {
				cursorOff = len(ip.Value)
			}
		}
		onChange := ip.OnChange
		onCommit := ip.OnCommit
		version := ip.effectiveVersion()
		focus.Input = &ui.InputState{
			Value: ip.Value,
			OnChange: func(newVal string) {
				filtered := filterIPChars(newVal, version)
				if onChange != nil {
					onChange(filtered)
				}
				if onCommit != nil && isCompleteIP(filtered, version) {
					onCommit(filtered)
				}
			},
			FocusUID:       focusUID,
			CursorOffset:   cursorOff,
			SelectionStart: -1,
		}
	}

	// Hit target for focus acquisition.
	if focus != nil && !ip.Disabled {
		uid := focusUID
		fm := focus
		ix.RegisterHit(outerRect, func() { fm.SetFocusedUID(uid) })
	}

	if focused {
		ui.DrawFocusRing(canvas, outerRect, tokens.Radii.Input, tokens)
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: totalW, H: h}
}

// TreeEqual implements ui.TreeEqualizer.
func (ip IPInput) TreeEqual(other ui.Element) bool {
	ib, ok := other.(IPInput)
	return ok && ip.Value == ib.Value && ip.Version == ib.Version
}

// ResolveChildren implements ui.ChildResolver. IPInput is a leaf.
func (ip IPInput) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return ip
}

// WalkAccess implements ui.AccessWalker.
func (ip IPInput) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	label := "IPv4"
	if ip.effectiveVersion() == IPVersion6 {
		label = "IPv6"
	}
	b.AddNode(a11y.AccessNode{
		Role:   a11y.RoleTextInput,
		Label:  fmt.Sprintf("%s Address", label),
		Value:  ip.Value,
		States: a11y.AccessStates{Disabled: ip.Disabled},
	}, parentIdx, a11y.Rect{})
}

// filterIPChars filters input to only valid IP characters.
func filterIPChars(s string, version IPVersion) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.' && version != IPVersion6:
			b.WriteRune(r)
		case r == ':' && version == IPVersion6:
			b.WriteRune(r)
		case version == IPVersion6 && IsValidHexChar(r):
			b.WriteRune(r)
		}
	}
	return b.String()
}

// isCompleteIP checks if the IP string has all segments filled.
func isCompleteIP(s string, version IPVersion) bool {
	if version == IPVersion6 {
		parts := strings.Split(s, ":")
		return len(parts) == 8 && parts[7] != ""
	}
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || n > 255 {
			return false
		}
	}
	return true
}

// Compile-time interface checks.
var _ osk.OSKRequester = IPInput{}
