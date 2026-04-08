package form

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/icons"
)

// NumericKeypadState holds the overlay state for an inline numeric keypad.
type NumericKeypadState struct {
	// Open is true while the keypad overlay is visible.
	Open bool
	// EditingValue is the text buffer during editing (user types digits here).
	EditingValue string
	// OriginalValue is the value when the keypad was opened (for Cancel).
	OriginalValue float64
}

// NewNumericKeypadState creates a new keypad state.
func NewNumericKeypadState() *NumericKeypadState { return &NumericKeypadState{} }

// keypadRegistry is a package-level map that caches keypad state per focus UID.
// This allows NumericInput (a value type) to maintain persistent keypad state
// across frames without requiring the user to manage it.
var (
	keypadRegistry   = map[ui.UID]*NumericKeypadState{}
	keypadRegistryMu sync.Mutex
)

// getKeypadState returns or creates a keypad state for the given UID.
func getKeypadState(uid ui.UID) *NumericKeypadState {
	keypadRegistryMu.Lock()
	defer keypadRegistryMu.Unlock()
	s, ok := keypadRegistry[uid]
	if !ok {
		s = &NumericKeypadState{}
		keypadRegistry[uid] = s
	}
	return s
}

// cleanupKeypadState removes the keypad state for a UID when no longer focused.
func cleanupKeypadState(uid ui.UID) {
	keypadRegistryMu.Lock()
	defer keypadRegistryMu.Unlock()
	delete(keypadRegistry, uid)
}

// numericKeypadConfig holds the parameters for rendering the keypad overlay.
type numericKeypadConfig struct {
	State     *NumericKeypadState
	Kind      NumericKind
	Step      float64
	Min       *float64
	Max       *float64
	Precision int
	OnSubmit  func(float64) // value accepted
	OnCancel  func()        // editing cancelled
}

// keypad layout constants
const (
	kpCols    = 4
	kpRows    = 4
	kpGap     = 4
	kpPadding = 8
)

