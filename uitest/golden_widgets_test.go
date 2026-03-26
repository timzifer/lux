package uitest

import (
	"testing"
	"time"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/form"
	"github.com/timzifer/lux/ui/layout"
)

const (
	testW = 800
	testH = 600
)

// ── Text ──────────────────────────────────────────────────────────

func TestGoldenText(t *testing.T) {
	scene := BuildScene(display.Text("Hello World"), testW, testH)
	AssertScene(t, scene, "testdata/text.golden")
}

func TestGoldenTextEmpty(t *testing.T) {
	scene := BuildScene(display.Text(""), testW, testH)
	AssertScene(t, scene, "testdata/text_empty.golden")
}

// ── Row / Column ──────────────────────────────────────────────────

func TestGoldenColumn(t *testing.T) {
	scene := BuildScene(
		layout.Column(
			display.Text("First"),
			display.Text("Second"),
			display.Text("Third"),
		),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/column.golden")
}

func TestGoldenRow(t *testing.T) {
	scene := BuildScene(
		layout.Row(
			display.Text("Left"),
			display.Text("Right"),
		),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/row.golden")
}

// ── Stack ─────────────────────────────────────────────────────────

func TestGoldenStack(t *testing.T) {
	scene := BuildScene(
		layout.NewStack(
			display.Text("Bottom"),
			display.Text("Top"),
		),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/stack.golden")
}

// ── Padding ───────────────────────────────────────────────────────

func TestGoldenPadding(t *testing.T) {
	scene := BuildScene(
		layout.Pad(
			draw.Insets{Top: 20, Right: 30, Bottom: 20, Left: 30},
			display.Text("Padded"),
		),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/padding.golden")
}

// ── Card ──────────────────────────────────────────────────────────

func TestGoldenCard(t *testing.T) {
	scene := BuildScene(
		display.Card(
			display.Text("Card Title"),
			display.Text("Card body text"),
		),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/card.golden")
}

// ── Flex Layouts ──────────────────────────────────────────────────

func TestGoldenFlexRow(t *testing.T) {
	scene := BuildScene(
		layout.NewFlex(
			[]ui.Element{
				display.Text("A"),
				display.Text("B"),
				display.Text("C"),
			},
			layout.WithDirection(layout.FlexRow),
			layout.WithGap(10),
		),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/flex_row.golden")
}

func TestGoldenFlexColumnCenter(t *testing.T) {
	scene := BuildScene(
		layout.NewFlex(
			[]ui.Element{
				display.Text("Top"),
				display.Text("Bottom"),
			},
			layout.WithDirection(layout.FlexColumn),
			layout.WithJustify(layout.JustifyCenter),
			layout.WithAlign(layout.AlignCenter),
		),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/flex_column_center.golden")
}

// ── Nested Layout ─────────────────────────────────────────────────

func TestGoldenNestedLayout(t *testing.T) {
	scene := BuildScene(
		layout.Column(
			display.Text("Header"),
			layout.Row(
				display.Text("Left"),
				display.Text("Right"),
			),
			display.Text("Footer"),
		),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/nested_layout.golden")
}

// ── Form Components: Pickers & Numeric ───────────────────────────

func TestGoldenSpinner(t *testing.T) {
	scene := BuildScene(form.NewSpinner(0.0), testW, testH)
	AssertScene(t, scene, "testdata/spinner.golden")
}

func TestGoldenSpinnerMidPhase(t *testing.T) {
	scene := BuildScene(form.NewSpinner(0.5), testW, testH)
	AssertScene(t, scene, "testdata/spinner_mid.golden")
}

func TestGoldenNumericInput(t *testing.T) {
	scene := BuildScene(
		form.NewNumericInput(42, form.WithUnit("px")),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/numeric_input.golden")
}

func TestGoldenNumericInputDisabled(t *testing.T) {
	scene := BuildScene(
		form.NumericInputDisabled(10, form.WithUnit("em")),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/numeric_input_disabled.golden")
}

func TestGoldenColorPicker(t *testing.T) {
	scene := BuildScene(
		form.NewColorPicker(draw.Color{R: 0.25, G: 0.32, B: 0.71, A: 1}),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/color_picker.golden")
}

func TestGoldenTimePicker(t *testing.T) {
	scene := BuildScene(
		form.NewTimePicker(14, 30),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/time_picker.golden")
}

func TestGoldenDatePicker(t *testing.T) {
	scene := BuildScene(
		form.NewDatePicker(time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/date_picker.golden")
}
