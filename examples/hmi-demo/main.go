// HMI Demo — industrial machine control panel showcasing interaction profiles.
//
// Demonstrates how Lux widgets adapt to Desktop, Touch, and HMI profiles
// (RFC-004 §2). Starts in HMI mode (64dp touch targets, glove-optimized).
//
//	go run ./examples/hmi-demo/
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/timzifer/lux/app"
	"github.com/timzifer/lux/interaction"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/button"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/form"
	"github.com/timzifer/lux/ui/layout"
	"github.com/timzifer/lux/ui/nav"
	"github.com/timzifer/lux/ui/osk"
)

// ── Model ────────────────────────────────────────────────────────

type Model struct {
	Dark        bool
	ProfileName string
	ActiveTab   int

	// Dashboard
	MotorRunning    bool
	Temperature     float32 // 0–100 °C
	Pressure        float32 // 0–1.0 (displayed as 0–10 bar)
	ProductionCount int

	// Controls
	PumpActive    bool
	ValveOpen     bool
	FanSpeed      float32 // 0–1.0
	ConveyorSpeed float32 // 0–1.0

	// Toggle animation states
	MotorToggle *form.ToggleState
	PumpToggle  *form.ToggleState
	ValveToggle *form.ToggleState

	// Touch feedback states (RFC-004 §4)
	ConfirmMotorState *button.ConfirmButtonState
	HoldStopState     *button.HoldButtonState

	// On-Screen Keyboard demo (RFC-004 §5)
	OSKTextField1 string
	OSKTextField2 string
	OSKMode       osk.OSKMode

	// HMI Widgets demo (RFC-004 §6)
	NumericVal      float64
	StepperVal      int
	PinValue        string
	HexValue        uint64
	IPValue         string
	UnitValue       float64
	UnitSymbol      string
	UnitState       *form.UnitInputState
	RangeLow        float64
	RangeHigh       float64
	TimeValue       time.Time
	DateValue       time.Time
}

func initialModel() Model {
	return Model{
		Dark:              true,
		ProfileName:       "HMI",
		Temperature:       42,
		Pressure:          0.65,
		ProductionCount:   1247,
		FanSpeed:          0.5,
		ConveyorSpeed:     0.3,
		MotorToggle:       form.NewToggleState(),
		PumpToggle:        form.NewToggleState(),
		ValveToggle:       form.NewToggleState(),
		ConfirmMotorState: button.NewConfirmButtonState(),
		HoldStopState:     button.NewHoldButtonState(),
		// HMI Widgets defaults
		NumericVal:  23.5,
		StepperVal:  5,
		HexValue:    0x00FF,
		IPValue:     "192.168.1.1",
		UnitValue:   25.0,
		UnitSymbol:  "mm",
		UnitState:   form.NewUnitInputState(),
		RangeLow:    20,
		RangeHigh:   80,
		TimeValue:   time.Date(2026, 1, 1, 14, 30, 0, 0, time.UTC),
		DateValue:   time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC),
	}
}

// ── Messages ─────────────────────────────────────────────────────

type SelectTabMsg struct{ Index int }
type ToggleThemeMsg struct{}
type SetProfileMsg struct {
	Name    string
	Profile interaction.InteractionProfile
}

type ToggleMotorMsg struct{}
type TogglePumpMsg struct{}
type ToggleValveMsg struct{}
type SetFanSpeedMsg struct{ V float32 }
type SetConveyorSpeedMsg struct{ V float32 }

// Touch-feedback messages (RFC-004 §4)
type ConfirmMotorStartMsg struct{}
type EmergencyHoldCompleteMsg struct{}

// On-Screen Keyboard messages (RFC-004 §5)
type SetOSKField1Msg struct{ V string }
type SetOSKField2Msg struct{ V string }
type SetDemoOSKModeMsg struct{ Mode osk.OSKMode }

// HMI Widget messages (RFC-004 §6)
type SetNumericValMsg struct{ V float64 }
type SetStepperValMsg struct{ V int }
type SetPinValueMsg struct{ V string }
type SetHexValueMsg struct{ V uint64 }
type SetIPValueMsg struct{ V string }
type SetUnitValueMsg struct {
	Value float64
	Unit  string
}
type SetRangeMsg struct {
	Low  float64
	High float64
}
type SetTimeValueMsg struct{ V time.Time }
type SetDateValueMsg struct{ V time.Time }

// ── Update ───────────────────────────────────────────────────────

