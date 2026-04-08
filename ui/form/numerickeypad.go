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
	// OriginalValue is the value when the keypad was opened (for reference).
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
	Input     *ui.InputState // the text field's InputState (for direct editing)
	Focus     *ui.FocusManager
	FocusUID  ui.UID
	Kind      NumericKind
	Step      float64
	Min       *float64
	Max       *float64
	Precision int
	OnSubmit  func(float64) // value accepted, closes keypad
}

// keypad layout constants
const (
	kpCols    = 4
	kpRows    = 5 // 3 digit rows + function row + navigation row
	kpGap     = 4
	kpPadding = 8
)

// renderNumericKeypad draws the keypad into an overlay render callback.
// Key presses write directly into the InputState (the text field's value).
func renderNumericKeypad(cfg numericKeypadConfig, anchor draw.Rect,
	canvas draw.Canvas, tokens theme.TokenSet, ix *ui.Interactor,
	winW, winH int) {

	state := cfg.State
	input := cfg.Input
	if state == nil || !state.Open || input == nil {
		return
	}

	keySize := float32(44)
	gap := float32(kpGap)
	pad := float32(kpPadding)

	totalW := float32(kpCols)*keySize + float32(kpCols-1)*gap + pad*2
	totalH := float32(kpRows)*keySize + float32(kpRows-1)*gap + pad*2

	contentSize := draw.Size{W: totalW, H: totalH}
	pos := ui.ComputeOverlayPosition(anchor, ui.PlacementBelow, contentSize, winW, winH)

	// Full-screen backdrop: clicking outside submits the current value.
	onSubmit := cfg.OnSubmit
	ix.RegisterHit(draw.R(0, 0, float32(winW), float32(winH)), func() {
		v := parseInputValue(input.Value, cfg)
		v = clampKeypad(v, cfg)
		state.Open = false
		if onSubmit != nil {
			onSubmit(v)
		}
	})

	// Background
	bgRect := draw.R(pos.X, pos.Y, totalW, totalH)
	// Eat clicks on the keypad body so they don't trigger the backdrop,
	// and re-assert focus on the input field.
	ix.RegisterHit(bgRect, func() {
		if cfg.Focus != nil {
			cfg.Focus.SetFocusedUID(cfg.FocusUID)
		}
	})
	canvas.FillRoundRect(bgRect, tokens.Radii.Input+2, draw.SolidPaint(tokens.Colors.Surface.Elevated))
	canvas.StrokeRoundRect(bgRect, tokens.Radii.Input+2,
		draw.Stroke{Paint: draw.SolidPaint(tokens.Colors.Stroke.Border), Width: 1})

	// ── Key grid (no display buffer — keys write directly to text field) ──
	gridY := pos.Y + pad
	isFloat := cfg.Kind == NumericFloat

	type keyDef struct {
		label  string
		icon   string // if non-empty, render as Phosphor icon
		action func()
		filled bool // use accent fill
	}

	// Build the 5×4 grid:
	// Row 0: 7 8 9 ▲
	// Row 1: 4 5 6 ▼
	// Row 2: 1 2 3 ⌫
	// Row 3: ± 0 . ✓
	// Row 4: ←     →
	keys := [kpRows][kpCols]keyDef{}

	// Helper: insert character at cursor position in InputState.
	insertChar := func(ch string) {
		off := input.CursorOffset
		if off > len(input.Value) {
			off = len(input.Value)
		}
		newVal := input.Value[:off] + ch + input.Value[off:]
		input.CursorOffset = off + len(ch)
		input.SelectionStart = -1
		if input.OnChange != nil {
			input.OnChange(newVal)
		}
	}

	// Helper: delete character before cursor.
	backspace := func() {
		off := input.CursorOffset
		if off <= 0 || len(input.Value) == 0 {
			return
		}
		if off > len(input.Value) {
			off = len(input.Value)
		}
		newVal := input.Value[:off-1] + input.Value[off:]
		input.CursorOffset = off - 1
		input.SelectionStart = -1
		if input.OnChange != nil {
			input.OnChange(newVal)
		}
	}

	// Helper: clear all text.
	clearAll := func() {
		input.CursorOffset = 0
		input.SelectionStart = -1
		if input.OnChange != nil {
			input.OnChange("")
		}
	}

	// Digit keys (rows 0–2)
	digitOrder := [3][3]rune{
		{'7', '8', '9'},
		{'4', '5', '6'},
		{'1', '2', '3'},
	}
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			ch := string(digitOrder[r][c])
			keys[r][c] = keyDef{
				label:  ch,
				action: func() { insertChar(ch) },
			}
		}
	}

	// Row 0, col 3: Backspace
	keys[0][3] = keyDef{
		icon:   icons.Backspace,
		action: backspace,
	}

	// Row 1, col 3: Clear (delete all)
	keys[1][3] = keyDef{
		icon:   icons.Eraser,
		action: clearAll,
	}

	// Row 3: ±, 0, decimal, ✓ (submit)
	keys[3][0] = keyDef{
		label: "±",
		action: func() {
			val := input.Value
			if strings.HasPrefix(val, "-") {
				newVal := val[1:]
				input.CursorOffset = max(input.CursorOffset-1, 0)
				if input.OnChange != nil {
					input.OnChange(newVal)
				}
			} else {
				newVal := "-" + val
				input.CursorOffset = input.CursorOffset + 1
				if input.OnChange != nil {
					input.OnChange(newVal)
				}
			}
		},
	}
	keys[3][1] = keyDef{
		label:  "0",
		action: func() { insertChar("0") },
	}
	if isFloat {
		keys[3][2] = keyDef{
			label: ".",
			action: func() {
				if !strings.Contains(input.Value, ".") {
					insertChar(".")
				}
			},
		}
	}
	// Submit (row 3, col 3)
	keys[3][3] = keyDef{
		icon:   icons.Check,
		filled: true,
		action: func() {
			v := parseInputValue(input.Value, cfg)
			v = clampKeypad(v, cfg)
			state.Open = false
			if onSubmit != nil {
				onSubmit(v)
			}
		},
	}

	// Row 4: ← ▲ ▼ →  (navigation + increment/decrement)
	keys[4][0] = keyDef{
		icon: icons.CaretLeft,
		action: func() {
			if input.CursorOffset > 0 {
				input.CursorOffset--
			}
			input.SelectionStart = -1
		},
	}
	keys[4][1] = keyDef{
		icon: icons.CaretUp,
		action: func() {
			v := parseInputValue(input.Value, cfg)
			v = clampKeypad(snapToStepKeypad(v+cfg.Step, cfg), cfg)
			newVal := formatKeypadValue(v, cfg.Kind, cfg.Precision)
			input.CursorOffset = len(newVal)
			input.SelectionStart = -1
			if input.OnChange != nil {
				input.OnChange(newVal)
			}
		},
	}
	keys[4][2] = keyDef{
		icon: icons.CaretDown,
		action: func() {
			v := parseInputValue(input.Value, cfg)
			v = clampKeypad(snapToStepKeypad(v-cfg.Step, cfg), cfg)
			newVal := formatKeypadValue(v, cfg.Kind, cfg.Precision)
			input.CursorOffset = len(newVal)
			input.SelectionStart = -1
			if input.OnChange != nil {
				input.OnChange(newVal)
			}
		},
	}
	keys[4][3] = keyDef{
		icon: icons.CaretRight,
		action: func() {
			if input.CursorOffset < len(input.Value) {
				input.CursorOffset++
			}
			input.SelectionStart = -1
		},
	}

	// Render keys
	btnStyle := tokens.Typography.Body
	iconStyle := tokens.Typography.Body
	iconStyle.FontFamily = "Phosphor"

	radius := tokens.Radii.Button

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
			if kd.filled {
				bgColor = tokens.Colors.Accent.Primary
			} else {
				bgColor = tokens.Colors.Surface.Base
			}
			canvas.FillRoundRect(kr, radius, draw.SolidPaint(bgColor))
			if !kd.filled {
				canvas.StrokeRoundRect(kr, radius,
					draw.Stroke{Paint: draw.SolidPaint(tokens.Colors.Stroke.Border), Width: 1})
			}

			// Hover + click — re-assert focus so the field stays focused
			// after the click-blur that the framework applies.
			// Skip re-assertion when the keypad was just closed (submit)
			// to avoid immediately reopening on the next frame.
			fn := kd.action
			focusMgr := cfg.Focus
			focusUID := cfg.FocusUID
			ho := ix.RegisterHit(kr, func() {
				if fn != nil {
					fn()
				}
				if focusMgr != nil && state.Open {
					focusMgr.SetFocusedUID(focusUID)
				}
			})
			if ho > 0 {
				canvas.FillRoundRect(kr, radius,
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

// parseInputValue parses the text field value as float64.
func parseInputValue(s string, cfg numericKeypadConfig) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" {
		return cfg.State.OriginalValue
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return cfg.State.OriginalValue
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