// renderNumericKeypad draws the keypad into an overlay render callback.
// anchor is the field rect the overlay is positioned relative to.
func renderNumericKeypad(cfg numericKeypadConfig, anchor draw.Rect,
	canvas draw.Canvas, tokens theme.TokenSet, ix *ui.Interactor,
	winW, winH int) {

	state := cfg.State
	if state == nil || !state.Open {
		return
	}

	// Determine key size from profile-like heuristic. Use a compact 44dp default.
	keySize := float32(44)
	gap := float32(kpGap)
	pad := float32(kpPadding)

	totalW := float32(kpCols)*keySize + float32(kpCols-1)*gap + pad*2
	totalH := float32(kpRows)*keySize + float32(kpRows-1)*gap + pad*2

	// Display buffer row above the grid.
	bufferH := float32(32)
	totalH += bufferH + gap

	contentSize := draw.Size{W: totalW, H: totalH}

	pos := ui.ComputeOverlayPosition(anchor, ui.PlacementBelow, contentSize, winW, winH)

	// Full-screen backdrop: clicking outside dismisses.
	onCancel := cfg.OnCancel
	ix.RegisterHit(draw.R(0, 0, float32(winW), float32(winH)), func() {
		if onCancel != nil {
			onCancel()
		}
	})

	// Background
	bgRect := draw.R(pos.X, pos.Y, totalW, totalH)
	canvas.FillRoundRect(bgRect, tokens.Radii.Input+2, draw.SolidPaint(tokens.Colors.Surface.Elevated))
	canvas.StrokeRoundRect(bgRect, tokens.Radii.Input+2,
		draw.Stroke{Paint: draw.SolidPaint(tokens.Colors.Stroke.Border), Width: 1})

	// ── Display buffer ──
	bufY := pos.Y + pad
	bufRect := draw.R(pos.X+pad, bufY, totalW-pad*2, bufferH)
	canvas.FillRoundRect(bufRect, tokens.Radii.Input,
		draw.SolidPaint(tokens.Colors.Surface.Base))

	displayStr := state.EditingValue
	if displayStr == "" {
		displayStr = formatKeypadValue(state.OriginalValue, cfg.Kind, cfg.Precision)
	}
	style := tokens.Typography.Body
	tm := canvas.MeasureText(displayStr, style)
	textX := bufRect.X + bufRect.W - tm.Width - 4
	textY := bufY + bufferH/2 - style.Size/2
	canvas.DrawText(displayStr, draw.Pt(textX, textY), style, tokens.Colors.Text.Primary)

	// ── Key grid ──
	gridY := bufY + bufferH + gap

	isFloat := cfg.Kind == NumericFloat

	type keyDef struct {
		label  string
		icon   string // if non-empty, render as Phosphor icon
		action func()
		filled bool // use accent fill
		ghost  bool // minimal style
	}

	// Build the 4×4 grid:
	// Row 0: 7 8 9 ▲
	// Row 1: 4 5 6 ▼
	// Row 2: 1 2 3 OK
	// Row 3: ± 0 . ✕
	keys := [kpRows][kpCols]keyDef{}

	// Digit keys
	digitOrder := [3][3]rune{
		{'7', '8', '9'},
		{'4', '5', '6'},
		{'1', '2', '3'},
	}
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			ch := digitOrder[r][c]
			charStr := string(ch)
			keys[r][c] = keyDef{
				label: charStr,
				action: func() {
					state.EditingValue += charStr
				},
			}
		}
	}

	// Row 3: ±, 0, decimal, cancel
	keys[3][0] = keyDef{
		label: "±",
		action: func() {
			if strings.HasPrefix(state.EditingValue, "-") {
				state.EditingValue = state.EditingValue[1:]
			} else if state.EditingValue != "" {
				state.EditingValue = "-" + state.EditingValue
			} else {
				// Toggle the original value's sign
				state.EditingValue = formatKeypadValue(-state.OriginalValue, cfg.Kind, cfg.Precision)
			}
		},
	}
	keys[3][1] = keyDef{
		label: "0",
		action: func() {
			state.EditingValue += "0"
		},
	}
	if isFloat {
		keys[3][2] = keyDef{
			label: ".",
			action: func() {
				if !strings.Contains(state.EditingValue, ".") {
					if state.EditingValue == "" || state.EditingValue == "-" {
						state.EditingValue += "0"
					}
					state.EditingValue += "."
				}
			},
		}
	} else {
		// Backspace for integer mode
		keys[3][2] = keyDef{
			label: "⌫",
			action: func() {
				if len(state.EditingValue) > 0 {
					state.EditingValue = state.EditingValue[:len(state.EditingValue)-1]
				}
			},
		}
	}

	// Cancel (bottom-right)
	keys[3][3] = keyDef{
		icon:  icons.X,
		ghost: true,
		action: func() {
			if onCancel != nil {
				onCancel()
			}
		},
	}

	// Increment (col 3, row 0)
	keys[0][3] = keyDef{
		icon: icons.CaretUp,
		action: func() {
			v := resolveKeypadValue(state, cfg)
			v = clampKeypad(snapToStepKeypad(v+cfg.Step, cfg), cfg)
			state.EditingValue = formatKeypadValue(v, cfg.Kind, cfg.Precision)
		},
	}

	// Decrement (col 3, row 1)
	keys[1][3] = keyDef{
		icon: icons.CaretDown,
		action: func() {
			v := resolveKeypadValue(state, cfg)
			v = clampKeypad(snapToStepKeypad(v-cfg.Step, cfg), cfg)
			state.EditingValue = formatKeypadValue(v, cfg.Kind, cfg.Precision)
		},
	}

	// OK / Submit (col 3, row 2)
	onSubmit := cfg.OnSubmit
	keys[2][3] = keyDef{
		icon:   icons.Check,
		filled: true,
		action: func() {
			v := resolveKeypadValue(state, cfg)
			v = clampKeypad(v, cfg)
			if onSubmit != nil {
				onSubmit(v)
			}
		},
	}

	// Render keys
	btnStyle := tokens.Typography.Body
	iconStyle := tokens.Typography.Body
	iconStyle.FontFamily = "Phosphor"

	for r := 0; r < kpRows; r++ {
		for c := 0; c < kpCols; c++ {
			kd := keys[r][c]
			if kd.action == nil && kd.label == "" && kd.icon == "" {
				continue
			}

			kx := pos.X + pad + float32(c)*(keySize+gap)
			ky := gridY + float32(r)*(keySize+gap)
			kr := draw.R(kx, ky, keySize, keySize)

			// Background
			var bgColor draw.Color
			switch {
			case kd.filled:
				bgColor = tokens.Colors.Accent.Primary
			case kd.ghost:
				bgColor = draw.Color{A: 0}
			default:
				bgColor = tokens.Colors.Surface.Base
			}
			if bgColor.A > 0 {
				canvas.FillRoundRect(kr, tokens.Radii.Button, draw.SolidPaint(bgColor))
			}
			if !kd.filled && !kd.ghost {
				canvas.StrokeRoundRect(kr, tokens.Radii.Button,
					draw.Stroke{Paint: draw.SolidPaint(tokens.Colors.Stroke.Border), Width: 1})
			}

			// Hover + click
			fn := kd.action
			ho := ix.RegisterHit(kr, fn)
			if ho > 0 {
				canvas.FillRoundRect(kr, tokens.Radii.Button,
					draw.SolidPaint(draw.Color{A: ho * 0.10}))
			}

			// Label
			textColor := tokens.Colors.Text.Primary
			if kd.filled {
				textColor = tokens.Colors.Text.OnAccent
			}

			if kd.icon != "" {
				im := canvas.MeasureText(kd.icon, iconStyle)
				ix0 := kx + keySize/2 - im.Width/2
				iy0 := ky + keySize/2 - iconStyle.Size/2
				canvas.DrawText(kd.icon, draw.Pt(ix0, iy0), iconStyle, textColor)
			} else {
				lm := canvas.MeasureText(kd.label, btnStyle)
				lx := kx + keySize/2 - lm.Width/2
				ly := ky + keySize/2 - btnStyle.Size/2
				canvas.DrawText(kd.label, draw.Pt(lx, ly), btnStyle, textColor)
			}
		}
	}
}

// resolveKeypadValue parses the editing buffer, falling back to OriginalValue.
func resolveKeypadValue(state *NumericKeypadState, cfg numericKeypadConfig) float64 {
	if state.EditingValue == "" {
		return state.OriginalValue
	}
	v, err := strconv.ParseFloat(state.EditingValue, 64)
	if err != nil {
		return state.OriginalValue
	}
	return v
}

func clampKeypad(v float64, cfg numericKeypadConfig) float64 {
	if cfg.Min != nil && v < *cfg.Min {
		return *cfg.Min
	}
	if cfg.Max != nil && v > *cfg.Max {
		return *cfg.Max
	}
	return v
}

func snapToStepKeypad(v float64, cfg numericKeypadConfig) float64 {
	if cfg.Step <= 0 {
		return v
	}
	base := 0.0
	if cfg.Min != nil {
		base = *cfg.Min
	}
	return math.Round((v-base)/cfg.Step)*cfg.Step + base
}

func formatKeypadValue(v float64, kind NumericKind, precision int) string {
	if kind == NumericFloat && precision > 0 {
		return fmt.Sprintf("%.*f", precision, v)
	}
	if kind == NumericInteger {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.2f", v)
}