func update(m Model, msg app.Msg) Model {
	switch msg := msg.(type) {
	case app.TickMsg:
		dt := msg.DeltaTime
		m.MotorToggle.Tick(dt)
		m.PumpToggle.Tick(dt)
		m.ValveToggle.Tick(dt)
		m.ConfirmMotorState.Tick(dt)
		m.HoldStopState.Tick(dt)
		// Simulate slow temperature drift
		if m.MotorRunning {
			m.Temperature += float32(dt.Seconds()) * 2
			if m.Temperature > 95 {
				m.Temperature = 95
			}
		} else {
			m.Temperature -= float32(dt.Seconds()) * 1
			if m.Temperature < 20 {
				m.Temperature = 20
			}
		}
		// Simulate production counter
		if m.MotorRunning && m.PumpActive {
			m.ProductionCount += int(dt.Milliseconds() / 500)
		}

	case app.ModelRestoredMsg:
		app.Send(app.SetDarkModeMsg{Dark: m.Dark})

	case SelectTabMsg:
		m.ActiveTab = msg.Index

	case ToggleThemeMsg:
		m.Dark = !m.Dark
		app.Send(app.SetDarkModeMsg{Dark: m.Dark})

	case SetProfileMsg:
		m.ProfileName = msg.Name
		app.Send(app.SetInteractionProfileMsg{Profile: msg.Profile})

	case ToggleMotorMsg:
		m.MotorRunning = !m.MotorRunning
	case TogglePumpMsg:
		m.PumpActive = !m.PumpActive
	case ToggleValveMsg:
		m.ValveOpen = !m.ValveOpen
	case SetFanSpeedMsg:
		m.FanSpeed = msg.V
	case SetConveyorSpeedMsg:
		m.ConveyorSpeed = msg.V

	case ConfirmMotorStartMsg:
		m.MotorRunning = true

	case EmergencyHoldCompleteMsg:
		m.MotorRunning = false
		m.PumpActive = false

	case SetOSKField1Msg:
		m.OSKTextField1 = msg.V
	case SetOSKField2Msg:
		m.OSKTextField2 = msg.V
	case SetDemoOSKModeMsg:
		m.OSKMode = msg.Mode
		app.Send(app.SetOSKModeMsg{Mode: uint8(msg.Mode)})

	// HMI Widget messages
	case SetNumericValMsg:
		m.NumericVal = msg.V
	case SetStepperValMsg:
		m.StepperVal = msg.V
	case SetPinValueMsg:
		m.PinValue = msg.V
	case SetHexValueMsg:
		m.HexValue = msg.V
	case SetIPValueMsg:
		m.IPValue = msg.V
	case SetUnitValueMsg:
		m.UnitValue = msg.Value
		m.UnitSymbol = msg.Unit
	case SetRangeMsg:
		m.RangeLow = msg.Low
		m.RangeHigh = msg.High
	case SetTimeValueMsg:
		m.TimeValue = msg.V
	case SetDateValueMsg:
		m.DateValue = msg.V
	}
	return m
}

// ── View ─────────────────────────────────────────────────────────

func view(m Model) ui.Element {
	return layout.Column(
		viewToolbar(m),
		display.Divider(),
		nav.New(
			[]nav.TabItem{
				{Header: display.Text("Dashboard"), Content: viewDashboard(m)},
				{Header: display.Text("Controls"), Content: viewControls(m)},
				{Header: display.Text("Touch Feedback"), Content: viewTouchFeedback(m)},
				{Header: display.Text("Keyboard"), Content: viewKeyboard(m)},
				{Header: display.Text("HMI Widgets"), Content: viewHMIWidgets(m)},
				{Header: display.Text("Alarms"), Content: viewAlarms(m)},
			},
			m.ActiveTab,
			func(i int) { app.Send(SelectTabMsg{Index: i}) },
		),
	)
}

// viewToolbar renders the top bar with profile switcher and theme toggle.
func viewToolbar(m Model) ui.Element {
	themeLabel := "DARK"
	if !m.Dark {
		themeLabel = "LIGHT"
	}
	return layout.Row(
		display.Text(fmt.Sprintf("Profile: %s", m.ProfileName)),
		button.Text("Desktop", func() {
			app.Send(SetProfileMsg{"Desktop", interaction.ProfileDesktop})
		}),
		button.Text("Touch", func() {
			app.Send(SetProfileMsg{"Touch", interaction.ProfileTouch})
		}),
		button.Text("HMI", func() {
			app.Send(SetProfileMsg{"HMI", interaction.ProfileHMI})
		}),
		button.Text(themeLabel, func() { app.Send(ToggleThemeMsg{}) }),
	)
}

// ── Dashboard Tab ────────────────────────────────────────────────

func viewDashboard(m Model) ui.Element {
	motorStatus := "OFF"
	if m.MotorRunning {
		motorStatus = "ON"
	}

	return layout.Column(
		display.Text("Machine Status"),
		display.Divider(),

		// Motor status
		display.Card(
			layout.Row(
				display.Text("Motor"),
				statusBadge(motorStatus, m.MotorRunning),
			),
		),

		// Temperature
		display.Card(
			layout.Column(
				display.Text(fmt.Sprintf("Temperature: %.0f °C", m.Temperature)),
				form.NewProgressBar(m.Temperature/100),
			),
		),

		// Pressure
		display.Card(
			layout.Column(
				display.Text(fmt.Sprintf("Pressure: %.1f bar", m.Pressure*10)),
				form.NewProgressBar(m.Pressure),
			),
		),

		// Production counter
		display.Card(
			layout.Row(
				display.Text("Production Count"),
				display.Text(fmt.Sprintf("%d units", m.ProductionCount)),
			),
		),

		// Profile info
		display.Divider(),
		display.Text(fmt.Sprintf(
			"Active Profile: %s — MinTouchTarget: %.0fdp, Spacing: %.0fdp, Typography: %.1fx",
			m.ProfileName,
			profileFor(m.ProfileName).MinTouchTarget,
			profileFor(m.ProfileName).TouchTargetSpacing,
			profileFor(m.ProfileName).ScaleTypography,
		)),
	)
}

// ── Controls Tab ─────────────────────────────────────────────────

func viewControls(m Model) ui.Element {
	return layout.Column(
		display.Text("Machine Controls"),
		display.Divider(),

		// Motor
		display.Card(
			layout.Row(
				display.Text("Motor"),
				form.NewToggle(m.MotorRunning, func(v bool) {
					app.Send(ToggleMotorMsg{})
				}, m.MotorToggle),
			),
		),

		// Pump
		display.Card(
			layout.Row(
				display.Text("Pump"),
				form.NewToggle(m.PumpActive, func(v bool) {
					app.Send(TogglePumpMsg{})
				}, m.PumpToggle),
			),
		),

		// Valve
		display.Card(
			layout.Row(
				display.Text("Valve"),
				form.NewToggle(m.ValveOpen, func(v bool) {
					app.Send(ToggleValveMsg{})
				}, m.ValveToggle),
			),
		),

		// Fan speed
		display.Card(
			layout.Column(
				display.Text(fmt.Sprintf("Fan Speed: %.0f%%", m.FanSpeed*100)),
				form.NewSlider(m.FanSpeed, func(v float32) {
					app.Send(SetFanSpeedMsg{V: v})
				}),
			),
		),

		// Conveyor speed
		display.Card(
			layout.Column(
				display.Text(fmt.Sprintf("Conveyor Speed: %.0f%%", m.ConveyorSpeed*100)),
				form.NewSlider(m.ConveyorSpeed, func(v float32) {
					app.Send(SetConveyorSpeedMsg{V: v})
				}),
			),
		),

		display.Divider(),

		// Emergency stop — requires 2s hold (RFC-004 §4.3 Stufe 3)
		button.Hold("EMERGENCY STOP — HOLD 2s", 2*time.Second, func() {
			app.Send(EmergencyHoldCompleteMsg{})
		}, m.HoldStopState),
	)
}

// ── Touch Feedback Tab (RFC-004 §4 Demo) ────────────────────────

func viewTouchFeedback(m Model) ui.Element {
	motorLabel := "Motor starten"
	if m.MotorRunning {
		motorLabel = "Motor läuft bereits"
	}

	return layout.Column(
		display.Text("Touch-Feedback Demo (RFC-004 §4)"),
		display.Divider(),

		// ConfirmButton demo — two-step confirmation
		display.Card(
			layout.Column(
				display.Text("ConfirmButton — Zwei-Schritt-Bestätigung"),
				display.Text("Erster Tap wechselt in Confirm-Modus, zweiter Tap bestätigt."),
				button.Confirm(motorLabel, "Bestätigen: Motor starten!", func() {
					app.Send(ConfirmMotorStartMsg{})
				}, m.ConfirmMotorState),
			),
		),

		display.Divider(),

		// HoldButton demo — hold-to-confirm
		display.Card(
			layout.Column(
				display.Text("HoldButton — Halten zum Bestätigen"),
				display.Text("Gedrückt halten bis der Fortschrittsring voll ist."),
				button.Hold("NOTFALL-STOPP — 2s HALTEN", 2*time.Second, func() {
					app.Send(EmergencyHoldCompleteMsg{})
				}, m.HoldStopState),
			),
		),

		display.Divider(),

		// Status display
		display.Card(
			layout.Row(
				display.Text("Motor"),
				statusBadge(fmt.Sprintf("%v", m.MotorRunning), m.MotorRunning),
				display.Text("Pump"),
				statusBadge(fmt.Sprintf("%v", m.PumpActive), m.PumpActive),
			),
		),
	)
}

// ── Keyboard Tab (RFC-004 §5 Demo) ──────────────────────────────

func viewKeyboard(m Model) ui.Element {
	modeLabel := func(mode osk.OSKMode) string {
		switch mode {
		case osk.ModeAlpha:
			return "Alpha"
		case osk.ModeNumPad:
			return "NumPad"
		case osk.ModeFull:
			return "Full"
		case osk.ModeCondensed:
			return "Condensed"
		default:
			return "?"
		}
	}

	return layout.Column(
		display.Text("On-Screen Keyboard Demo (RFC-004 §5)"),
		display.Divider(),

		display.Card(
			layout.Column(
				display.Text("Tap a text field to open the OSK (works in Touch/HMI profile)."),
				display.Text("Text Field 1:"),
				form.NewTextField(m.OSKTextField1, "Type here...",
					form.WithOnChange(func(v string) { app.Send(SetOSKField1Msg{V: v}) }),
					form.WithFocus(app.Focus()),
				),
				display.Text("Text Field 2:"),
				form.NewTextField(m.OSKTextField2, "Or here...",
					form.WithOnChange(func(v string) { app.Send(SetOSKField2Msg{V: v}) }),
					form.WithFocus(app.Focus()),
				),
			),
		),

		display.Divider(),

		display.Card(
			layout.Column(
				display.Text(fmt.Sprintf("OSK Mode: %s", modeLabel(m.OSKMode))),
				layout.Row(
					button.Text("Alpha", func() { app.Send(SetDemoOSKModeMsg{Mode: osk.ModeAlpha}) }),
					button.Text("NumPad", func() { app.Send(SetDemoOSKModeMsg{Mode: osk.ModeNumPad}) }),
					button.Text("Full", func() { app.Send(SetDemoOSKModeMsg{Mode: osk.ModeFull}) }),
					button.Text("Condensed", func() { app.Send(SetDemoOSKModeMsg{Mode: osk.ModeCondensed}) }),
				),
			),
		),

		display.Divider(),

		display.Card(
			layout.Column(
				display.Text("Programmatic OSK Control:"),
				layout.Row(
					button.Text("Show OSK (Alpha)", func() {
						app.Send(app.ShowOSKMsg{Layout: uint8(osk.OSKLayoutAlpha)})
					}),
					button.Text("Show OSK (Numeric)", func() {
						app.Send(app.ShowOSKMsg{Layout: uint8(osk.OSKLayoutNumeric)})
					}),
					button.Text("Dismiss OSK", func() {
						app.Send(app.DismissOSKMsg{})
					}),
				),
			),
		),
	)
}

// ── HMI Widgets Tab (RFC-004 §6 Demo) ───────────────────────────

func viewHMIWidgets(m Model) ui.Element {
	units := []form.UnitDef{
		{Symbol: "mm", Label: "Millimeter", Factor: 1},
		{Symbol: "cm", Label: "Centimeter", Factor: 10},
		{Symbol: "m", Label: "Meter", Factor: 1000},
	}

	return layout.Column(
		display.Text("Spezialisierte HMI-Widgets (RFC-004 §6)"),
		display.Divider(),

		// NumericInput
		display.Card(
			layout.Column(
				display.Text("NumericInput — Zahlenwert-Eingabe"),
				form.NewNumericInput(m.NumericVal,
					form.WithNumericLabel("Temperatur"),
					form.WithUnit("°C"),
					form.WithNumericRange(0, 100),
					form.WithNumericStep(0.5),
					form.WithNumericKind(form.NumericFloat),
					form.WithPrecision(1),
					form.WithOnNumericChange(func(v float64) { app.Send(SetNumericValMsg{V: v}) }),
				),
			),
		),

		// Stepper
		display.Card(
			layout.Column(
				display.Text("Stepper — Inkrement/Dekrement"),
				layout.Row(
					form.NewStepper(m.StepperVal,
						form.WithStepperRange(0, 20),
						form.WithStepperStep(1),
						form.WithOnStepperChange(func(v int) { app.Send(SetStepperValMsg{V: v}) }),
					),
					display.Text(fmt.Sprintf("Wert: %d", m.StepperVal)),
				),
			),
		),

		// PinInput
		display.Card(
			layout.Column(
				display.Text("PinInput — PIN-Eingabe"),
				form.NewPinInput(4, m.PinValue,
					form.WithPinMasked(),
					form.WithOnPinChange(func(v string) { app.Send(SetPinValueMsg{V: v}) }),
				),
			),
		),

		// HexInput
		display.Card(
			layout.Column(
				display.Text("HexInput — Hexadezimal-Eingabe"),
				form.NewHexInput(m.HexValue,
					form.WithHexDigits(4),
					form.WithHexPrefix(),
					form.WithHexUpper(),
					form.WithOnHexChange(func(v uint64) { app.Send(SetHexValueMsg{V: v}) }),
				),
			),
		),

		// IPInput
		display.Card(
			layout.Column(
				display.Text("IPInput — IPv4-Adresse"),
				form.NewIPInput(m.IPValue,
					form.WithOnIPChange(func(v string) { app.Send(SetIPValueMsg{V: v}) }),
				),
			),
		),

		// UnitInput
		display.Card(
			layout.Column(
				display.Text("UnitInput — Wert mit Einheit"),
				form.NewUnitInput(m.UnitValue, m.UnitSymbol, units,
					form.WithUnitRange(0, 1000),
					form.WithUnitStep(0.5),
					form.WithUnitState(m.UnitState),
					form.WithOnUnitChange(func(v float64, u string) {
						app.Send(SetUnitValueMsg{Value: v, Unit: u})
					}),
				),
			),
		),

		// RangeInput
		display.Card(
			layout.Column(
				display.Text(fmt.Sprintf("RangeInput — Bereich: %.0f – %.0f", m.RangeLow, m.RangeHigh)),
				form.NewRangeInput(m.RangeLow, m.RangeHigh, 0, 100,
					form.WithRangeStep(5),
					form.WithRangeLabels(),
					form.WithOnRangeChange(func(lo, hi float64) {
						app.Send(SetRangeMsg{Low: lo, High: hi})
					}),
				),
			),
		),

		// TimeInput
		display.Card(
			layout.Column(
				display.Text("TimeInput — Uhrzeit-Eingabe"),
				form.NewTimeInput(m.TimeValue,
					form.WithTimeFormat(form.TimeFormatHHMM),
					form.WithMinuteStep(5),
					form.WithOnTimeInputChange(func(t time.Time) { app.Send(SetTimeValueMsg{V: t}) }),
				),
			),
		),

		// DateInput
		display.Card(
			layout.Column(
				display.Text("DateInput — Datum-Eingabe"),
				form.NewDateInput(m.DateValue,
					form.WithDateMode(form.DateModeNumeric),
					form.WithOnDateInputChange(func(t time.Time) { app.Send(SetDateValueMsg{V: t}) }),
				),
			),
		),
	)
}

// ── Alarms Tab ───────────────────────────────────────────────────

type alarm struct {
	Severity string
	Time     string
	Message  string
}

var demoAlarms = []alarm{
	{"CRITICAL", "14:32:07", "Temperature exceeded 90 °C — motor throttled"},
	{"WARNING", "14:28:51", "Pressure approaching upper limit (9.2 bar)"},
	{"INFO", "14:15:00", "Scheduled maintenance in 2 hours"},
	{"WARNING", "13:58:22", "Conveyor belt slip detected — speed reduced"},
	{"INFO", "13:45:10", "Shift change — operator login required"},
	{"CRITICAL", "12:10:05", "Emergency stop activated by operator"},
	{"INFO", "12:00:00", "System startup complete"},
}

func viewAlarms(m Model) ui.Element {
	children := []ui.Element{
		display.Text("Alarm Log"),
		display.Divider(),
	}
	for _, a := range demoAlarms {
		children = append(children, alarmRow(a))
	}
	return layout.Column(children...)
}

func alarmRow(a alarm) ui.Element {
	return display.Card(
		layout.Row(
			display.BadgeText(a.Severity),
			display.Text(a.Time),
			display.Text(a.Message),
		),
	)
}

// ── Helpers ──────────────────────────────────────────────────────

func statusBadge(label string, _ bool) ui.Element {
	return display.BadgeText(label)
}

func profileFor(name string) interaction.InteractionProfile {
	switch name {
	case "Touch":
		return interaction.ProfileTouch
	case "HMI":
		return interaction.ProfileHMI
	default:
		return interaction.ProfileDesktop
	}
}

// ── Main ─────────────────────────────────────────────────────────

func main() {
	if err := app.Run(initialModel(), update, view,
		app.WithTheme(theme.Default),
		app.WithTitle("HMI Demo — Industrial Control Panel"),
		app.WithSize(1024, 600),
		app.WithInteractionProfile(interaction.ProfileHMI),
	); err != nil {
		log.Fatal(err)
	}
}
