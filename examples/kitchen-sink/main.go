// Kitchen Sink — demonstrates all Lux widgets (Tier 1 + Tier 2 + M5).
//
// Split-view layout: Tree navigation on the left, active test case on the right.
// Showcases Flex, Grid, Padding, SizedBox, VirtualList, Tree, and RichText.
//
//	go run ./examples/kitchen-sink/
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"runtime"
	"strings"
	"time"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/anim"
	"github.com/timzifer/lux/app"
	"github.com/timzifer/lux/dialog"
	"github.com/timzifer/lux/draw"
	luximage "github.com/timzifer/lux/image"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/platform"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/button"
	"github.com/timzifer/lux/ui/data"
	uidialog "github.com/timzifer/lux/ui/dialog"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/effects"
	"github.com/timzifer/lux/ui/form"
	"github.com/timzifer/lux/ui/icons"
	"github.com/timzifer/lux/ui/layout"
	"github.com/timzifer/lux/ui/link"
	"github.com/timzifer/lux/ui/menu"
	"github.com/timzifer/lux/ui/nav"
	"github.com/timzifer/lux/validation"

	"github.com/timzifer/lux/internal/text"
	"github.com/timzifer/lux/richtext"
)

// ── Section Registry ──────────────────────────────────────────────

// sectionIDs lists the top-level group nodes for the navigation tree.
var sectionIDs = []string{
	"group-basics",
	"group-input",
	"group-layout",
	"group-data",
	"group-navigation",
	"group-overlays",
	"group-images",
	"group-animation",
	"group-theming",
	"group-i18n",
	"group-platform",
	"group-rendering",
	"group-architecture",
	"group-a11y",
}

// sectionGroupChildren maps each group to its leaf section IDs.
var sectionGroupChildren = map[string][]string{
	"group-basics":       {"typography", "buttons", "links"},
	"group-input":        {"form-controls", "range-progress", "selection", "validation", "pickers", "numeric-spinner"},
	"group-layout":       {"layout", "flex-grid-css", "split-view", "custom-layout", "table-layout"},
	"group-data":         {"virtual-list", "tree", "dataset-slice", "dataset-paged", "dataset-stream", "datatable", "cards", "tabs", "accordion", "badges-chips"},
	"group-navigation":   {"menus", "shortcuts", "toolbar"},
	"group-overlays":     {"overlays", "dialogs"},
	"group-images":       {"images", "shader-effects"},
	"group-animation":    {"spring-anim", "cubic-bezier", "motion-spec", "animation-id", "anim-group-seq"},
	"group-theming":      {"scoped-themes", "gradients", "effects", "blur"},
	"group-i18n":         {"rtl-layout", "locale", "ime-compose"},
	"group-platform":     {"platform-info", "window-controls", "clipboard", "gpu-backend", "multi-window"},
	"group-rendering":    {"canvas-paints", "surfaces", "svg-rendering"},
	"group-architecture": {"commands", "sub-models"},
	"group-a11y":         {"a11y-tree", "a11y-focus-trap", "a11y-bridge"},
}

func sectionLabel(id string) string {
	switch id {
	// Group labels
	case "group-basics":
		return "Basics"
	case "group-input":
		return "Input & Forms"
	case "group-layout":
		return "Layout"
	case "group-data":
		return "Data Display"
	case "group-navigation":
		return "Navigation"
	case "group-overlays":
		return "Overlays & Dialogs"
	case "group-images":
		return "Images & Media"
	case "group-animation":
		return "Animation"
	case "group-theming":
		return "Theming & Style"
	case "group-i18n":
		return "Internationalization"
	case "group-platform":
		return "Platform & System"
	case "group-rendering":
		return "Rendering"
	case "group-architecture":
		return "Architecture"
	// Leaf labels
	case "typography":
		return "Typography"
	case "buttons":
		return "Buttons & Icons"
	case "links":
		return "Links"
	case "form-controls":
		return "Form Controls"
	case "range-progress":
		return "Range & Progress"
	case "selection":
		return "Selection"
	case "validation":
		return "Validation & Hints"
	case "pickers":
		return "Pickers"
	case "numeric-spinner":
		return "Numeric & Spinner"
	case "layout":
		return "Layout"
	case "split-view":
		return "SplitView"
	case "virtual-list":
		return "VirtualList"
	case "tree":
		return "Tree"
	case "dataset-slice":
		return "SliceDataset"
	case "dataset-paged":
		return "PagedDataset"
	case "dataset-stream":
		return "StreamDataset"
	case "datatable":
		return "DataTable"
	case "cards":
		return "Cards"
	case "tabs":
		return "Tabs"
	case "accordion":
		return "Accordion"
	case "badges-chips":
		return "Badges & Chips"
	case "menus":
		return "Menus"
	case "shortcuts":
		return "Shortcuts"
	case "toolbar":
		return "Toolbar"
	case "overlays":
		return "Overlays"
	case "canvas-paints":
		return "Canvas & Paints"
	case "scoped-themes":
		return "Scoped Themes"
	case "commands":
		return "Commands"
	case "sub-models":
		return "Sub-Models"
	case "spring-anim":
		return "Spring Animation"
	case "cubic-bezier":
		return "Cubic Bezier"
	case "motion-spec":
		return "Motion Spec"
	case "animation-id":
		return "Animation ID"
	case "anim-group-seq":
		return "AnimGroup & Seq"
	case "flex-grid-css":
		return "Flex & Grid (CSS)"
	case "custom-layout":
		return "Custom Layout"
	case "table-layout":
		return "Table Layout (CSS)"
	case "rtl-layout":
		return "RTL Layout"
	case "locale":
		return "Locale / i18n"
	case "ime-compose":
		return "IME Compose"
	case "platform-info":
		return "Platform Info"
	case "window-controls":
		return "Window Controls"
	case "clipboard":
		return "Clipboard"
	case "gpu-backend":
		return "GPU Backend"
	case "surfaces":
		return "Surfaces"
	case "svg-rendering":
		return "SVG Rendering"
	case "dialogs":
		return "Dialogs"
	case "gradients":
		return "Gradients"
	case "blur":
		return "Blur"
	case "multi-window":
		return "Multi-Window"
	case "effects":
		return "Effects"
	// New image sections
	case "images":
		return "Images"
	case "shader-effects":
		return "Shader Effects"
	// Accessibility sections
	case "group-a11y":
		return "Accessibility"
	case "a11y-tree":
		return "AccessTree"
	case "a11y-focus-trap":
		return "FocusTrap"
	case "a11y-bridge":
		return "Platform Bridge"
	default:
		return id
	}
}

func sectionChildren(id string) []string {
	return sectionGroupChildren[id]
}

// ── Model ────────────────────────────────────────────────────────

type Model struct {
	Dark           bool
	Count          int
	CheckA         bool
	CheckB         bool
	RadioChoice    string
	ToggleOn       bool
	SliderVal      float32
	Progress       float32
	SelectVal      string
	SelectState    *form.SelectState
	TextValue      string
	Scroll         *ui.ScrollState
	AnimTime       float64
	NavTree        *ui.TreeState
	ActiveSection  string
	ToggleAnim     *form.ToggleState
	VListScroll    *ui.ScrollState
	DemoTree       *ui.TreeState
	TabIndex       int
	AccordionState *nav.AccordionState
	ChipASelected  bool
	ChipBSelected  bool
	ChipCSelected  bool
	ChipDismissed  bool
	LastMenuAction string
	MenuBarState   *menu.MenuBarState
	// SplitView
	NavSplitRatio      float32
	SplitHorizontal    float32
	SplitVertical      float32
	SplitNested1       float32
	SplitNested2       float32
	SplitThreeColLeft  float32
	SplitThreeColRight float32
	// Phase 1 features
	ShortcutLog   string
	OverlayOpen   bool
	HandlerLog    string
	KineticScroll *ui.KineticScroll
	// Commands
	AsyncResult  string
	AsyncLoading bool
	// Sub-Models
	SubCounter int
	// Phase 2 — Animation & Layout
	SpringVal      anim.SpringAnim[float32]
	SpringPreset   string
	AnimIDResult   string
	FadeOpacity    anim.Anim[float32]
	FadeActive     bool
	GroupSeqStatus string
	GroupA         anim.Anim[float32]
	GroupB         anim.Anim[float32]
	SeqA           anim.Anim[float32]
	SeqB           anim.Anim[float32]
	SeqRunning     bool
	BezierAnim     anim.Anim[float32]
	BezierPreset   string
	MotionAnim     anim.Anim[float32]
	MotionPreset   string
	LayoutGap      float32
	// Phase 4b — i18n & Layout
	CurrentLocale  string
	IMEComposeText string
	// Phase 5 — Platform Extension
	ClipboardText string
	IsFullscreen  bool
	// Phase 6 — Surfaces
	Pyramid *PyramidSurface
	// Phase 7 — Dialogs
	ShowMsgDialog     bool
	ShowConfirmDialog bool
	ShowInputDialog   bool
	InputDialogValue  string
	DialogResult      string
	DialogMsgKind     platform.DialogKind
	// Phase F — Multi-Window
	SecondWindowOpen bool
	// Images
	ImageStore   *luximage.Store
	ImgChecker1  draw.ImageID // blue/white checkerboard
	ImgChecker2  draw.ImageID // orange/teal checkerboard (wide, for scale mode demos)
	ImgChecker3  draw.ImageID // pink/green checkerboard (for opacity demo)
	ImageOpacity float32
	// SVG Rendering demo
	SvgStarID     draw.ImageID // rasterized SVG star
	SvgCirclesID  draw.ImageID // rasterized SVG circles
	// Accessibility demos
	A11yTreeText     string
	A11yTrapOpen     bool
	A11yTrapResult   string
	A11yTrapCheckA   bool
	A11yTrapCheckB   bool
	A11yTrapText     string
	// Validation demo
	ValEmail       string
	ValPassword    string
	ValConfirm     string
	ValRole        string
	ValRoleState   *form.SelectState
	ValResults     validation.FormResult
	ValPwRevealed  bool
	// TextArea
	TextAreaValue  string
	TextAreaScroll *ui.ScrollState
	// Toolbar demo
	ToolbarDoc       richtext.AttributedString
	ToolbarDocScroll *ui.ScrollState
	// Pickers & Numeric
	DateVal      time.Time
	DateState    *form.DatePickerState
	ColorVal     draw.Color
	ColorState   *form.ColorPickerState
	TimeHour     int
	TimeMinute   int
	TimeState    *form.TimePickerState
	NumericVal   float64
	// FilePicker
	FilePickerVal      string
	FilePickerState    *form.FilePickerState
	FilePickerDirVal   string
	FilePickerDirState *form.FilePickerState
	// DynamicDataset demos (RFC-002 §6)
	PagedContacts *data.PagedDataset[int]
	PagedScroll   *ui.ScrollState
	StreamLog     *data.StreamDataset[int]
	StreamScroll  *ui.ScrollState
	StreamCounter int
	// DataTable demos
	DTSliceState  *data.DataTableState
	DTPagedState  *data.DataTableState
	DTPaged       *data.PagedDataset[int]
	DTStreamState *data.DataTableState
	DTStream      *data.StreamDataset[int]
	DTStreamCtr   int
	DTSelectedRow int
}

// ── Messages ─────────────────────────────────────────────────────

type IncrMsg struct{}
type DecrMsg struct{}
type ToggleThemeMsg struct{}
type SetCheckAMsg struct{ Value bool }
type SetCheckBMsg struct{ Value bool }
type SetRadioMsg struct{ Choice string }
type SetToggleMsg struct{ Value bool }
type SetSliderMsg struct{ Value float32 }
type SetTextMsg struct{ Value string }
type SetSelectValMsg struct{ Value string }
type SelectSectionMsg struct{ Section string }
type SetTabMsg struct{ Index int }
type ToggleChipAMsg struct{}
type ToggleChipBMsg struct{}
type ToggleChipCMsg struct{}
type DismissChipMsg struct{}
type MenuActionMsg struct{ Action string }
type ToggleOverlayMsg struct{}
type DismissOverlayMsg struct{}
type SetHandlerLogMsg struct{ Text string }
type StartAsyncMsg struct{}
type AsyncDoneMsg struct{ Result string }
type SetNavSplitMsg struct{ Ratio float32 }
type SetSplitHorizontalMsg struct{ Ratio float32 }
type SetSplitVerticalMsg struct{ Ratio float32 }
type SetSplitNested1Msg struct{ Ratio float32 }
type SetSplitNested2Msg struct{ Ratio float32 }
type SetSplitThreeColLeftMsg struct{ Ratio float32 }
type SetSplitThreeColRightMsg struct{ Ratio float32 }
type SubCounterIncrMsg struct{}
type SubCounterDecrMsg struct{}

// Phase 2 messages
type SetSpringPresetMsg struct{ Preset string }
type StartSpringMsg struct{}
type StartFadeMsg struct{}
type StartGroupMsg struct{}
type StartSeqMsg struct{}
type SetBezierPresetMsg struct{ Preset string }
type StartBezierMsg struct{}
type SetMotionPresetMsg struct{ Preset string }
type StartMotionMsg struct{}
type SetLayoutGapMsg struct{ Value float32 }

// Phase 4b messages
type SetLocaleChoiceMsg struct{ Locale string }

// Phase 5 messages
type ToggleFullscreenMsg struct{}
type ResizeWindowMsg struct{ W, H int }
type CopyToClipboardMsg struct{}
type PasteFromClipboardMsg struct{}
type ClipboardResultMsg struct{ Text string }
type SetClipboardTextMsg struct{ Text string }

// Phase 7 — Dialog messages
type ShowMsgDialogMsg struct{ Kind platform.DialogKind }
type ShowConfirmDialogMsg struct{}
type ShowInputDialogMsg struct{}
type DismissDialogMsg struct{}
type DialogConfirmedMsg struct{}
type DialogInputChangedMsg struct{ Value string }
type NativeConfirmMsg struct{}

// Image messages
type SetImageOpacityMsg struct{ Value float32 }

// Validation messages
type SetTextAreaMsg struct{ Value string }


// Toolbar messages
type SetToolbarDocMsg struct{ Doc richtext.AttributedString }

type SetValEmailMsg struct{ Value string }
type SetValPasswordMsg struct{ Value string }
type SetValConfirmMsg struct{ Value string }
type SetValPwRevealedMsg struct{ Value bool }
type SetValRoleMsg struct{ Value string }
type ValidateFormMsg struct{}

// Picker & Numeric messages
type SetDateMsg struct{ Value time.Time }
type SetColorMsg struct{ Value draw.Color }
type SetTimeMsg struct{ Hour, Minute int }
type SetNumericMsg struct{ Value float64 }
type SetFilePickerMsg struct{ Value string }
type SetDirPickerMsg struct{ Value string }

// Accessibility messages
type BuildA11yTreeMsg struct{}
type ToggleA11yTrapMsg struct{}
type DismissA11yTrapMsg struct{}
type A11yTrapConfirmMsg struct{}
type SetA11yTrapCheckAMsg struct{ Value bool }
type SetA11yTrapCheckBMsg struct{ Value bool }
type SetA11yTrapTextMsg struct{ Value string }

// DynamicDataset demo messages
type PagedLoadPageMsg struct{ Page int }
type PagedPageLoadedMsg struct {
	Page  int
	IDs   []int
	Total int
}
type StreamAddItemMsg struct{}
// DataTable messages
type DTPageLoadMsg struct{ Page int }
type DTPageLoadedMsg struct {
	Page  int
	IDs   []int
	Total int
}
type DTStreamAddMsg struct{}
type DTSelectRowMsg struct{ Row int }

// ── Update ───────────────────────────────────────────────────────

// subCounterModel defines a SubModel for the embedded counter demo.
var subCounterModel = app.SubModel[Model, int]{
	Get: func(m Model) int { return m.SubCounter },
	Set: func(m Model, c int) Model { m.SubCounter = c; return m },
	Update: func(c int, msg app.Msg) int {
		switch msg.(type) {
		case SubCounterIncrMsg:
			c++
		case SubCounterDecrMsg:
			c--
		}
		return c
	},
}

func update(m Model, msg app.Msg) (Model, app.Cmd) {
	switch msg := msg.(type) {
	case IncrMsg:
		m.Count++
	case DecrMsg:
		m.Count--
	case app.ModelRestoredMsg:
		app.Send(app.SetDarkModeMsg{Dark: m.Dark})
	case ToggleThemeMsg:
		m.Dark = !m.Dark
		app.Send(app.SetDarkModeMsg{Dark: m.Dark})
	case SetCheckAMsg:
		m.CheckA = msg.Value
	case SetCheckBMsg:
		m.CheckB = msg.Value
	case SetRadioMsg:
		m.RadioChoice = msg.Choice
	case SetToggleMsg:
		m.ToggleOn = msg.Value
	case SetSliderMsg:
		m.SliderVal = msg.Value
	case SetTextMsg:
		m.TextValue = msg.Value
	case SetTextAreaMsg:
		m.TextAreaValue = msg.Value
	case SetToolbarDocMsg:
		m.ToolbarDoc = msg.Doc
	case SelectSectionMsg:
		m.ActiveSection = msg.Section
	case SetTabMsg:
		m.TabIndex = msg.Index
	case ToggleChipAMsg:
		m.ChipASelected = !m.ChipASelected
	case ToggleChipBMsg:
		m.ChipBSelected = !m.ChipBSelected
	case ToggleChipCMsg:
		m.ChipCSelected = !m.ChipCSelected
	case DismissChipMsg:
		m.ChipDismissed = true
	case MenuActionMsg:
		m.LastMenuAction = msg.Action
	case input.ShortcutMsg:
		m.ShortcutLog = fmt.Sprintf("Shortcut: %s", msg.ID)
		switch msg.ID {
		case "incr":
			m.Count++
		case "decr":
			m.Count--
		}
	case ToggleOverlayMsg:
		m.OverlayOpen = !m.OverlayOpen
	case DismissOverlayMsg:
		m.OverlayOpen = false
	case ui.DismissOverlayMsg:
		m.OverlayOpen = false
	case SetHandlerLogMsg:
		m.HandlerLog = msg.Text
	case SetNavSplitMsg:
		m.NavSplitRatio = msg.Ratio
	case SetSplitHorizontalMsg:
		m.SplitHorizontal = msg.Ratio
	case SetSplitVerticalMsg:
		m.SplitVertical = msg.Ratio
	case SetSplitNested1Msg:
		m.SplitNested1 = msg.Ratio
	case SetSplitNested2Msg:
		m.SplitNested2 = msg.Ratio
	case SetSplitThreeColLeftMsg:
		m.SplitThreeColLeft = msg.Ratio
	case SetSplitThreeColRightMsg:
		m.SplitThreeColRight = msg.Ratio

	// Commands section
	case StartAsyncMsg:
		m.AsyncLoading = true
		return m, func() app.Msg {
			time.Sleep(500 * time.Millisecond)
			return AsyncDoneMsg{Result: "Async operation completed!"}
		}
	case AsyncDoneMsg:
		m.AsyncLoading = false
		m.AsyncResult = msg.Result

	// Sub-Models section: delegate to SubModel
	case SubCounterIncrMsg, SubCounterDecrMsg:
		m = app.Delegate(subCounterModel, m, msg)

	// Images
	case SetImageOpacityMsg:
		m.ImageOpacity = msg.Value

	// Accessibility demo messages
	case BuildA11yTreeMsg:
		m.A11yTreeText = buildA11yTreeDemo(m)
	case ToggleA11yTrapMsg:
		m.A11yTrapOpen = true
		m.A11yTrapResult = ""
	case DismissA11yTrapMsg:
		m.A11yTrapOpen = false
		m.A11yTrapResult = "Dialog dismissed (Escape/backdrop)"
	case A11yTrapConfirmMsg:
		m.A11yTrapOpen = false
		m.A11yTrapResult = fmt.Sprintf("Confirmed — CheckA=%v CheckB=%v Text=%q", m.A11yTrapCheckA, m.A11yTrapCheckB, m.A11yTrapText)
	case SetA11yTrapCheckAMsg:
		m.A11yTrapCheckA = msg.Value
	case SetA11yTrapCheckBMsg:
		m.A11yTrapCheckB = msg.Value
	case SetA11yTrapTextMsg:
		m.A11yTrapText = msg.Value

	// DynamicDataset demos
	case data.DatasetLoadRequestMsg:
		// Simulate paged loading for both VirtualList and DataTable demos
		page := msg.PageIndex
		if m.PagedContacts != nil && !m.PagedContacts.IsPageLoading(page) {
			m.PagedContacts.SetLoading(page)
			pg := page
			return m, func() app.Msg {
				pageSize := 20
				ids := make([]int, pageSize)
				for i := range ids {
					ids[i] = pg*pageSize + i
				}
				return PagedPageLoadedMsg{Page: pg, IDs: ids, Total: 100}
			}
		}
		if m.DTPaged != nil && !m.DTPaged.IsPageLoading(page) && !m.DTPaged.IsPageLoaded(page) {
			m.DTPaged.SetLoading(page)
			pg := page
			return m, func() app.Msg {
				pageSize := 20
				ids := make([]int, pageSize)
				for i := range ids {
					ids[i] = pg*pageSize + i
				}
				return DTPageLoadedMsg{Page: pg, IDs: ids, Total: 100}
			}
		}
	case PagedPageLoadedMsg:
		if m.PagedContacts != nil {
			m.PagedContacts.SetPage(msg.Page, msg.IDs, msg.Total)
		}
	case StreamAddItemMsg:
		if m.StreamLog != nil {
			m.StreamCounter++
			m.StreamLog.Append(m.StreamCounter)
		}

	// DataTable messages
	case data.DataTableSortMsg:
		// Update all states that match the sort column.
		m.DTSliceState.SortColumn = msg.Column
		m.DTSliceState.SortDir = msg.Direction
		m.DTPagedState.SortColumn = msg.Column
		m.DTPagedState.SortDir = msg.Direction
	case data.DataTableFilterMsg:
		m.DTSliceState.FilterText = msg.Text
	case data.DataTablePageMsg:
		m.DTPagedState.CurrentPage = msg.Page
		if m.DTPaged != nil {
			page := msg.Page
			if !m.DTPaged.IsPageLoaded(page) && !m.DTPaged.IsPageLoading(page) {
				m.DTPaged.SetLoading(page)
				return m, func() app.Msg {
					pageSize := 20
					ids := make([]int, pageSize)
					for i := range ids {
						ids[i] = page*pageSize + i
					}
					return DTPageLoadedMsg{Page: page, IDs: ids, Total: 100}
				}
			}
		}
	case DTPageLoadedMsg:
		if m.DTPaged != nil {
			m.DTPaged.SetPage(msg.Page, msg.IDs, msg.Total)
		}
	case DTStreamAddMsg:
		if m.DTStream != nil {
			m.DTStreamCtr++
			m.DTStream.Append(m.DTStreamCtr)
		}
	case DTSelectRowMsg:
		m.DTSelectedRow = msg.Row

	// Phase 2: Spring Animation
	case SetSpringPresetMsg:
		m.SpringPreset = msg.Preset
	case StartSpringMsg:
		spec := anim.SpringGentle
		switch m.SpringPreset {
		case "snappy":
			spec = anim.SpringSnappy
		case "bouncy":
			spec = anim.SpringBouncy
		}
		target := float32(1.0)
		if m.SpringVal.Value() > 0.5 {
			target = 0.0
		}
		m.SpringVal.SetTargetWithSpec(target, spec)

	// Phase 2: Animation ID
	case StartFadeMsg:
		m.FadeActive = true
		m.FadeOpacity.SetTargetWithID(0.0, 500*time.Millisecond, anim.OutCubic, "fade-demo")
	case anim.AnimationEnded:
		m.AnimIDResult = fmt.Sprintf("AnimationEnded: %s", msg.ID)
		if msg.ID == "fade-demo" {
			m.FadeActive = false
			m.FadeOpacity.SetImmediate(1.0)
		}

	// Phase 2: AnimGroup & Seq
	case StartGroupMsg:
		m.GroupA.SetTarget(1.0, 300*time.Millisecond, anim.OutCubic)
		m.GroupB.SetTarget(1.0, 500*time.Millisecond, anim.OutCubic)
		m.GroupSeqStatus = "Group running..."
	case StartSeqMsg:
		m.SeqA.SetTarget(1.0, 300*time.Millisecond, anim.OutCubic)
		m.SeqB.SetImmediate(0.0)
		m.SeqRunning = true
		m.GroupSeqStatus = "Seq step 1..."

	// Phase 2: Cubic Bezier
	case SetBezierPresetMsg:
		m.BezierPreset = msg.Preset
	case StartBezierMsg:
		var easing anim.EasingFunc
		switch m.BezierPreset {
		case "ease":
			easing = anim.CubicBezier(0.25, 0.1, 0.25, 1.0)
		case "ease-in":
			easing = anim.CubicBezier(0.42, 0, 1.0, 1.0)
		case "ease-out":
			easing = anim.CubicBezier(0, 0, 0.58, 1.0)
		case "ease-in-out":
			easing = anim.CubicBezier(0.42, 0, 0.58, 1.0)
		default:
			easing = anim.CubicBezier(0.25, 0.1, 0.25, 1.0)
		}
		target := float32(1.0)
		if m.BezierAnim.Value() > 0.5 {
			target = 0.0
		}
		m.BezierAnim.SetTarget(target, 600*time.Millisecond, easing)

	// Phase 2: Motion Spec
	case SetMotionPresetMsg:
		m.MotionPreset = msg.Preset
	case StartMotionMsg:
		tokens := theme.Default.Tokens()
		de := tokens.Motion.Standard
		switch m.MotionPreset {
		case "emphasized":
			de = tokens.Motion.Emphasized
		case "quick":
			de = tokens.Motion.Quick
		}
		target := float32(1.0)
		if m.MotionAnim.Value() > 0.5 {
			target = 0.0
		}
		m.MotionAnim.SetTarget(target, de.Duration, de.Easing)

	// Phase 2: Custom Layout gap
	case SetLayoutGapMsg:
		m.LayoutGap = msg.Value

	// Phase 4b: Locale
	case SetLocaleChoiceMsg:
		m.CurrentLocale = msg.Locale
		app.Send(app.SetLocaleMsg{Locale: msg.Locale})

	// Phase 5: Platform Extension
	case ToggleFullscreenMsg:
		m.IsFullscreen = !m.IsFullscreen
		app.Send(app.SetFullscreenMsg{Fullscreen: m.IsFullscreen})
	case ResizeWindowMsg:
		app.Send(app.SetSizeMsg{Width: msg.W, Height: msg.H})
	case CopyToClipboardMsg:
		return m, func() app.Msg {
			_ = app.SetClipboard(m.ClipboardText)
			return nil
		}
	case PasteFromClipboardMsg:
		return m, func() app.Msg {
			text, _ := app.GetClipboard()
			return ClipboardResultMsg{Text: text}
		}
	case ClipboardResultMsg:
		m.ClipboardText = msg.Text
	case SetClipboardTextMsg:
		m.ClipboardText = msg.Text

	// Phase 7 — Dialogs
	case ShowMsgDialogMsg:
		m.ShowMsgDialog = true
		m.DialogMsgKind = msg.Kind
	case ShowConfirmDialogMsg:
		m.ShowConfirmDialog = true
	case ShowInputDialogMsg:
		m.ShowInputDialog = true
		m.InputDialogValue = ""
	case DismissDialogMsg:
		m.ShowMsgDialog = false
		m.ShowConfirmDialog = false
		m.ShowInputDialog = false
	case DialogConfirmedMsg:
		m.ShowConfirmDialog = false
		m.ShowInputDialog = false
		m.DialogResult = fmt.Sprintf("Confirmed (input: %q)", m.InputDialogValue)
	case DialogInputChangedMsg:
		m.InputDialogValue = msg.Value
	case NativeConfirmMsg:
		return m, dialog.ShowConfirm("Native Confirm", "Do you want to proceed?")
	case dialog.ConfirmResultMsg:
		m.DialogResult = fmt.Sprintf("Native: confirmed=%v", msg.Confirmed)
	case dialog.ShowFallbackConfirmMsg:
		m.ShowConfirmDialog = true
		m.DialogResult = "(native unavailable, using fallback)"

	// Phase F — Multi-Window
	case app.WindowOpenedMsg:
		m.SecondWindowOpen = true
	case app.WindowClosedMsg:
		m.SecondWindowOpen = false

	case app.TickMsg:
		dt := msg.DeltaTime.Seconds()
		m.AnimTime += dt
		m.Progress = float32(math.Mod(m.AnimTime*0.15, 1.0))
		m.ToggleAnim.Tick(msg.DeltaTime)
		m.NavTree.Tick(msg.DeltaTime)
		if m.Pyramid != nil {
			m.Pyramid.Tick(msg.DeltaTime)
		}
		m.DemoTree.Tick(msg.DeltaTime)
		if m.KineticScroll != nil {
			m.KineticScroll.Tick(msg.DeltaTime)
		}
		// Phase 2 animations
		m.SpringVal.Tick(msg.DeltaTime)
		m.FadeOpacity.Tick(msg.DeltaTime)
		m.BezierAnim.Tick(msg.DeltaTime)
		m.MotionAnim.Tick(msg.DeltaTime)
		m.GroupA.Tick(msg.DeltaTime)
		m.GroupB.Tick(msg.DeltaTime)
		if m.GroupA.IsDone() && m.GroupB.IsDone() && m.GroupSeqStatus == "Group running..." {
			m.GroupSeqStatus = "Group done!"
			m.GroupA.SetImmediate(0)
			m.GroupB.SetImmediate(0)
		}
		if m.SeqRunning {
			m.SeqA.Tick(msg.DeltaTime)
			if m.SeqA.IsDone() && m.SeqB.Value() == 0 && m.GroupSeqStatus == "Seq step 1..." {
				m.SeqB.SetTarget(1.0, 300*time.Millisecond, anim.OutCubic)
				m.GroupSeqStatus = "Seq step 2..."
			}
			m.SeqB.Tick(msg.DeltaTime)
			if m.SeqB.IsDone() && m.GroupSeqStatus == "Seq step 2..." {
				m.GroupSeqStatus = "Seq done!"
				m.SeqRunning = false
				m.SeqA.SetImmediate(0)
				m.SeqB.SetImmediate(0)
			}
		}

	// Validation messages
	case SetValEmailMsg:
		m.ValEmail = msg.Value
	case SetValPasswordMsg:
		m.ValPassword = msg.Value
	case SetValConfirmMsg:
		m.ValConfirm = msg.Value
	case SetValPwRevealedMsg:
		m.ValPwRevealed = msg.Value
	case SetValRoleMsg:
		m.ValRole = msg.Value
	case SetSelectValMsg:
		m.SelectVal = msg.Value
	case SetDateMsg:
		m.DateVal = msg.Value
	case SetColorMsg:
		m.ColorVal = msg.Value
	case SetTimeMsg:
		m.TimeHour = msg.Hour
		m.TimeMinute = msg.Minute
	case SetNumericMsg:
		m.NumericVal = msg.Value
	case SetFilePickerMsg:
		m.FilePickerVal = msg.Value
	case SetDirPickerMsg:
		m.FilePickerDirVal = msg.Value
	case ValidateFormMsg:
		schema := validation.Schema{
			"email":    validation.Rules(validation.Required, validation.Email),
			"password": validation.Rules(validation.Required, validation.MinLen(8)),
			"confirm":  validation.RulesCross([]validation.Validator{validation.Required}, validation.EqualField("password")),
			"role":     validation.Rules(validation.Required),
		}
		m.ValResults = schema.ValidateMap(map[string]string{
			"email":    m.ValEmail,
			"password": m.ValPassword,
			"confirm":  m.ValConfirm,
			"role":     m.ValRole,
		})
	}
	return m, nil
}

// ── View ─────────────────────────────────────────────────────────

func view(m Model) ui.Element {
	themeLabel := "Light"
	if !m.Dark {
		themeLabel = "Dark"
	}

	// Left panel: Tree navigation (MaxHeight 0 = fill available space)
	navTree := data.NewTree(ui.TreeConfig{
		RootIDs:  sectionIDs,
		Children: sectionChildren,
		BuildNode: func(id string, depth int, _, selected bool) ui.Element {
			label := sectionLabel(id)
			if depth == 0 {
				// Group nodes rendered bold
				return display.TextStyled(label, draw.TextStyle{
					Size:   13,
					Weight: draw.FontWeightSemiBold,
				})
			}
			return display.Text(label)
		},
		NodeHeight: 28,
		MaxHeight:  0,
		State:      m.NavTree,
		OnSelect:   func(id string) { app.Send(SelectSectionMsg{id}) },
	})

	// Right panel: active section content (maxHeight 0 = fill available space)
	content := nav.NewScrollView(sectionContent(m), 0, m.Scroll)

	return layout.Pad(ui.UniformInsets(16), layout.NewFlex(
		[]ui.Element{
			// SplitView: nav on the left, content on the right — Expanded fills remaining height
			layout.Expand(nav.NewSplitView(
				navTree,
				content,
				m.NavSplitRatio,
				func(r float32) { app.Send(SetNavSplitMsg{r}) },
			)),
			// Footer
			display.Spacer(12),
			layout.Row(
				button.Text(themeLabel, func() { app.Send(ToggleThemeMsg{}) }),
			),
		},
		layout.WithDirection(layout.FlexColumn),
	))
}

func sectionContent(m Model) ui.Element {
	switch m.ActiveSection {
	case "typography":
		return typographySection()
	case "buttons":
		return buttonsSection(m)
	case "links":
		return linksSection()
	case "form-controls":
		return formControlsSection(m)
	case "range-progress":
		return rangeProgressSection(m)
	case "selection":
		return selectionSection(m)
	case "validation":
		return validationSection(m)
	case "pickers":
		return pickersSection(m)
	case "numeric-spinner":
		return numericSpinnerSection(m)
	case "layout":
		return layoutSection()
	case "flex-grid-css":
		return flexGridCSSSection()
	case "split-view":
		return splitViewSection(m)
	case "virtual-list":
		return virtualListSection(m)
	case "tree":
		return treeSection(m)
	case "dataset-slice":
		return datasetSliceSection(m)
	case "dataset-paged":
		return datasetPagedSection(m)
	case "dataset-stream":
		return datasetStreamSection(m)
	case "datatable":
		return dataTableSection(m)
	case "cards":
		return cardsSection()
	case "tabs":
		return tabsSection(m)
	case "accordion":
		return accordionSection(m)
	case "badges-chips":
		return badgesChipsSection(m)
	case "menus":
		return menusSection(m)
	case "shortcuts":
		return shortcutsSection(m)
	case "toolbar":
		return toolbarSection(m)
	case "overlays":
		return overlaysSection(m)
	case "canvas-paints":
		return canvasPaintsSection()
	case "scoped-themes":
		return scopedThemesSection()
	case "commands":
		return commandsSection(m)
	case "sub-models":
		return subModelsSection(m)
	// Phase 2
	case "spring-anim":
		return springAnimSection(m)
	case "cubic-bezier":
		return cubicBezierSection(m)
	case "motion-spec":
		return motionSpecSection(m)
	case "animation-id":
		return animationIDSection(m)
	case "anim-group-seq":
		return animGroupSeqSection(m)
	case "custom-layout":
		return customLayoutSection(m)
	case "table-layout":
		return tableLayoutSection()
	// Phase 4b
	case "rtl-layout":
		return rtlLayoutSection()
	case "locale":
		return localeSection(m)
	case "ime-compose":
		return imeComposeSection(m)
	// Phase 5
	case "platform-info":
		return platformInfoSection()
	case "window-controls":
		return windowControlsSection(m)
	case "clipboard":
		return clipboardSection(m)
	case "gpu-backend":
		return gpuBackendSection()
	// Phase 6
	case "surfaces":
		return surfacesSection(m.Pyramid)
	case "svg-rendering":
		return svgRenderingSection(m)
	// Phase 7
	case "dialogs":
		return dialogsSection(m)
	// Phase E
	case "gradients":
		return gradientsSection()
	// Phase F
	case "blur":
		return blurSection()
	case "multi-window":
		return multiWindowSection(m)
	// Phase G
	case "effects":
		return effectsSection()
	// Images & Media
	case "images":
		return imagesSection(m)
	case "shader-effects":
		return shaderEffectsSection()
	// Accessibility
	case "a11y-tree":
		return a11yTreeSection(m)
	case "a11y-focus-trap":
		return a11yFocusTrapSection(m)
	case "a11y-bridge":
		return a11yBridgeSection()
	default:
		// Group nodes show a hint to expand
		if children := sectionGroupChildren[m.ActiveSection]; len(children) > 0 {
			items := make([]ui.Element, 0, len(children)+2)
			items = append(items, sectionHeader(sectionLabel(m.ActiveSection)))
			items = append(items, display.Text("Expand this group in the tree to see:"))
			items = append(items, display.Spacer(8))
			for _, child := range children {
				items = append(items, display.Text("  "+sectionLabel(child)))
			}
			return layout.Column(items...)
		}
		return layout.Column(
			display.Spacer(24),
			display.Text("Select a section from the tree on the left."),
		)
	}
}

// ── Section Views ────────────────────────────────────────────────

func sectionHeader(title string) ui.Element {
	return layout.Column(
		display.Spacer(8),
		display.TextStyled(title, draw.TextStyle{
			Size:   16,
			Weight: draw.FontWeightSemiBold,
		}),
		display.Spacer(4),
	)
}

func typographySection() ui.Element {
	return layout.Column(
		sectionHeader("Typography"),
		display.TextStyled("Heading 1 (H1)", theme.Default.Tokens().Typography.H1),
		display.TextStyled("Heading 2 (H2)", theme.Default.Tokens().Typography.H2),
		display.TextStyled("Heading 3 (H3)", theme.Default.Tokens().Typography.H3),
		display.Text("Body text — the quick brown fox jumps over the lazy dog."),
		display.TextStyled("Body Small — metadata and captions", theme.Default.Tokens().Typography.BodySmall),
	)
}

func buttonsSection(m Model) ui.Element {
	noop := func() {}
	return layout.Column(
		sectionHeader("Buttons & Icons"),

		// Counter
		display.Text(fmt.Sprintf("Counter: %d", m.Count)),
		layout.Row(
			button.Text("-", func() { app.Send(DecrMsg{}) }),
			button.Text("+", func() { app.Send(IncrMsg{}) }),
		),

		// Filled Buttons
		display.Spacer(8),
		display.Text("Filled (default):"),
		layout.Row(
			button.Text("Action", noop),
			button.Text("Save", noop),
			button.New(layout.Row(display.Icon(icons.Download), display.Text("Download")), noop),
		),

		// Outlined Buttons
		display.Spacer(8),
		display.Text("Outlined:"),
		layout.Row(
			button.OutlinedText("Cancel", noop),
			button.OutlinedText("Details", noop),
			button.VariantOf(ui.ButtonOutlined, layout.Row(display.Icon(icons.Share), display.Text("Share")), noop),
		),

		// Text (Ghost) Buttons
		display.Spacer(8),
		display.Text("Text (chromeless):"),
		layout.Row(
			button.GhostText("Learn more", noop),
			button.GhostText("Skip", noop),
			button.VariantOf(ui.ButtonGhost, layout.Row(display.Icon(icons.ArrowRight), display.Text("Next")), noop),
		),

		// Tonal Buttons
		display.Spacer(8),
		display.Text("Tonal:"),
		layout.Row(
			button.TonalText("Draft", noop),
			button.TonalText("Archive", noop),
			button.VariantOf(ui.ButtonTonal, layout.Row(display.Icon(icons.Copy), display.Text("Duplicate")), noop),
		),

		// Icon Buttons
		display.Spacer(8),
		display.Text("Icon Buttons:"),
		layout.Row(
			button.IconButton(icons.Heart, noop),
			button.IconButton(icons.Star, noop),
			button.IconButton(icons.Trash, noop),
			button.IconButtonVariant(ui.ButtonOutlined, icons.Pencil, noop),
			button.IconButtonVariant(ui.ButtonOutlined, icons.Share, noop),
			button.IconButtonVariant(ui.ButtonGhost, icons.DotsThreeVertical, noop),
			button.IconButtonVariant(ui.ButtonTonal, icons.Play, noop),
		),

		// Split Button
		display.Spacer(8),
		display.Text("Split Button:"),
		layout.Row(
			button.SplitButton("Merge", noop, noop, []button.SplitItem{
				{Label: "Merge commit", OnClick: noop},
				{Label: "Squash and merge", OnClick: noop},
				{Label: "Rebase and merge", OnClick: noop},
			}),
		),

		// Segmented Buttons
		display.Spacer(8),
		display.Text("Segmented Buttons:"),
		button.SegmentedButtons([]button.SegmentedItem{
			{Label: "Day", OnClick: noop},
			{Label: "Week", OnClick: noop},
			{Label: "Month", OnClick: noop},
			{Label: "Year", OnClick: noop},
		}, 1),
		display.Spacer(4),
		button.SegmentedButtons([]button.SegmentedItem{
			{Icon: icons.SortAscending, Label: "Sort", OnClick: noop},
			{Icon: icons.FunnelSimple, Label: "Filter", OnClick: noop},
			{Icon: icons.MagnifyingGlass, Label: "Search", OnClick: noop},
		}, 0),
		display.Spacer(4),
		button.SegmentedButtons([]button.SegmentedItem{
			{Icon: icons.Play, OnClick: noop},
			{Icon: icons.Pause, OnClick: noop},
		}, 0),

		// Icons
		display.Spacer(8),
		display.Text("Icons (Phosphor):"),
		layout.Row(
			display.Icon(icons.Star),
			display.Icon(icons.ArrowRight),
			display.Icon(icons.Heart),
			display.Icon(icons.Gear),
			display.Icon(icons.Eye),
			display.Icon(icons.Sun),
			display.Icon(icons.Moon),
		),
		layout.Row(
			display.Icon(icons.Download),
			display.Icon(icons.Upload),
			display.Icon(icons.Share),
			display.Icon(icons.Copy),
			display.Icon(icons.Link),
			display.Icon(icons.Play),
			display.Icon(icons.Pause),
		),
	)
}

func linksSection() ui.Element {
	noop := func() {}
	return layout.Column(
		sectionHeader("Links"),

		// Basic text links
		display.Text("Text Links:"),
		layout.Row(
			link.Text("Click me", noop),
			link.Text("Learn more", noop),
			link.Text("View details", noop),
		),

		// Link with URL (for accessibility)
		display.Spacer(8),
		display.Text("Links with URL:"),
		layout.Row(
			link.WithURL("Documentation", "https://example.com/docs", noop),
			link.WithURL("GitHub", "https://github.com", noop),
		),

		// Disabled links
		display.Spacer(8),
		display.Text("Disabled:"),
		layout.Row(
			link.TextDisabled("Unavailable"),
			link.TextDisabled("Coming soon"),
		),

		// Link with custom content (icon + text)
		display.Spacer(8),
		display.Text("Custom Content:"),
		layout.Row(
			link.New(layout.Row(display.Icon(icons.ArrowRight), display.Text("Next page")), noop),
			link.New(layout.Row(display.Icon(icons.Download), display.Text("Download")), noop),
		),

		// Inline links in RichText
		display.Spacer(8),
		display.Text("Inline in RichText:"),
		display.RichTextContent(
			display.Span{Text: "Please read the "},
			display.InlineElement(link.Text("terms of service", noop)),
			display.Span{Text: " and "},
			display.InlineElement(link.Text("privacy policy", noop)),
			display.Span{Text: " before continuing."},
		),

		// Multiple inline links in flowing text
		display.Spacer(8),
		display.Text("Links in flowing text:"),
		display.RichTextContent(
			display.Span{Text: "Lux supports "},
			display.InlineElement(link.Text("inline links", noop)),
			display.Span{Text: " that sit on the text baseline. You can place "},
			display.InlineElement(link.Text("multiple links", noop)),
			display.Span{Text: " within a paragraph and they will "},
			display.InlineElement(link.Text("wrap naturally", noop)),
			display.Span{Text: " with the surrounding text flow."},
		),

		// Link with URL in rich text
		display.Spacer(8),
		display.Text("Links with URL in RichText:"),
		display.RichTextContent(
			display.Span{Text: "Visit "},
			display.InlineElement(link.WithURL("our website", "https://example.com", noop)),
			display.Span{Text: " for more information, or check the "},
			display.InlineElement(link.WithURL("API reference", "https://example.com/api", noop)),
			display.Span{Text: "."},
		),
	)
}

func formControlsSection(m Model) ui.Element {
	return layout.Column(
		sectionHeader("Form Controls"),
		form.NewTextField(m.TextValue, "Enter text...",
			form.WithOnChange(func(v string) { app.Send(SetTextMsg{v}) }),
			form.WithFocus(app.Focus()),
		),
		display.Spacer(12),
		display.Text("TextArea:"),
		form.NewTextArea(m.TextAreaValue, "Multiline (press Enter for newline)...",
			form.TextAreaOnChange(func(v string) { app.Send(SetTextAreaMsg{v}) }),
			form.TextAreaFocus(app.Focus()),
			form.TextAreaRows(4),
			form.TextAreaScroll(m.TextAreaScroll),
		),
		display.Spacer(8),
		form.NewCheckbox("Enable notifications", m.CheckA, func(v bool) { app.Send(SetCheckAMsg{v}) }),
		form.NewCheckbox("Auto-save", m.CheckB, func(v bool) { app.Send(SetCheckBMsg{v}) }),
		display.Spacer(8),
		form.NewRadio("Alpha", m.RadioChoice == "alpha", func() { app.Send(SetRadioMsg{"alpha"}) }),
		form.NewRadio("Beta", m.RadioChoice == "beta", func() { app.Send(SetRadioMsg{"beta"}) }),
		form.NewRadio("Gamma", m.RadioChoice == "gamma", func() { app.Send(SetRadioMsg{"gamma"}) }),
		display.Spacer(8),
		layout.Row(
			display.Text("Dark mode:"),
			form.NewToggle(m.ToggleOn, func(v bool) { app.Send(SetToggleMsg{v}) }, m.ToggleAnim),
		),
	)
}

func rangeProgressSection(m Model) ui.Element {
	return layout.Column(
		sectionHeader("Range & Progress"),
		display.Text(fmt.Sprintf("Slider value: %.0f%%", m.SliderVal*100)),
		form.NewSlider(m.SliderVal, func(v float32) { app.Send(SetSliderMsg{v}) }),
		display.Spacer(8),
		display.Text(fmt.Sprintf("Progress: %.0f%%", m.Progress*100)),
		form.NewProgressBar(m.Progress),
		display.Spacer(4),
		display.Text("Indeterminate:"),
		form.ProgressBarIndeterminate(float32(math.Mod(m.AnimTime*0.8, 1.0))),
	)
}

func selectionSection(m Model) ui.Element {
	return layout.Column(
		sectionHeader("Selection"),
		form.NewSelect(m.SelectVal, []string{"Option 1", "Option 2", "Option 3"},
			form.WithSelectState(m.SelectState),
			form.WithOnSelect(func(v string) { app.Send(SetSelectValMsg{v}) }),
		),
	)
}

func pickersSection(m Model) ui.Element {
	return layout.Column(
		sectionHeader("Pickers"),

		display.Text("DatePicker:"),
		form.NewDatePicker(m.DateVal,
			form.WithDatePickerState(m.DateState),
			form.WithOnDateChange(func(v time.Time) { app.Send(SetDateMsg{v}) }),
		),
		display.Spacer(12),

		display.Text("ColorPicker:"),
		form.NewColorPicker(m.ColorVal,
			form.WithColorPickerState(m.ColorState),
			form.WithOnColorChange(func(v draw.Color) { app.Send(SetColorMsg{v}) }),
		),
		display.Spacer(12),

		display.Text("TimePicker:"),
		form.NewTimePicker(m.TimeHour, m.TimeMinute,
			form.WithTimePickerState(m.TimeState),
			form.WithOnTimeChange(func(h, min int) { app.Send(SetTimeMsg{h, min}) }),
		),
		display.Spacer(12),

		display.Text(fmt.Sprintf("FilePicker (Open): %s", m.FilePickerVal)),
		form.NewFilePicker(m.FilePickerVal,
			form.WithFilePickerState(m.FilePickerState),
			form.WithOnFileSelect(func(v string) { app.Send(SetFilePickerMsg{v}) }),
			form.WithFileFilters(
				form.FileFilter{Label: "Go Files", Extensions: []string{".go"}},
				form.FileFilter{Label: "All Files", Extensions: []string{"*"}},
			),
		),
		display.Spacer(12),

		display.Text(fmt.Sprintf("FilePicker (Directory): %s", m.FilePickerDirVal)),
		form.NewFilePicker(m.FilePickerDirVal,
			form.WithFilePickerState(m.FilePickerDirState),
			form.WithOnFileSelect(func(v string) { app.Send(SetDirPickerMsg{v}) }),
			form.WithFilePickerMode(form.FilePickerDirectory),
		),
	)
}

func numericSpinnerSection(m Model) ui.Element {
	return layout.Column(
		sectionHeader("Numeric & Spinner"),

		display.Text(fmt.Sprintf("NumericInput (value: %.0f):", m.NumericVal)),
		form.NewNumericInput(m.NumericVal,
			form.WithNumericRange(0, 100),
			form.WithNumericStep(1),
			form.WithUnit("px"),
			form.WithOnNumericChange(func(v float64) { app.Send(SetNumericMsg{v}) }),
		),
		display.Spacer(12),

		display.Text("Spinner:"),
		form.NewSpinner(float32(math.Mod(m.AnimTime*1.2, 1.0))),
		display.Spacer(8),

		display.Text("Large Spinner:"),
		form.NewSpinner(float32(math.Mod(m.AnimTime*0.8, 1.0)),
			form.WithSpinnerSize(48),
		),
	)
}

func validationSection(m Model) ui.Element {
	return layout.Column(
		sectionHeader("Validation & Hints"),

		// Email with hint (info icon by default in Lux theme).
		form.NewFormField(
			form.NewTextField(m.ValEmail, "you@example.com",
				form.WithOnChange(func(v string) { app.Send(SetValEmailMsg{v}) }),
				form.WithFocus(app.Focus()),
			),
			form.WithFormLabel("Email"),
			form.WithFormHint("We'll never share your email."),
			form.WithFormValidation(m.ValResults.Get("email")),
		),
		display.Spacer(12),

		// Password with hint rendered as label (overridden per field).
		form.NewFormField(
			form.NewPasswordField(m.ValPassword, "Min. 8 characters",
				form.WithPasswordOnChange(func(v string) { app.Send(SetValPasswordMsg{v}) }),
				form.WithPasswordFocus(app.Focus()),
				form.WithPasswordReveal(m.ValPwRevealed, func(b bool) { app.Send(SetValPwRevealedMsg{b}) }),
			),
			form.WithFormLabel("Password"),
			form.WithFormHint("Use a strong password with letters, numbers, and symbols."),
			form.WithFormHintMode(theme.HintModeLabel),
			form.WithFormValidation(m.ValResults.Get("password")),
		),
		display.Spacer(12),

		// Confirm password (cross-field validation).
		form.NewFormField(
			form.NewPasswordField(m.ValConfirm, "Repeat password",
				form.WithPasswordOnChange(func(v string) { app.Send(SetValConfirmMsg{v}) }),
				form.WithPasswordFocus(app.Focus()),
			),
			form.WithFormLabel("Confirm Password"),
			form.WithFormValidation(m.ValResults.Get("confirm")),
		),
		display.Spacer(12),

		// Role select (required validation).
		form.NewFormField(
			form.NewSelect(m.ValRole, []string{"Admin", "Editor", "Viewer"},
				form.WithSelectState(m.ValRoleState),
				form.WithOnSelect(func(v string) { app.Send(SetValRoleMsg{v}) }),
			),
			form.WithFormLabel("Role"),
			form.WithFormHint("Select your role."),
			form.WithFormValidation(m.ValResults.Get("role")),
		),
		display.Spacer(12),

		button.Text("Validate", func() { app.Send(ValidateFormMsg{}) }),
	)
}

func layoutSection() ui.Element {
	return layout.Column(
		sectionHeader("Layout"),

		// Row
		display.Text("Row:"),
		layout.Row(display.Text("A"), display.Text("B"), display.Text("C")),
		display.Spacer(8),

		// Stack
		display.Text("Stack (overlapping):"),
		layout.NewStack(display.Text("Bottom"), display.Text("Top")),
		display.Spacer(8),

		// Flex with Justify
		display.Text("Flex (JustifySpaceBetween):"),
		layout.NewFlex([]ui.Element{
			display.Text("Left"),
			display.Text("Center"),
			display.Text("Right"),
		}, layout.WithJustify(layout.JustifySpaceBetween)),
		display.Spacer(8),

		// Flex with Expanded
		display.Text("Flex with Expanded:"),
		layout.NewFlex([]ui.Element{
			button.Text("Fixed", nil),
			layout.Expand(display.Text("← takes remaining space →")),
			button.Text("Fixed", nil),
		}),
		display.Spacer(8),

		// Grid
		display.Text("Grid (3 columns):"),
		layout.NewGrid(3, []ui.Element{
			display.Text("Cell 1"), display.Text("Cell 2"), display.Text("Cell 3"),
			display.Text("Cell 4"), display.Text("Cell 5"), display.Text("Cell 6"),
		}, layout.WithColGap(12), layout.WithRowGap(8)),
		display.Spacer(8),

		// Padding
		display.Text("Padding (16dp):"),
		layout.Pad(ui.UniformInsets(16), display.Text("Padded content")),
		display.Spacer(8),

		// SizedBox
		display.Text("SizedBox (100x50):"),
		layout.Sized(100, 50, display.Text("Sized")),
	)
}

// flexGridCSSSection demonstrates CSS-spec-compliant Flex and Grid features.
func flexGridCSSSection() ui.Element {
	label := func(text string) ui.Element {
		return display.TextStyled(text, draw.TextStyle{Size: 13, Weight: draw.FontWeightMedium})
	}
	chip := func(text string) ui.Element {
		return layout.Pad(ui.InlineInsets(12, 6),
			display.TextStyled(text, draw.TextStyle{Size: 12}))
	}
	box := func(text string, w, h float32) ui.Element {
		return layout.Sized(w, h, display.Text(text))
	}

	return layout.Column(
		sectionHeader("Flex & Grid (CSS Compliance)"),

		// ── 1. Flex Wrap ──────────────────────────────────────
		label("flex-wrap: wrap (8 items in constrained row)"),
		display.Spacer(4),
		layout.Sized(0, 80, layout.NewFlex([]ui.Element{
			chip("Alpha"), chip("Beta"), chip("Gamma"), chip("Delta"),
			chip("Epsilon"), chip("Zeta"), chip("Eta"), chip("Theta"),
		},
			layout.WithWrap(layout.FlexWrapOn),
			layout.WithGap(8),
		)),
		display.Spacer(12),

		// ── 2. Flex Direction Reverse ─────────────────────────
		label("flex-direction: row-reverse"),
		display.Spacer(4),
		layout.NewFlex([]ui.Element{
			display.Text("1"), display.Text("2"), display.Text("3"),
		},
			layout.WithDirection(layout.FlexRowReverse),
			layout.WithGap(12),
		),
		display.Spacer(12),

		// ── 3. Flex Grow & Shrink ─────────────────────────────
		label("flex-grow: 1 / 2 / 1 (proportional fill)"),
		display.Spacer(4),
		layout.NewFlex([]ui.Element{
			layout.Expand(box("1×", 0, 30), 1),
			layout.Expand(box("2×", 0, 30), 2),
			layout.Expand(box("1×", 0, 30), 1),
		}, layout.WithGap(8)),
		display.Spacer(12),

		// ── 4. Flex AlignSelf ─────────────────────────────────
		label("align-self per child (start / center / end / stretch)"),
		display.Spacer(4),
		layout.Sized(0, 60, layout.NewFlex([]ui.Element{
			layout.FlexChild(box("S", 60, 20), layout.WithAlignSelf(layout.AlignSelfStart)),
			layout.FlexChild(box("C", 60, 20), layout.WithAlignSelf(layout.AlignSelfCenter)),
			layout.FlexChild(box("E", 60, 20), layout.WithAlignSelf(layout.AlignSelfEnd)),
			layout.FlexChild(box("X", 60, 20), layout.WithAlignSelf(layout.AlignSelfStretch)),
		}, layout.WithGap(8))),
		display.Spacer(12),

		// ── 5. Flex Order ─────────────────────────────────────
		label("order: visual reordering (source: C A B → visual: A B C)"),
		display.Spacer(4),
		layout.NewFlex([]ui.Element{
			layout.FlexChild(display.Text("C (order=3)"), layout.WithOrder(3)),
			layout.FlexChild(display.Text("A (order=1)"), layout.WithOrder(1)),
			layout.FlexChild(display.Text("B (order=2)"), layout.WithOrder(2)),
		}, layout.WithGap(12)),
		display.Spacer(12),

		// ── 6. Flex AlignContent ──────────────────────────────
		label("align-content: space-between (wrapped lines)"),
		display.Spacer(4),
		layout.Sized(0, 100, layout.NewFlex([]ui.Element{
			box("A", 80, 25), box("B", 80, 25), box("C", 80, 25),
			box("D", 80, 25), box("E", 80, 25), box("F", 80, 25),
		},
			layout.WithWrap(layout.FlexWrapOn),
			layout.WithAlignContent(layout.AlignContentSpaceBetween),
			layout.WithGap(8),
		)),
		display.Spacer(20),

		// ── 7. Grid Template Columns (fr) ─────────────────────
		label("grid-template-columns: 1fr 2fr 1fr"),
		display.Spacer(4),
		layout.NewTemplateGrid(
			[]layout.TrackSize{layout.Fr(1), layout.Fr(2), layout.Fr(1)},
			[]ui.Element{
				display.Text("1fr"), display.Text("2fr"), display.Text("1fr"),
				display.Text("1fr"), display.Text("2fr"), display.Text("1fr"),
			},
			layout.WithColGap(8), layout.WithRowGap(4),
		),
		display.Spacer(12),

		// ── 8. Grid Template Mixed ────────────────────────────
		label("grid-template-columns: 80px 1fr auto"),
		display.Spacer(4),
		layout.NewTemplateGrid(
			[]layout.TrackSize{layout.Px(80), layout.Fr(1), layout.AutoTrack()},
			[]ui.Element{
				display.Text("80px"), display.Text("fills"), display.Text("auto"),
			},
			layout.WithColGap(12),
		),
		display.Spacer(12),

		// ── 9. Grid Item Placement & Span ─────────────────────
		label("grid: explicit placement (item spans 2 columns)"),
		display.Spacer(4),
		layout.NewTemplateGrid(
			[]layout.TrackSize{layout.Fr(1), layout.Fr(1), layout.Fr(1)},
			[]ui.Element{
				layout.PlaceGridItem(display.Text("Span 2 cols"), layout.ColSpan(1, 2)),
				display.Text("auto"), display.Text("auto"), display.Text("auto"),
			},
			layout.WithColGap(8), layout.WithRowGap(4),
		),
		display.Spacer(12),

		// ── 10. Grid Auto-Flow Column ─────────────────────────
		label("grid-auto-flow: column (items fill columns first)"),
		display.Spacer(4),
		layout.NewGrid(3, []ui.Element{
			display.Text("1"), display.Text("2"), display.Text("3"),
			display.Text("4"), display.Text("5"),
		},
			layout.WithAutoFlow(layout.GridFlowColumn),
			layout.WithColGap(12), layout.WithRowGap(4),
		),
	)
}

func splitViewSection(m Model) ui.Element {
	return layout.Column(
		sectionHeader("SplitView"),
		display.Text("Draggable split panes — resize by dragging the divider."),

		// 1. Horizontal (side-by-side)
		display.Spacer(12),
		display.Text("Horizontal split (default):"),
		display.Spacer(4),
		layout.Sized(0, 120, nav.NewSplitView(
			layout.Pad(ui.UniformInsets(8), layout.Column(
				display.TextStyled("Left Pane", draw.TextStyle{Size: 14, Weight: draw.FontWeightSemiBold}),
				display.Spacer(4),
				display.Text("This panel resizes"),
				display.Text("when you drag the divider."),
			)),
			layout.Pad(ui.UniformInsets(8), layout.Column(
				display.TextStyled("Right Pane", draw.TextStyle{Size: 14, Weight: draw.FontWeightSemiBold}),
				display.Spacer(4),
				display.Text("Content is clipped"),
				display.Text("at the pane boundary."),
			)),
			m.SplitHorizontal,
			func(r float32) { app.Send(SetSplitHorizontalMsg{r}) },
		)),

		// 2. Vertical (stacked)
		display.Spacer(12),
		display.Text("Vertical split (stacked, WithSplitAxis):"),
		display.Spacer(4),
		layout.Sized(0, 160, nav.NewSplitView(
			layout.Pad(ui.UniformInsets(8), layout.Column(
				display.TextStyled("Top", draw.TextStyle{Size: 14, Weight: draw.FontWeightSemiBold}),
				display.Spacer(4),
				display.Text("Vertical divider splits top/bottom."),
			)),
			layout.Pad(ui.UniformInsets(8), layout.Column(
				display.TextStyled("Bottom", draw.TextStyle{Size: 14, Weight: draw.FontWeightSemiBold}),
				display.Spacer(4),
				display.Text("Drag the horizontal bar to resize."),
			)),
			m.SplitVertical,
			func(r float32) { app.Send(SetSplitVerticalMsg{r}) },
			nav.WithSplitAxis(ui.AxisColumn),
		)),

		// 3. Nested splits (editor-like layout)
		display.Spacer(12),
		display.Text("Nested splits (IDE-style layout):"),
		display.Spacer(4),
		layout.Sized(0, 180, nav.NewSplitView(
			layout.Pad(ui.UniformInsets(8), layout.Column(
				layout.Row(display.Icon(icons.Folder), display.Text(" Explorer")),
				display.Spacer(4),
				display.Text("  src/"),
				display.Text("  docs/"),
				display.Text("  tests/"),
			)),
			nav.NewSplitView(
				layout.Pad(ui.UniformInsets(8), layout.Column(
					layout.Row(display.Icon(icons.FileText), display.Text(" main.go")),
					display.Spacer(4),
					display.Text("  func main() {"),
					display.Text("    // ..."),
					display.Text("  }"),
				)),
				layout.Pad(ui.UniformInsets(8), layout.Column(
					layout.Row(display.Icon(icons.Play), display.Text(" Terminal")),
					display.Spacer(4),
					display.Text("  $ go run ."),
				)),
				m.SplitNested2,
				func(r float32) { app.Send(SetSplitNested2Msg{r}) },
				nav.WithSplitAxis(ui.AxisColumn),
			),
			m.SplitNested1,
			func(r float32) { app.Send(SetSplitNested1Msg{r}) },
		)),

		// 4. Three-column layout
		display.Spacer(12),
		display.Text("Three columns (nested horizontal):"),
		display.Spacer(4),
		layout.Sized(0, 120, nav.NewSplitView(
			layout.Pad(ui.UniformInsets(8), layout.Column(
				display.TextStyled("Nav", draw.TextStyle{Size: 14, Weight: draw.FontWeightSemiBold}),
				display.Text("Home"),
				display.Text("Settings"),
			)),
			nav.NewSplitView(
				layout.Pad(ui.UniformInsets(8), layout.Column(
					display.TextStyled("Content", draw.TextStyle{Size: 14, Weight: draw.FontWeightSemiBold}),
					display.Text("Main area"),
				)),
				layout.Pad(ui.UniformInsets(8), layout.Column(
					display.TextStyled("Details", draw.TextStyle{Size: 14, Weight: draw.FontWeightSemiBold}),
					display.Text("Inspector"),
				)),
				m.SplitThreeColRight,
				func(r float32) { app.Send(SetSplitThreeColRightMsg{r}) },
			),
			m.SplitThreeColLeft,
			func(r float32) { app.Send(SetSplitThreeColLeftMsg{r}) },
		)),

		// 5. Fixed (non-draggable)
		display.Spacer(12),
		display.Text("Fixed split (no drag — nil onResize):"),
		display.Spacer(4),
		layout.Sized(0, 80, nav.NewSplitView(
			layout.Pad(ui.UniformInsets(8), display.Text("Fixed left (30%)")),
			layout.Pad(ui.UniformInsets(8), display.Text("Fixed right (70%)")),
			0.3,
			nil,
		)),
	)
}

func virtualListSection(m Model) ui.Element {
	return layout.Column(
		sectionHeader("VirtualList"),
		display.Text("1000 items — only visible items are rendered:"),
		display.Spacer(8),
		data.NewVirtualList(ui.VirtualListConfig{
			ItemCount:  1000,
			ItemHeight: 24,
			BuildItem: func(i int) ui.Element {
				return display.Text(fmt.Sprintf("  Item %d — virtualized row", i))
			},
			MaxHeight: 200,
			State:     m.VListScroll,
		}),
	)
}

// Demo tree data for the Tree section.
var demoTreeRoots = []string{"Documents", "Pictures", "Music"}

func demoTreeChildren(id string) []string {
	switch id {
	case "Documents":
		return []string{"Work", "Personal", "Archive"}
	case "Work":
		return []string{"Reports", "Presentations"}
	case "Pictures":
		return []string{"Vacation", "Family"}
	case "Music":
		return []string{"Rock", "Jazz", "Classical"}
	default:
		return nil
	}
}

func treeSection(m Model) ui.Element {
	return layout.Column(
		sectionHeader("Tree"),
		display.Text("Hierarchical tree with expand/collapse:"),
		display.Spacer(8),
		data.NewTree(ui.TreeConfig{
			RootIDs:  demoTreeRoots,
			Children: demoTreeChildren,
			BuildNode: func(id string, _ int, expanded, _ bool) ui.Element {
				kids := demoTreeChildren(id)
				if len(kids) > 0 {
					icon := icons.Folder
					if expanded {
						icon = icons.FolderOpen
					}
					return layout.Row(display.Icon(icon), display.Text(id))
				}
				return layout.Row(display.Icon(icons.FileText), display.Text(id))
			},
			NodeHeight: 24,
			MaxHeight:  200,
			State:      m.DemoTree,
		}),
	)
}

// ── DynamicDataset Demos (RFC-002 §6) ────────────────────────────

func datasetSliceSection(m Model) ui.Element {
	// SliceDataset — static, known length. Drop-in replacement for old ItemCount.
	items := make([]int, 200)
	for i := range items {
		items[i] = i
	}
	ds := data.NewSliceDataset(items)

	return layout.Column(
		sectionHeader("SliceDataset"),
		display.Text("Drop-in replacement for the old ItemCount API."),
		display.Text("All 200 items are immediately available (loaded=true)."),
		display.Spacer(8),
		data.VirtualList{
			Dataset:    ds,
			ItemHeight: 24,
			MaxHeight:  200,
			State:      m.VListScroll,
			BuildItemDS: func(i int, loaded bool) ui.Element {
				id, _ := ds.Get(i)
				status := "loaded"
				if !loaded {
					status = "loading..."
				}
				return display.Text(fmt.Sprintf("  SliceDataset[%d] = %d (%s)", i, id, status))
			},
		},
	)
}

func datasetPagedSection(m Model) ui.Element {
	// PagedDataset — paginated, loads on demand.
	totalInfo := "unknown"
	if m.PagedContacts.Len() >= 0 {
		totalInfo = fmt.Sprintf("%d", m.PagedContacts.Len())
	}

	return layout.Column(
		sectionHeader("PagedDataset"),
		display.Text("Paginated dataset — pages load on demand when scrolled into view."),
		display.Text(fmt.Sprintf("Total: %s  |  PageSize: %d", totalInfo, m.PagedContacts.PageSize)),
		display.Spacer(8),
		data.VirtualList{
			Dataset:    m.PagedContacts,
			ItemHeight: 28,
			MaxHeight:  250,
			State:      m.PagedScroll,
			BuildItemDS: func(i int, loaded bool) ui.Element {
				if !loaded {
					return display.Text(fmt.Sprintf("  [%d] Loading...", i))
				}
				id, _ := m.PagedContacts.Get(i)
				return display.Text(fmt.Sprintf("  Contact #%d  (page %d)", id, m.PagedContacts.PageForIndex(i)))
			},
		},
		display.Spacer(8),
		display.Text("Scroll down to trigger automatic page loading via DatasetLoadRequestMsg."),
	)
}

func datasetStreamSection(m Model) ui.Element {
	// StreamDataset — append-only, unknown total length.
	return layout.Column(
		sectionHeader("StreamDataset"),
		display.Text("Append-only stream (chat, log). Total length is always unknown (Len()=-1)."),
		display.Text(fmt.Sprintf("Items: %d  |  Len(): %d", m.StreamLog.Count(), m.StreamLog.Len())),
		display.Spacer(8),
		button.Text("Add Log Entry", func() { app.Send(StreamAddItemMsg{}) }),
		display.Spacer(8),
		data.VirtualList{
			Dataset:    m.StreamLog,
			ItemHeight: 24,
			MaxHeight:  200,
			State:      m.StreamScroll,
			BuildItemDS: func(i int, loaded bool) ui.Element {
				if !loaded {
					return display.Text("  ...")
				}
				id, _ := m.StreamLog.Get(i)
				return display.Text(fmt.Sprintf("  [stream] Log entry #%d", id))
			},
		},
	)
}

// ── DataTable Section ────────────────────────────────────────────

func dataTableSection(m Model) ui.Element {
	// --- Demo data for static table ---
	type person struct {
		Name string
		Age  int
		City string
	}
	people := []person{
		{"Alice", 32, "Berlin"}, {"Bob", 28, "Munich"}, {"Charlie", 45, "Hamburg"},
		{"Diana", 36, "Frankfurt"}, {"Eve", 29, "Cologne"}, {"Frank", 51, "Stuttgart"},
		{"Grace", 27, "Düsseldorf"}, {"Hank", 39, "Leipzig"}, {"Ivy", 33, "Dresden"},
		{"Jack", 44, "Hannover"}, {"Karen", 30, "Nuremberg"}, {"Leo", 55, "Bremen"},
		{"Mia", 26, "Essen"}, {"Nick", 48, "Dortmund"}, {"Olivia", 35, "Bonn"},
		{"Paul", 41, "Mannheim"}, {"Quinn", 31, "Karlsruhe"}, {"Rita", 37, "Augsburg"},
		{"Sam", 43, "Wiesbaden"}, {"Tina", 29, "Freiburg"}, {"Ulrich", 50, "Münster"},
		{"Vera", 34, "Aachen"}, {"Walter", 46, "Kiel"}, {"Xena", 28, "Lübeck"},
		{"Yves", 38, "Rostock"}, {"Zara", 42, "Mainz"}, {"Anton", 25, "Potsdam"},
		{"Berta", 53, "Erfurt"}, {"Claus", 33, "Weimar"}, {"Doris", 40, "Jena"},
	}
	sliceIDs := make([]int, len(people))
	for i := range sliceIDs {
		sliceIDs[i] = i
	}
	sliceDS := data.NewSliceDataset(sliceIDs)

	staticCols := []data.DataTableColumn{
		{
			Key: "name", Header: "Name", Width: layout.Fr(2), Sortable: true,
			Build: func(id int, loaded bool) ui.Element {
				return display.Text(people[id].Name)
			},
			FilterValue: func(id int) string { return people[id].Name },
			SortLess: func(i, j int) bool {
				return people[i].Name < people[j].Name
			},
		},
		{
			Key: "age", Header: "Age", Width: layout.Px(80), Sortable: true,
			Build: func(id int, loaded bool) ui.Element {
				return display.Text(fmt.Sprintf("%d", people[id].Age))
			},
			FilterValue: func(id int) string { return fmt.Sprintf("%d", people[id].Age) },
			SortLess: func(i, j int) bool {
				return people[i].Age < people[j].Age
			},
		},
		{
			Key: "city", Header: "City", Width: layout.Fr(2), Sortable: true,
			Build: func(id int, loaded bool) ui.Element {
				return display.Text(people[id].City)
			},
			FilterValue: func(id int) string { return people[id].City },
			SortLess: func(i, j int) bool {
				return people[i].City < people[j].City
			},
		},
	}

	// --- Paged table columns ---
	pagedCols := []data.DataTableColumn{
		{
			Key: "id", Header: "ID", Width: layout.Px(60),
			Build: func(id int, loaded bool) ui.Element {
				if !loaded {
					return display.Text("…")
				}
				v, _ := m.DTPaged.Get(id)
				return display.Text(fmt.Sprintf("%d", v))
			},
		},
		{
			Key: "contact", Header: "Contact", Width: layout.Fr(1),
			Build: func(id int, loaded bool) ui.Element {
				if !loaded {
					return display.Text("Loading…")
				}
				v, _ := m.DTPaged.Get(id)
				return display.Text(fmt.Sprintf("Contact #%d", v))
			},
		},
		{
			Key: "page", Header: "Page", Width: layout.Px(80),
			Build: func(id int, loaded bool) ui.Element {
				if !loaded {
					return display.Text("—")
				}
				return display.Text(fmt.Sprintf("%d", m.DTPaged.PageForIndex(id)))
			},
		},
	}

	// --- Stream table columns ---
	streamCols := []data.DataTableColumn{
		{
			Key: "idx", Header: "#", Width: layout.Px(50),
			Build: func(id int, loaded bool) ui.Element {
				return display.Text(fmt.Sprintf("%d", id+1))
			},
		},
		{
			Key: "entry", Header: "Log Entry", Width: layout.Fr(1),
			Build: func(id int, loaded bool) ui.Element {
				if !loaded {
					return display.Text("…")
				}
				v, _ := m.DTStream.Get(id)
				return display.Text(fmt.Sprintf("Log message #%d", v))
			},
		},
	}

	return layout.Column(
		sectionHeader("DataTable"),
		display.Text("Data-driven table with sorting, filtering, and auto-detected pagination."),
		display.Spacer(12),

		// 1. Static table with SliceDataset
		display.TextStyled("Static Table (SliceDataset)", theme.Default.Tokens().Typography.H3),
		display.Spacer(4),
		display.Text("30 rows, sortable columns, filterable. Click headers to sort."),
		display.Spacer(4),
		data.NewDataTable(sliceDS, staticCols, m.DTSliceState,
			data.WithDTMaxHeight(280),
			data.WithDTFilterable(true),
			data.WithDTSelectedRow(m.DTSelectedRow, func(idx int) {
				app.Send(DTSelectRowMsg{Row: idx})
			}),
		),
		display.Spacer(4),
		display.Text(fmt.Sprintf("Selected row: %d", m.DTSelectedRow)),
		display.Spacer(16),

		// 2. Paged table with PagedDataset
		display.TextStyled("Paged Table (PagedDataset)", theme.Default.Tokens().Typography.H3),
		display.Spacer(4),
		display.Text("Server-paginated contacts. Pagination toolbar auto-detected."),
		display.Spacer(4),
		data.NewDataTable(m.DTPaged, pagedCols, m.DTPagedState,
			data.WithDTMaxHeight(250),
		),
		display.Spacer(16),

		// 3. Stream table with StreamDataset
		display.TextStyled("Stream Table (StreamDataset)", theme.Default.Tokens().Typography.H3),
		display.Spacer(4),
		display.Text("Append-only log. Toolbar shows item count."),
		display.Spacer(4),
		button.Text("Add Log Entry", func() { app.Send(DTStreamAddMsg{}) }),
		display.Spacer(4),
		data.NewDataTable(m.DTStream, streamCols, m.DTStreamState,
			data.WithDTMaxHeight(200),
		),
	)
}

// ── Tier 3 Section Views ─────────────────────────────────────────

func cardsSection() ui.Element {
	return layout.Column(
		sectionHeader("Cards"),
		display.Text("Card with text content:"),
		display.Spacer(4),
		display.Card(
			display.Text("This content lives inside a Card."),
			display.Text("Cards have elevation and borders."),
		),
		display.Spacer(12),
		display.Text("Nested cards:"),
		display.Spacer(4),
		display.Card(
			display.Text("Outer card"),
			display.Spacer(8),
			display.Card(display.Text("Inner nested card")),
		),
	)
}

func tabsSection(m Model) ui.Element {
	return layout.Column(
		sectionHeader("Tabs"),
		display.Text("Tabs with rich headers (Icon + Text + Badge):"),
		display.Spacer(4),
		nav.New([]nav.TabItem{
			{
				Header:  layout.Row(display.Icon(icons.Star), display.Text("General")),
				Content: display.Text("General settings content goes here."),
			},
			{
				Header:  layout.Row(display.Icon(icons.Gear), display.Text("Advanced"), display.BadgeText("3")),
				Content: layout.Column(display.Text("Advanced settings."), display.Text("With multiple items.")),
			},
			{
				Header:  layout.Row(display.Icon(icons.Eye), display.Text("Preview")),
				Content: display.Card(display.Text("Preview content inside a Card.")),
			},
		}, m.TabIndex, func(idx int) { app.Send(SetTabMsg{idx}) }),
	)
}

func accordionSection(m Model) ui.Element {
	return layout.Column(
		sectionHeader("Accordion"),
		display.Text("Collapsible sections (click to expand/collapse):"),
		display.Spacer(4),
		nav.NewAccordion([]nav.AccordionSection{
			{
				Header:  display.Text("Section 1 — Getting Started"),
				Content: display.Text("Welcome! This section covers the basics."),
			},
			{
				Header:  display.Text("Section 2 — Configuration"),
				Content: layout.Column(display.Text("Configure your settings here."), display.Text("Multiple widgets supported.")),
			},
			{
				Header:  display.Text("Section 3 — Advanced Topics"),
				Content: display.Card(display.Text("Advanced content inside a Card.")),
			},
		}, m.AccordionState),
	)
}

func badgesChipsSection(m Model) ui.Element {
	tokens := theme.Default.Tokens()
	children := []ui.Element{
		sectionHeader("Badges & Chips"),

		display.Text("Badges (colorful pill indicators):"),
		display.Spacer(4),
		layout.Row(
			display.BadgeText("3"),
			display.BadgeColor(display.Text("99+"), tokens.Colors.Status.Error),
			display.BadgeColor(display.Icon(icons.Star), tokens.Colors.Status.Warning),
			display.BadgeColor(display.Text("New"), tokens.Colors.Status.Success),
			display.BadgeColor(layout.Row(display.Icon(icons.Heart), display.Text("Hot")), tokens.Colors.Accent.Secondary),
		),

		display.Spacer(12),
		display.Text("Chips (selectable):"),
		display.Spacer(4),
		layout.Row(
			display.Chip(display.Text("Go"), m.ChipASelected, func() { app.Send(ToggleChipAMsg{}) }),
			display.Chip(display.Text("Rust"), m.ChipBSelected, func() { app.Send(ToggleChipBMsg{}) }),
			display.Chip(display.Text("Python"), m.ChipCSelected, func() { app.Send(ToggleChipCMsg{}) }),
		),
	}

	// Dismissible chip (shown until dismissed)
	if !m.ChipDismissed {
		children = append(children,
			display.Spacer(8),
			display.Text("Dismissible chip (click × to remove):"),
			display.ChipDismissible(
				layout.Row(display.Icon(icons.Star), display.Text("Featured")),
				true,
				func() {},
				func() { app.Send(DismissChipMsg{}) },
			),
		)
	} else {
		children = append(children,
			display.Spacer(8),
			display.Text("Chip dismissed!"),
		)
	}

	children = append(children,
		display.Spacer(12),
		display.Text("Tooltip (hover to show):"),
		display.Spacer(4),
		layout.Row(
			menu.New(
				display.Text("← Hover me for tooltip"),
				display.Text("This is a tooltip with arbitrary content!"),
			),
		),
	)

	return layout.Column(children...)
}

func menusSection(m Model) ui.Element {
	menuAction := func(action string) func() {
		return func() { app.Send(MenuActionMsg{action}) }
	}

	children := []ui.Element{
		sectionHeader("Menus"),

		display.Text("MenuBar (click to open dropdown):"),
		display.Spacer(4),
		menu.NewMenuBar([]menu.MenuItem{
			{Label: display.Text("File"), Items: []menu.MenuItem{
				{Label: display.Text("New"), OnClick: menuAction("File > New")},
				{Label: display.Text("Open"), OnClick: menuAction("File > Open")},
				{Label: display.Text("Save"), OnClick: menuAction("File > Save")},
			}},
			{Label: display.Text("Edit"), Items: []menu.MenuItem{
				{Label: display.Text("Undo"), OnClick: menuAction("Edit > Undo")},
				{Label: display.Text("Redo"), OnClick: menuAction("Edit > Redo")},
				{Label: display.Text("Cut"), OnClick: menuAction("Edit > Cut")},
				{Label: display.Text("Copy"), OnClick: menuAction("Edit > Copy")},
				{Label: display.Text("Paste"), OnClick: menuAction("Edit > Paste")},
			}},
			{Label: display.Text("View"), Items: []menu.MenuItem{
				{Label: display.Text("Zoom In"), OnClick: menuAction("View > Zoom In")},
				{Label: display.Text("Zoom Out"), OnClick: menuAction("View > Zoom Out")},
			}},
			{Label: display.Text("Help"), OnClick: menuAction("Help")},
		}, m.MenuBarState),
	}

	if m.LastMenuAction != "" {
		children = append(children,
			display.Spacer(4),
			display.Text(fmt.Sprintf("Last action: %s", m.LastMenuAction)),
		)
	}

	children = append(children,
		display.Spacer(12),
		display.Text("ContextMenu:"),
		display.Spacer(4),
		menu.NewContextMenu([]menu.MenuItem{
			{Label: display.Text("Cut"), OnClick: menuAction("Cut")},
			{Label: display.Text("Copy"), OnClick: menuAction("Copy")},
			{Label: display.Text("Paste"), OnClick: menuAction("Paste")},
			{Label: display.Text("Delete"), OnClick: menuAction("Delete")},
		}, true, 0, 0),
	)

	return layout.Column(children...)
}

// ── Phase 1 Sections ──────────────────────────────────────────────

func shortcutsSection(m Model) ui.Element {
	children := []ui.Element{
		sectionHeader("Keyboard Shortcuts"),
		display.Text("Registered shortcuts:"),
		display.Spacer(4),
		display.Text("  Ctrl+I → Increment counter"),
		display.Text("  Ctrl+D → Decrement counter"),
		display.Spacer(8),
		display.Text(fmt.Sprintf("Counter value: %d", m.Count)),
	}

	if m.ShortcutLog != "" {
		children = append(children,
			display.Spacer(8),
			display.Text(fmt.Sprintf("Last shortcut: %s", m.ShortcutLog)),
		)
	}

	children = append(children,
		display.Spacer(16),
		sectionHeader("Global Handler Layer"),
		display.Text("A global handler logs all key events before widget dispatch."),
	)
	if m.HandlerLog != "" {
		children = append(children,
			display.Spacer(4),
			display.Text(fmt.Sprintf("Handler log: %s", m.HandlerLog)),
		)
	}

	return layout.Column(children...)
}

func toolbarSection(m Model) ui.Element {
	return layout.Column(
		sectionHeader("Toolbar (RFC-003 §4.1)"),

		// ── Demo 1: Basic toolbar ────────────────────────────
		display.Text("Basic toolbar with action buttons:"),
		display.Spacer(4),
		nav.NewToolbar([]nav.ToolbarItem{
			{Element: display.Text("Cut"), OnClick: func() {}},
			{Element: display.Text("Copy"), OnClick: func() {}},
			{Element: display.Text("Paste"), OnClick: func() {}},
		}),

		display.Spacer(16),

		// ── Demo 2: Toolbar with separators ──────────────────
		display.Text("Toolbar with separators and icon buttons:"),
		display.Spacer(4),
		nav.NewToolbar([]nav.ToolbarItem{
			{Element: display.Text("New"), OnClick: func() {}},
			{Element: display.Text("Open"), OnClick: func() {}},
			{Element: display.Text("Save"), OnClick: func() {}},
			nav.ToolbarSeparator(),
			{Element: display.Text("Undo"), OnClick: func() {}},
			{Element: display.Text("Redo"), OnClick: func() {}},
			nav.ToolbarSeparator(),
			{Element: display.Icon(icons.Gear), OnClick: func() {}},
		}),

		display.Spacer(16),

		// ── Demo 3: Formatting toolbar + RichTextEditor ──────
		display.Text("Integrated RichTextEditor with pluggable toolbar:"),
		display.Spacer(4),
		richtext.NewEditorWithToolbar(m.ToolbarDoc,
			richtext.WithWidgetOnChange(func(as richtext.AttributedString) { app.Send(SetToolbarDocMsg{as}) }),
			richtext.WithWidgetFocus(app.Focus()),
			richtext.WithWidgetRows(6),
			richtext.WithWidgetScroll(m.ToolbarDocScroll),
			richtext.WithWidgetPlaceholder("Select text and click Bold / Italic / Underline..."),
		),

		display.Spacer(16),

		// ── Demo 4: Toolbar with custom gap ──────────────────
		display.Text("Toolbar with larger gap (8px):"),
		display.Spacer(4),
		nav.NewToolbarWithGap([]nav.ToolbarItem{
			{Element: display.Text("Zoom In"), OnClick: func() {}},
			{Element: display.Text("Zoom Out"), OnClick: func() {}},
			{Element: display.Text("100%"), OnClick: func() {}},
		}, 8),
	)
}

func overlaysSection(m Model) ui.Element {
	children := []ui.Element{
		sectionHeader("Overlay System"),
		display.Text("Click the button to toggle a dismissable overlay:"),
		display.Spacer(4),
		button.Text("Toggle Overlay", func() { app.Send(ToggleOverlayMsg{}) }),
	}

	if m.OverlayOpen {
		children = append(children,
			display.Spacer(8),
			display.Text("Overlay is OPEN (click outside or press button to close)"),
			display.Spacer(4),
			// The actual Overlay element rendered above normal flow.
			ui.Overlay{
				ID:          "demo-overlay",
				Anchor:      draw.R(300, 300, 100, 30),
				Placement:   ui.PlacementBelow,
				Dismissable: true,
				OnDismiss:   func() { app.Send(DismissOverlayMsg{}) },
				Content: display.Card(layout.Column(
					display.Text("This is an overlay!"),
					display.Spacer(4),
					display.Text("It renders above normal content."),
					display.Spacer(8),
					button.Text("Close", func() { app.Send(DismissOverlayMsg{}) }),
				)),
			},
		)
	} else {
		children = append(children,
			display.Spacer(8),
			display.Text("Overlay is closed."),
		)
	}

	children = append(children,
		display.Spacer(16),
		sectionHeader("Kinetic Scrolling"),
		display.Text("KineticScroll with friction-decay physics is available."),
		display.Text("Use trackpad for smooth kinetic scrolling or mouse wheel for discrete steps."),
	)

	return layout.Column(children...)
}

// ── Phase 3 Sections ──────────────────────────────────────────

func canvasPaintsSection() ui.Element {
	// Demonstrate the new Phase 3 Canvas API and Paint variants.
	// Since the GPU backend doesn't render these yet, this section
	// serves as an API showcase and compile-time validation.

	// 1. PathBuilder with ArcTo
	arcPath := draw.NewPath().
		MoveTo(draw.Pt(0, 0)).
		ArcTo(30, 30, 0, false, true, draw.Pt(60, 0)).
		LineTo(draw.Pt(60, 40)).
		LineTo(draw.Pt(0, 40)).
		Close().
		Build()
	_ = arcPath // used for FillPath once GPU supports it

	// 2. Gradient paints
	linearPaint := draw.LinearGradientPaint(
		draw.Pt(0, 0), draw.Pt(200, 0),
		draw.GradientStop{Offset: 0, Color: draw.Hex("#3b82f6")},
		draw.GradientStop{Offset: 1, Color: draw.Hex("#6366f1")},
	)
	radialPaint := draw.RadialGradientPaint(
		draw.Pt(50, 50), 50,
		draw.GradientStop{Offset: 0, Color: draw.Hex("#ffffff")},
		draw.GradientStop{Offset: 1, Color: draw.Hex("#09090b")},
	)
	_ = linearPaint
	_ = radialPaint

	// 3. TextLayout
	_ = draw.TextLayout{
		Text:      "Centered text layout",
		Style:     draw.TextStyle{Size: 14, Weight: draw.FontWeightRegular},
		MaxWidth:  300,
		Alignment: draw.TextAlignCenter,
	}

	// 4. LayerOptions
	_ = draw.LayerOptions{
		BlendMode: draw.BlendNormal,
		Opacity:   0.8,
		CacheHint: true,
	}

	return layout.Column(
		sectionHeader("Canvas & Paints (Phase 3)"),

		display.Text("New Canvas API (GPU stubs — API validation):"),
		display.Spacer(4),
		display.Text("  PathBuilder.ArcTo — elliptical arc segments"),
		display.Text("  PushClipRoundRect / PushClipPath — advanced clipping"),
		display.Text("  PushBlur / PopBlur — backdrop blur effects"),
		display.Text("  PushLayer / PopLayer — compositing layers"),
		display.Text("  PushScale — uniform/non-uniform scaling"),
		display.Text("  DrawTextLayout — rich text layout with alignment"),
		display.Text("  DrawImageSlice — 9-slice image rendering"),
		display.Text("  DrawTexture — external texture surfaces"),

		display.Spacer(12),
		display.Text("Paint Variants:"),
		display.Spacer(4),
		display.Text(fmt.Sprintf("  LinearGradientPaint: %d stops", 2)),
		display.Text(fmt.Sprintf("  RadialGradientPaint: radius=%.0f", float64(50))),
		display.Text("  PatternPaint: tiled image fills"),

		display.Spacer(12),
		display.Text("Theme-Lookup-Cache:"),
		display.Spacer(4),
		display.Text("  CachedTheme wraps Theme with lazy resolution"),
		display.Text("  Auto-invalidation on SetThemeMsg / SetDarkModeMsg"),
		display.Text("  Warm-up before first frame in app.Run"),
	)
}

// ── Scoped Themes Section ─────────────────────────────────────────

// Pre-built theme overrides for the scoped-themes demo.
var (
	dangerTheme = theme.Override(theme.Default, theme.OverrideSpec{
		Colors: &theme.ColorScheme{
			Surface: theme.Default.Tokens().Colors.Surface,
			Accent: theme.AccentColors{
				Primary:         draw.Hex("#dc2626"), // Red-600
				PrimaryContrast: draw.Hex("#ffffff"),
				Secondary:       draw.Hex("#f87171"), // Red-400
			},
			Stroke: theme.Default.Tokens().Colors.Stroke,
			Text:   theme.Default.Tokens().Colors.Text,
			Status: theme.Default.Tokens().Colors.Status,
		},
	})

	successTheme = theme.Override(theme.Default, theme.OverrideSpec{
		Colors: &theme.ColorScheme{
			Surface: theme.Default.Tokens().Colors.Surface,
			Accent: theme.AccentColors{
				Primary:         draw.Hex("#16a34a"), // Green-600
				PrimaryContrast: draw.Hex("#ffffff"),
				Secondary:       draw.Hex("#4ade80"), // Green-400
			},
			Stroke: theme.Default.Tokens().Colors.Stroke,
			Text:   theme.Default.Tokens().Colors.Text,
			Status: theme.Default.Tokens().Colors.Status,
		},
	})

	warningTheme = theme.Override(theme.Default, theme.OverrideSpec{
		Colors: &theme.ColorScheme{
			Surface: theme.Default.Tokens().Colors.Surface,
			Accent: theme.AccentColors{
				Primary:         draw.Hex("#d97706"), // Amber-600
				PrimaryContrast: draw.Hex("#ffffff"),
				Secondary:       draw.Hex("#fbbf24"), // Amber-400
			},
			Stroke: theme.Default.Tokens().Colors.Stroke,
			Text:   theme.Default.Tokens().Colors.Text,
			Status: theme.Default.Tokens().Colors.Status,
		},
	})
)

func scopedThemesSection() ui.Element {
	return layout.Column(
		sectionHeader("Scoped Themes"),
		display.Text("ui.Themed() overrides the theme for a subtree."),
		display.Text("Buttons below inherit their accent color from scoped themes:"),
		display.Spacer(12),

		// Default (no override)
		display.Text("Default theme:"),
		display.Spacer(4),
		layout.Row(
			button.Text("Save", nil),
			button.Text("Submit", nil),
		),
		display.Spacer(12),

		// Danger theme
		display.Text("Danger theme (red accent):"),
		display.Spacer(4),
		ui.Themed(dangerTheme,
			layout.Row(
				button.Text("Delete", nil),
				button.Text("Reset All", nil),
			),
		),
		display.Spacer(12),

		// Success theme
		display.Text("Success theme (green accent):"),
		display.Spacer(4),
		ui.Themed(successTheme,
			layout.Row(
				button.Text("Confirm", nil),
				button.Text("Approve", nil),
			),
		),
		display.Spacer(12),

		// Warning theme
		display.Text("Warning theme (amber accent):"),
		display.Spacer(4),
		ui.Themed(warningTheme,
			layout.Row(
				button.Text("Proceed", nil),
				button.Text("Override", nil),
			),
		),
		display.Spacer(12),

		// Mixed: default and themed side by side
		display.Text("Mixed — default and danger in one row:"),
		display.Spacer(4),
		layout.Row(
			button.Text("Normal", nil),
			ui.Themed(dangerTheme,
				button.Text("Danger", nil),
			),
			button.Text("Normal", nil),
		),
	)
}
// ── Commands Section ──────────────────────────────────────────────

func commandsSection(m Model) ui.Element {
	children := []ui.Element{
		sectionHeader("Commands (Async Side Effects)"),
		display.Text("Commands let your update function trigger async work."),
		display.Text("The result is sent back as a message when done."),
		display.Spacer(12),
		button.Text("Run Async", func() { app.Send(StartAsyncMsg{}) }),
	}

	if m.AsyncLoading {
		children = append(children,
			display.Spacer(8),
			display.Text("Loading..."),
		)
	}
	if m.AsyncResult != "" {
		children = append(children,
			display.Spacer(8),
			display.Text(fmt.Sprintf("Result: %s", m.AsyncResult)),
		)
	}

	return layout.Column(children...)
}

// ── Sub-Models Section ───────────────────────────────────────────

func subModelsSection(m Model) ui.Element {
	return layout.Column(
		sectionHeader("Sub-Models (Delegated State)"),
		display.Text("SubModel delegates messages to a child model."),
		display.Text("The counter below is managed by a separate update function:"),
		display.Spacer(12),
		display.Text(fmt.Sprintf("Sub-Counter: %d", m.SubCounter)),
		layout.Row(
			button.Text("-", func() { app.Send(SubCounterDecrMsg{}) }),
			button.Text("+", func() { app.Send(SubCounterIncrMsg{}) }),
		),
	)
}

// ── Phase 2 Section Views ──────────────────────────────────────────

func springAnimSection(m Model) ui.Element {
	presetLabel := "gentle"
	if m.SpringPreset != "" {
		presetLabel = m.SpringPreset
	}

	return layout.Column(
		sectionHeader("Spring Animation (Phase 2)"),
		display.Text("SpringAnim[T] — physics-based spring-damper system."),
		display.Text("No fixed duration — converges asymptotically."),
		display.Spacer(8),

		display.Text("Select preset:"),
		layout.Row(
			form.NewRadio("Gentle", presetLabel == "gentle", func() { app.Send(SetSpringPresetMsg{"gentle"}) }),
			form.NewRadio("Snappy", presetLabel == "snappy", func() { app.Send(SetSpringPresetMsg{"snappy"}) }),
			form.NewRadio("Bouncy", presetLabel == "bouncy", func() { app.Send(SetSpringPresetMsg{"bouncy"}) }),
		),
		display.Spacer(8),

		button.Text("Animate Spring", func() { app.Send(StartSpringMsg{}) }),
		display.Spacer(8),
		display.Text(fmt.Sprintf("Value: %.3f", m.SpringVal.Value())),
		form.NewProgressBar(m.SpringVal.Value()),
		display.Spacer(4),
		display.Text(fmt.Sprintf("Done: %v", m.SpringVal.IsDone())),
	)
}

func cubicBezierSection(m Model) ui.Element {
	preset := m.BezierPreset
	if preset == "" {
		preset = "ease"
	}

	return layout.Column(
		sectionHeader("Cubic Bezier Easing (Phase 2)"),
		display.Text("CubicBezier(x1, y1, x2, y2) — CSS-compatible easing."),
		display.Spacer(8),

		display.Text("Select CSS preset:"),
		layout.Row(
			form.NewRadio("ease", preset == "ease", func() { app.Send(SetBezierPresetMsg{"ease"}) }),
			form.NewRadio("ease-in", preset == "ease-in", func() { app.Send(SetBezierPresetMsg{"ease-in"}) }),
			form.NewRadio("ease-out", preset == "ease-out", func() { app.Send(SetBezierPresetMsg{"ease-out"}) }),
			form.NewRadio("ease-in-out", preset == "ease-in-out", func() { app.Send(SetBezierPresetMsg{"ease-in-out"}) }),
		),
		display.Spacer(8),

		button.Text("Animate", func() { app.Send(StartBezierMsg{}) }),
		display.Spacer(8),
		display.Text(fmt.Sprintf("Value: %.3f (preset: %s)", m.BezierAnim.Value(), preset)),
		form.NewProgressBar(m.BezierAnim.Value()),
	)
}

func motionSpecSection(m Model) ui.Element {
	tokens := theme.Default.Tokens()
	preset := m.MotionPreset
	if preset == "" {
		preset = "standard"
	}

	return layout.Column(
		sectionHeader("Motion Spec (Phase 2)"),
		display.Text("DurationEasing — theme tokens pair duration with easing."),
		display.Spacer(8),
		display.Text(fmt.Sprintf("Standard:   %v + OutCubic", tokens.Motion.Standard.Duration)),
		display.Text(fmt.Sprintf("Emphasized: %v + InOutCubic", tokens.Motion.Emphasized.Duration)),
		display.Text(fmt.Sprintf("Quick:      %v + OutExpo", tokens.Motion.Quick.Duration)),
		display.Spacer(12),

		display.Text("Select preset:"),
		layout.Row(
			form.NewRadio("Standard", preset == "standard", func() { app.Send(SetMotionPresetMsg{"standard"}) }),
			form.NewRadio("Emphasized", preset == "emphasized", func() { app.Send(SetMotionPresetMsg{"emphasized"}) }),
			form.NewRadio("Quick", preset == "quick", func() { app.Send(SetMotionPresetMsg{"quick"}) }),
		),
		display.Spacer(8),

		button.Text("Animate", func() { app.Send(StartMotionMsg{}) }),
		display.Spacer(8),
		display.Text(fmt.Sprintf("Value: %.3f", m.MotionAnim.Value())),
		form.NewProgressBar(m.MotionAnim.Value()),
	)
}

func animationIDSection(m Model) ui.Element {
	children := []ui.Element{
		sectionHeader("Animation ID (Phase 2)"),
		display.Text("SetTargetWithID — fires AnimationEnded{ID} on completion."),
		display.Text("The user update loop receives the message — no callbacks."),
		display.Spacer(8),
		button.Text("Start Fade (500ms)", func() { app.Send(StartFadeMsg{}) }),
	}

	if m.FadeActive {
		children = append(children,
			display.Spacer(8),
			display.Text(fmt.Sprintf("Fading... opacity: %.2f", m.FadeOpacity.Value())),
			form.NewProgressBar(m.FadeOpacity.Value()),
		)
	}

	if m.AnimIDResult != "" {
		children = append(children,
			display.Spacer(8),
			display.Text(fmt.Sprintf("Received: %s", m.AnimIDResult)),
		)
	}

	return layout.Column(children...)
}

func animGroupSeqSection(m Model) ui.Element {
	return layout.Column(
		sectionHeader("AnimGroup & AnimSeq (Phase 2)"),
		display.Text("AnimGroup — parallel animations. AnimSeq — sequential."),
		display.Spacer(8),

		display.Text("Parallel (AnimGroup):"),
		layout.Row(
			button.Text("Run Group", func() { app.Send(StartGroupMsg{}) }),
		),
		display.Spacer(4),
		display.Text(fmt.Sprintf("  A: %.2f  B: %.2f", m.GroupA.Value(), m.GroupB.Value())),
		display.Spacer(8),

		display.Text("Sequential (AnimSeq):"),
		layout.Row(
			button.Text("Run Seq", func() { app.Send(StartSeqMsg{}) }),
		),
		display.Spacer(4),
		display.Text(fmt.Sprintf("  A: %.2f  B: %.2f", m.SeqA.Value(), m.SeqB.Value())),
		display.Spacer(8),

		display.Text(fmt.Sprintf("Status: %s", m.GroupSeqStatus)),
	)
}

// stairLayout is a demo custom layout that arranges children in a stair pattern.
type stairLayout struct {
	Gap float32
}

func (s stairLayout) LayoutChildren(ctx ui.LayoutCtx, children []ui.Element) ui.Size {
	x, y := float32(0), float32(0)
	maxW, maxH := float32(0), float32(0)

	for _, child := range children {
		size := ctx.Measure(child, ui.LooseConstraints(ctx.Constraints.MaxWidth, ctx.Constraints.MaxHeight))
		ctx.Place(child, draw.Pt(x, y))
		endX := x + size.Width
		endY := y + size.Height
		if endX > maxW {
			maxW = endX
		}
		if endY > maxH {
			maxH = endY
		}
		x += s.Gap
		y += s.Gap
	}

	return ui.Size{Width: maxW, Height: maxH}
}

func customLayoutSection(m Model) ui.Element {
	gap := m.LayoutGap
	if gap == 0 {
		gap = 30
	}

	return layout.Column(
		sectionHeader("Custom Layout (Phase 2)"),
		display.Text("Layout interface — user-defined layout algorithms."),
		display.Text("LayoutCtx provides Measure/Place callbacks."),
		display.Spacer(8),

		display.Text(fmt.Sprintf("Stair gap: %.0f dp", gap)),
		form.NewSlider(gap/100, func(v float32) { app.Send(SetLayoutGapMsg{v * 100}) }),
		display.Spacer(8),

		display.Text("Stair layout demo:"),
		display.Spacer(4),
		ui.CustomLayout(stairLayout{Gap: gap},
			button.Text("Step 1", nil),
			button.Text("Step 2", nil),
			button.Text("Step 3", nil),
			button.Text("Step 4", nil),
		),

		display.Spacer(16),
		display.Text("Layout Cache:"),
		display.Spacer(4),
		display.Text("  LayoutCache stores constraints + size + childRects"),
		display.Text("  Invalidated when props or constraints change"),
		display.Text("  O(dirty subtree) re-layout, not O(n)"),
	)
}

// ── Table Layout Section ──────────────────────────────────────────

func tableLayoutSection() ui.Element {
	return layout.Column(
		sectionHeader("Table Layout (CSS)"),
		display.Text("HTML-spec-conformant table layout (CSS 2.1 §17)."),
		display.Text("Supports auto/fixed layout, colspan/rowspan, border-spacing, and captions."),
		display.Spacer(12),

		// 1. Basic Table
		display.Text("1. Basic Table:"),
		display.Spacer(4),
		layout.SimpleTable(
			[]ui.Element{display.Text("Name"), display.Text("Age"), display.Text("City")},
			[][]ui.Element{
				{display.Text("Alice"), display.Text("30"), display.Text("Berlin")},
				{display.Text("Bob"), display.Text("25"), display.Text("Munich")},
				{display.Text("Carol"), display.Text("35"), display.Text("Hamburg")},
			},
			layout.WithBorderSpacing(8, 4),
		),
		display.Spacer(16),

		// 2. Table with Header & Footer
		display.Text("2. Table with Header / Footer:"),
		display.Spacer(4),
		layout.NewTable([]ui.Element{
			layout.THead(
				layout.TR(layout.TH(display.Text("Product")), layout.TH(display.Text("Price")), layout.TH(display.Text("Qty"))),
			),
			layout.TBody(
				layout.TR(layout.TD(display.Text("Widget A")), layout.TD(display.Text("9.99")), layout.TD(display.Text("5"))),
				layout.TR(layout.TD(display.Text("Widget B")), layout.TD(display.Text("14.50")), layout.TD(display.Text("3"))),
				layout.TR(layout.TD(display.Text("Widget C")), layout.TD(display.Text("7.25")), layout.TD(display.Text("12"))),
			),
			layout.TFoot(
				layout.TR(layout.TD(display.Text("Total")), layout.TD(display.Text("31.74")), layout.TD(display.Text("20"))),
			),
		}, layout.WithBorderSpacing(12, 4)),
		display.Spacer(16),

		// 3. Column Spanning
		display.Text("3. Column Spanning (colspan):"),
		display.Spacer(4),
		layout.NewTable([]ui.Element{
			layout.TR(
				layout.TD(display.Text("Spans 2 columns"), layout.WithColSpan(2)),
				layout.TD(display.Text("Col 3")),
			),
			layout.TR(
				layout.TD(display.Text("Col 1")),
				layout.TD(display.Text("Col 2")),
				layout.TD(display.Text("Col 3")),
			),
			layout.TR(
				layout.TD(display.Text("Col 1")),
				layout.TD(display.Text("Spans 2 columns"), layout.WithColSpan(2)),
			),
		}, layout.WithBorderSpacing(8, 4)),
		display.Spacer(16),

		// 4. Row Spanning
		display.Text("4. Row Spanning (rowspan):"),
		display.Spacer(4),
		layout.NewTable([]ui.Element{
			layout.TR(
				layout.TD(display.Text("Spans 3 rows"), layout.WithRowSpan(3)),
				layout.TD(display.Text("Row 1, Col 2")),
				layout.TD(display.Text("Row 1, Col 3")),
			),
			layout.TR(
				layout.TD(display.Text("Row 2, Col 2")),
				layout.TD(display.Text("Row 2, Col 3")),
			),
			layout.TR(
				layout.TD(display.Text("Row 3, Col 2")),
				layout.TD(display.Text("Row 3, Col 3")),
			),
		}, layout.WithBorderSpacing(8, 4)),
		display.Spacer(16),

		// 5. Fixed vs Auto Layout
		display.Text("5. Fixed Layout (table-layout: fixed):"),
		display.Spacer(4),
		layout.NewTable([]ui.Element{
			layout.NewTableColGroup(
				layout.Col(layout.Px(120)),
				layout.Col(layout.Px(200)),
				layout.Col(layout.Px(100)),
			),
			layout.TR(layout.TH(display.Text("Fixed 120")), layout.TH(display.Text("Fixed 200")), layout.TH(display.Text("Fixed 100"))),
			layout.TR(layout.TD(display.Text("A")), layout.TD(display.Text("B")), layout.TD(display.Text("C"))),
		}, layout.WithTableLayout(layout.TableLayoutFixed), layout.WithBorderSpacing(8, 4)),
		display.Spacer(16),

		// 6. Border Spacing
		display.Text("6. Border Spacing (16px H, 8px V):"),
		display.Spacer(4),
		layout.NewTable([]ui.Element{
			layout.TR(layout.TD(display.Text("A")), layout.TD(display.Text("B")), layout.TD(display.Text("C"))),
			layout.TR(layout.TD(display.Text("D")), layout.TD(display.Text("E")), layout.TD(display.Text("F"))),
			layout.TR(layout.TD(display.Text("G")), layout.TD(display.Text("H")), layout.TD(display.Text("I"))),
		}, layout.WithBorderSpacing(16, 8)),
		display.Spacer(16),

		// 7. Collapsed Borders
		display.Text("7. Collapsed Borders (border-collapse: collapse):"),
		display.Spacer(4),
		layout.NewTable([]ui.Element{
			layout.TR(layout.TD(display.Text("A")), layout.TD(display.Text("B")), layout.TD(display.Text("C"))),
			layout.TR(layout.TD(display.Text("D")), layout.TD(display.Text("E")), layout.TD(display.Text("F"))),
		}, layout.WithBorderCollapse(layout.BorderCollapsed)),
		display.Spacer(16),

		// 8. Vertical Alignment
		display.Text("8. Vertical Alignment in Cells:"),
		display.Spacer(4),
		layout.NewTable([]ui.Element{
			layout.TR(
				layout.TD(display.Text("Top"), layout.WithVAlign(layout.VAlignTop)),
				layout.TD(display.Text("Middle"), layout.WithVAlign(layout.VAlignMiddle)),
				layout.TD(display.Text("Bottom"), layout.WithVAlign(layout.VAlignBottom)),
				layout.NewTableCell(layout.Sized(80, 60, nil)),
			),
		}, layout.WithBorderSpacing(8, 4)),
		display.Spacer(16),

		// 9. Caption
		display.Text("9. Table with Caption (top):"),
		display.Spacer(4),
		layout.NewTable([]ui.Element{
			layout.NewTableCaption(display.Text("Table 1: Monthly Sales")),
			layout.TR(layout.TH(display.Text("Month")), layout.TH(display.Text("Revenue"))),
			layout.TR(layout.TD(display.Text("January")), layout.TD(display.Text("$12,000"))),
			layout.TR(layout.TD(display.Text("February")), layout.TD(display.Text("$15,500"))),
		}, layout.WithBorderSpacing(8, 4)),
		display.Spacer(8),
		display.Text("Caption (bottom):"),
		display.Spacer(4),
		layout.NewTable([]ui.Element{
			layout.NewTableCaption(display.Text("Table 2: Quarterly Results")),
			layout.TR(layout.TH(display.Text("Q1")), layout.TH(display.Text("Q2"))),
			layout.TR(layout.TD(display.Text("$45k")), layout.TD(display.Text("$52k"))),
		}, layout.WithCaptionSide(layout.CaptionBottom), layout.WithBorderSpacing(8, 4)),
		display.Spacer(16),

		// 10. Complex Example
		display.Text("10. Complex Table (mixed spans + sections):"),
		display.Spacer(4),
		layout.NewTable([]ui.Element{
			layout.NewTableCaption(display.Text("Employee Schedule")),
			layout.THead(
				layout.TR(
					layout.TH(display.Text("Name")),
					layout.TH(display.Text("Mon")),
					layout.TH(display.Text("Tue")),
					layout.TH(display.Text("Wed")),
					layout.TH(display.Text("Thu")),
					layout.TH(display.Text("Fri")),
				),
			),
			layout.TBody(
				layout.TR(
					layout.TD(display.Text("Alice")),
					layout.TD(display.Text("9-5"), layout.WithColSpan(3)),
					layout.TD(display.Text("Off"), layout.WithColSpan(2)),
				),
				layout.TR(
					layout.TD(display.Text("Bob")),
					layout.TD(display.Text("Off")),
					layout.TD(display.Text("10-6"), layout.WithColSpan(4)),
				),
				layout.TR(
					layout.TD(display.Text("Carol")),
					layout.TD(display.Text("8-4"), layout.WithColSpan(5)),
				),
			),
		}, layout.WithBorderSpacing(8, 4)),
	)
}

// ── Phase 4b Section Views ────────────────────────────────────────

func rtlLayoutSection() ui.Element {
	return layout.Column(
		sectionHeader("RTL Layout (Phase 4b)"),
		display.Text("Insets now support Start/End for direction-aware spacing."),
		display.Spacer(8),

		// Demonstrate InlineInsets — Start=40, End=8
		display.Text("InlineInsets(40, 8) — Start has more padding:"),
		layout.Pad(ui.InlineInsets(40, 8),
			display.Card(layout.Column(
				display.Text("This card uses logical Start/End insets."),
				display.Text("In LTR: Start=Left, End=Right."),
				display.Text("In RTL: Start=Right, End=Left."),
			)),
		),
		display.Spacer(12),

		// Demonstrate LogicalInsets
		display.Text("LogicalInsets(8, 40, 8, 16) — top/end/bottom/start:"),
		layout.Pad(ui.LogicalInsets(8, 40, 8, 16),
			display.Card(display.Text("Four-sided logical insets.")),
		),
		display.Spacer(12),

		// FlexRow automatically mirrors in RTL
		display.Text("FlexRow mirrors child order in RTL:"),
		display.Spacer(4),
		layout.NewFlex([]ui.Element{
			display.BadgeText("First"),
			display.BadgeText("Second"),
			display.BadgeText("Third"),
		}, layout.WithDirection(layout.FlexRow), layout.WithGap(8)),
		display.Spacer(8),

		// JustifyStart resolves to left in LTR, right in RTL
		display.Text("JustifyStart — left-aligned in LTR, right-aligned in RTL:"),
		display.Spacer(4),
		layout.NewFlex([]ui.Element{
			button.Text("Start", nil),
		}, layout.WithDirection(layout.FlexRow), layout.WithJustify(layout.JustifyStart)),
		display.Spacer(8),

		display.Text("Switch to Arabic locale (in 'Locale' section) to see RTL mirroring."),

		display.Spacer(16),
		display.TextStyled("BiDi Run Analysis", draw.TextStyle{
			Size:   14,
			Weight: draw.FontWeightSemiBold,
		}),
		display.Spacer(4),
		display.Text("text.BidiParagraph() decomposes mixed-direction text into runs:"),
		display.Spacer(8),
		bidiAnalysis("Arabic+English", "مرحبا Hello عالم", text.TextDirectionRTL),
		bidiAnalysis("Hebrew+English", "שלום Hello עולם", text.TextDirectionRTL),
		bidiAnalysis("English only", "Hello World", text.TextDirectionLTR),
	)
}

func bidiAnalysis(label, sample string, dir text.TextDirection) ui.Element {
	runs := text.BidiParagraph(sample, dir)
	items := []ui.Element{
		display.Text(fmt.Sprintf("  %s: %q → %d run(s)", label, sample, len(runs))),
	}
	for i, r := range runs {
		dirStr := "LTR"
		if r.Direction == text.TextDirectionRTL {
			dirStr = "RTL"
		}
		items = append(items, display.Text(fmt.Sprintf("    run %d: %s, script=%s, %q",
			i, dirStr, r.Script, r.Text)))
	}
	items = append(items, display.Spacer(4))
	return layout.Column(items...)
}

func localeSection(m Model) ui.Element {
	currentLocale := m.CurrentLocale
	if currentLocale == "" {
		currentLocale = "en (default)"
	}
	return layout.Column(
		sectionHeader("Locale / i18n (Phase 4b)"),
		display.Text("app.WithLocale() sets the BCP 47 locale at startup."),
		display.Text("app.SetLocaleMsg switches locale at runtime."),
		display.Spacer(8),

		display.Text(fmt.Sprintf("Current locale: %s", currentLocale)),
		display.Spacer(8),

		display.Text("Switch locale:"),
		display.Spacer(4),
		layout.Row(
			button.Text("English (LTR)", func() { app.Send(SetLocaleChoiceMsg{Locale: "en"}) }),
			button.Text("العربية (RTL)", func() { app.Send(SetLocaleChoiceMsg{Locale: "ar"}) }),
			button.Text("עברית (RTL)", func() { app.Send(SetLocaleChoiceMsg{Locale: "he"}) }),
			button.Text("Deutsch (LTR)", func() { app.Send(SetLocaleChoiceMsg{Locale: "de"}) }),
		),
		display.Spacer(12),

		display.Text("The layout direction is derived from the locale:"),
		display.Text("  Arabic (ar) → RTL"),
		display.Text("  Hebrew (he) → RTL"),
		display.Text("  English (en), German (de) → LTR"),
		display.Spacer(8),
		display.Text("Switching triggers full layout invalidation."),
	)
}

func imeComposeSection(m Model) ui.Element {
	composeStatus := "No active composition"
	if m.IMEComposeText != "" {
		composeStatus = fmt.Sprintf("Composing: [%s]", m.IMEComposeText)
	}

	return layout.Column(
		sectionHeader("IME Compose (Phase 4b)"),
		display.Text("IME composition support for CJK and other input methods."),
		display.Spacer(8),

		display.Text("New message types:"),
		display.Text("  • IMEComposeMsg — pre-edit text (composition in progress)"),
		display.Text("  • IMECommitMsg — final committed text"),
		display.Spacer(8),

		display.Text("Platform integration:"),
		display.Text("  • Platform.SetIMECursorRect() — positions candidate window"),
		display.Text("  • GLFW: awaiting 3.4 for glfwSetPreeditCallback"),
		display.Text("  • Win32: IMM32 integration planned"),
		display.Spacer(8),

		display.Text("TextField composition state:"),
		display.Text("  • InputState.ComposeText — current pre-edit string"),
		display.Text("  • InputState.ComposeCursorStart/End — cursor range"),
		display.Spacer(8),

		display.Text(fmt.Sprintf("Status: %s", composeStatus)),
		display.Spacer(8),

		display.Text("EventDispatcher routes IME events to focused widget:"),
		display.Text("  • EventIMECompose → focused widget via RenderCtx.Events"),
		display.Text("  • EventIMECommit → focused widget via RenderCtx.Events"),
	)
}

// ── Phase 5 Section Views ──────────────────────────────────────────

func platformInfoSection() ui.Element {
	return layout.Column(
		sectionHeader("Platform Info (Phase 5)"),
		display.Text("Platform-Interface erweitert (RFC §7.1):"),
		display.Spacer(8),

		display.Text("Neue Methoden:"),
		display.Spacer(4),
		display.Text("  SetSize(w, h int) — Fenstergröße ändern"),
		display.Text("  SetFullscreen(bool) — Vollbildmodus umschalten"),
		display.Text("  RequestFrame() — Nächsten Frame anfordern"),
		display.Text("  SetClipboard(text) / GetClipboard() — Zwischenablage"),
		display.Text("  CreateWGPUSurface(instance) — wgpu Surface erstellen"),
		display.Spacer(12),

		display.Text("Verfügbare Backends:"),
		display.Spacer(4),
		display.Text("  • GLFW (macOS/Linux, default) — OpenGL 3.3 Core"),
		display.Text("  • Win32 (Windows, native) — GDI Software / wgpu"),
		display.Text("  • Wayland (Linux, -tags wayland) — Native Wayland + wgpu"),
		display.Text("  • X11 (Linux, -tags x11) — Native X11 + wgpu"),
		display.Text("  • Cocoa (macOS, -tags cocoa) — Native AppKit + Metal"),
		display.Text("  • DRM/KMS (Linux, -tags drm) — Direct framebuffer"),
		display.Spacer(12),

		display.Text("Backend-Auswahl via Build-Tags:"),
		display.Spacer(4),
		display.Text("  go build -tags wayland ./..."),
		display.Text("  go build -tags x11 ./..."),
		display.Text("  go build -tags cocoa ./..."),
		display.Text("  go build -tags drm ./..."),
	)
}

func windowControlsSection(m Model) ui.Element {
	fsLabel := "Enter Fullscreen"
	if m.IsFullscreen {
		fsLabel = "Exit Fullscreen"
	}

	return layout.Column(
		sectionHeader("Window Controls (Phase 5)"),
		display.Text("SetSize — resize the window programmatically:"),
		display.Spacer(8),
		layout.Row(
			button.Text("800×600", func() { app.Send(ResizeWindowMsg{800, 600}) }),
			button.Text("1024×768", func() { app.Send(ResizeWindowMsg{1024, 768}) }),
			button.Text("1280×720", func() { app.Send(ResizeWindowMsg{1280, 720}) }),
		),
		display.Spacer(12),
		display.Text("SetFullscreen — toggle fullscreen mode:"),
		display.Spacer(4),
		button.Text(fsLabel, func() { app.Send(ToggleFullscreenMsg{}) }),
		display.Spacer(4),
		display.Text(fmt.Sprintf("Fullscreen: %v", m.IsFullscreen)),
		display.Spacer(12),
		display.Text("RequestFrame — request immediate repaint:"),
		display.Spacer(4),
		display.Text("Used internally to trigger repaints outside the normal event loop."),
	)
}

func clipboardSection(m Model) ui.Element {
	return layout.Column(
		sectionHeader("Clipboard (Phase 5)"),
		display.Text("SetClipboard / GetClipboard — system clipboard access:"),
		display.Spacer(8),

		display.Text("Text to copy:"),
		display.Spacer(4),
		form.NewTextField(m.ClipboardText, "Enter text to copy...",
			form.WithOnChange(func(v string) { app.Send(SetClipboardTextMsg{v}) }),
			form.WithFocus(app.Focus()),
		),
		display.Spacer(8),

		layout.Row(
			button.Text("Copy to Clipboard", func() { app.Send(CopyToClipboardMsg{}) }),
			button.Text("Paste from Clipboard", func() { app.Send(PasteFromClipboardMsg{}) }),
		),
		display.Spacer(8),
		display.Text(fmt.Sprintf("Current text: %q", m.ClipboardText)),
		display.Spacer(12),

		display.Text("API:"),
		display.Spacer(4),
		display.Text("  app.SetClipboard(text) — set clipboard (package-level)"),
		display.Text("  app.GetClipboard() — get clipboard (package-level)"),
		display.Text("  platform.SetClipboard(text) — per-backend implementation"),
	)
}

func gpuBackendSection() ui.Element {
	return layout.Column(
		sectionHeader("GPU Backend (Phase 5)"),
		display.Text("wgpu — WebGPU-basiertes GPU-Backend (RFC §6.1):"),
		display.Spacer(8),

		display.Text("Zwei Implementierungen:"),
		display.Spacer(4),
		display.Text("  1. wgpu-native (CGo, Default)"),
		display.Text("     Wrapper für die C-Library wgpu-native"),
		display.Text("     Backends: Vulkan (Linux), Metal (macOS), D3D12 (Windows)"),
		display.Spacer(4),
		display.Text("  2. gogpu (Pure Go, -tags gogpu)"),
		display.Text("     Vollständig in Go, keine CGo-Abhängigkeit"),
		display.Text("     Backend: Vulkan via pure-Go Bindings"),
		display.Spacer(12),

		display.Text("Shim-Interface (internal/wgpu/):"),
		display.Spacer(4),
		display.Text("  Instance, Adapter, Device, Surface, SwapChain"),
		display.Text("  RenderPipeline, Buffer, Texture, CommandEncoder"),
		display.Text("  RenderPass, ShaderModule, BindGroup, Queue"),
		display.Spacer(12),

		display.Text("wgpu-Renderer (internal/gpu/wgpu_renderer.go):"),
		display.Spacer(4),
		display.Text("  WGSL-Shader für:"),
		display.Text("    • Rounded Rectangles (SDF-basiert, instanced)"),
		display.Text("    • Atlas-basierte Bitmap-Glyphen (<24px)"),
		display.Text("    • MSDF-Text (>=24px, Chlumsky-Methode)"),
		display.Spacer(12),

		display.Text("Migration von OpenGL 3.3:"),
		display.Spacer(4),
		display.Text("  • GLSL 330 → WGSL Shader-Konvertierung"),
		display.Text("  • glScissor → wgpu RenderPass Clipping"),
		display.Text("  • glDrawArraysInstanced → wgpu DrawInstanced"),
		display.Text("  • glBufferData → wgpu Queue.WriteBuffer"),
	)
}

// ── Phase 6 Section Views ──────────────────────────────────────────

func surfacesSection(pyramid *PyramidSurface) ui.Element {
	return layout.Column(
		sectionHeader("External Surfaces (Phase 6)"),
		display.Text("Surface Slots — RFC §8: Externe Surfaces"),
		display.Spacer(8),

		display.Text("SurfaceProvider Interface:"),
		display.Spacer(4),
		display.Text("  AcquireFrame(bounds) → (TextureID, FrameToken)"),
		display.Text("  ReleaseFrame(token)"),
		display.Text("  HandleMsg(msg) → consumed"),
		display.Spacer(12),

		display.Text("Zero-Copy Paths:"),
		display.Spacer(4),
		display.Text(fmt.Sprintf("  Preferred mode on this platform: %d", ui.PreferredZeroCopyMode())),
		display.Spacer(4),
		display.Text("  • macOS: IOSurface → wgpu Shared Texture"),
		display.Text("  • Linux: DMA-buf → wgpu External Memory"),
		display.Text("  • Windows: DXGI Shared Handle"),
		display.Text("  • Fallback: OSR → CPU-Copy → Upload"),
		display.Spacer(12),

		display.Text("RGB Cube (drag to rotate):"),
		display.Spacer(4),
		ui.Surface(1, pyramid, 400, 300),
		display.Spacer(12),

		display.Text("Input Routing:"),
		display.Spacer(4),
		display.Text("  Mouse/Key events in surface area → SurfaceMouseMsg/SurfaceKeyMsg"),
		display.Text("  Routed via SurfaceProvider.HandleMsg()"),
	)
}

// ── Gradients Section (Phase E) ──────────────────────────────────

func gradientsSection() ui.Element {
	return layout.Column(
		sectionHeader("Gradients (Phase E)"),
		display.Text("GPU-rendered gradient fills via the gradient pipeline:"),
		display.Spacer(12),

		// Linear gradient: 2-stop blue→indigo
		display.Text("Linear Gradient (2 stops):"),
		display.Spacer(4),
		display.GradientRect(200, 60, 8, draw.LinearGradientPaint(
			draw.Pt(0, 0), draw.Pt(200, 0),
			draw.GradientStop{Offset: 0, Color: draw.Hex("#3b82f6")},
			draw.GradientStop{Offset: 1, Color: draw.Hex("#6366f1")},
		)),
		display.Spacer(12),

		// Linear gradient: 4-stop rainbow
		display.Text("Linear Gradient (4 stops):"),
		display.Spacer(4),
		display.GradientRect(200, 60, 8, draw.LinearGradientPaint(
			draw.Pt(0, 0), draw.Pt(200, 0),
			draw.GradientStop{Offset: 0.0, Color: draw.Hex("#ef4444")},
			draw.GradientStop{Offset: 0.33, Color: draw.Hex("#eab308")},
			draw.GradientStop{Offset: 0.66, Color: draw.Hex("#22c55e")},
			draw.GradientStop{Offset: 1.0, Color: draw.Hex("#3b82f6")},
		)),
		display.Spacer(12),

		// Radial gradient
		display.Text("Radial Gradient:"),
		display.Spacer(4),
		display.GradientRect(200, 200, 8, draw.RadialGradientPaint(
			draw.Pt(100, 100), 100,
			draw.GradientStop{Offset: 0, Color: draw.Hex("#ffffff")},
			draw.GradientStop{Offset: 1, Color: draw.Hex("#09090b")},
		)),
		display.Spacer(12),

		// Sharp rect (no radius)
		display.Text("Sharp Linear Gradient (no radius):"),
		display.Spacer(4),
		display.GradientRect(200, 40, 0, draw.LinearGradientPaint(
			draw.Pt(0, 0), draw.Pt(0, 40),
			draw.GradientStop{Offset: 0, Color: draw.Hex("#f97316")},
			draw.GradientStop{Offset: 1, Color: draw.Hex("#dc2626")},
		)),
	)
}

// ── Dialogs Section (Phase 7) ────────────────────────────────────

func dialogsSection(m Model) ui.Element {
	children := []ui.Element{
		sectionHeader("Modal Dialogs"),
		display.Text("Framework-rendered dialog overlays with backdrop scrim:"),
		display.Spacer(8),
		layout.Row(
			button.Text("Info", func() { app.Send(ShowMsgDialogMsg{Kind: platform.DialogInfo}) }),
			display.Spacer(8),
			button.Text("Warning", func() { app.Send(ShowMsgDialogMsg{Kind: platform.DialogWarning}) }),
			display.Spacer(8),
			button.Text("Error", func() { app.Send(ShowMsgDialogMsg{Kind: platform.DialogError}) }),
		),
		display.Spacer(8),
		layout.Row(
			button.Text("Confirm", func() { app.Send(ShowConfirmDialogMsg{}) }),
			display.Spacer(8),
			button.Text("Input", func() { app.Send(ShowInputDialogMsg{}) }),
			display.Spacer(8),
			button.OutlinedText("Native Confirm", func() { app.Send(NativeConfirmMsg{}) }),
		),
	}

	if m.DialogResult != "" {
		children = append(children,
			display.Spacer(8),
			display.Text(fmt.Sprintf("Result: %s", m.DialogResult)),
		)
	}

	// Render active dialog overlays.
	if m.ShowMsgDialog {
		children = append(children, uidialog.MessageDialog(
			"demo-msg-dialog",
			"Message",
			"This is a sample message dialog.",
			m.DialogMsgKind,
			func() { app.Send(DismissDialogMsg{}) },
		))
	}
	if m.ShowConfirmDialog {
		children = append(children, uidialog.ConfirmDialog(
			"demo-confirm-dialog",
			"Confirm Action",
			"Are you sure you want to proceed?",
			func() { app.Send(DialogConfirmedMsg{}) },
			func() { app.Send(DismissDialogMsg{}) },
		))
	}
	if m.ShowInputDialog {
		children = append(children, uidialog.InputDialog(
			"demo-input-dialog",
			"Enter Value",
			"Please provide a value:",
			m.InputDialogValue,
			"Type here...",
			func(v string) { app.Send(DialogInputChangedMsg{Value: v}) },
			func() { app.Send(DialogConfirmedMsg{}) },
			func() { app.Send(DismissDialogMsg{}) },
		))
	}

	return layout.Column(children...)
}

// ── Blur Section (Phase F) ───────────────────────────────────────

func blurSection() ui.Element {
	radii := []float32{4, 8, 16, 32, 64}
	items := make([]ui.Element, 0, len(radii)+3)
	items = append(items,
		sectionHeader("Blur (Phase F)"),
		display.Text("Gaussian blur at various radii (PushBlur / PopBlur):"),
		display.Spacer(8),
	)

	for _, r := range radii {
		radius := r
		items = append(items,
			display.Spacer(4),
			effects.BlurBox(radius,
				layout.NewStack(
					display.GradientRect(200, 60, 8, draw.LinearGradientPaint(
						draw.Pt(0, 0), draw.Pt(200, 0),
						draw.GradientStop{Offset: 0, Color: draw.Hex("#3366cc")},
						draw.GradientStop{Offset: 1, Color: draw.Hex("#ffcc33")},
					)),
					layout.Sized(200, 60,
						layout.Pad(ui.UniformInsets(8),
							display.Text(fmt.Sprintf("radius = %.0f", radius)),
						),
					),
				),
			),
		)
	}

	return layout.Column(items...)
}

// ── Effects Section (Phase G) ────────────────────────────────────

func effectsSection() ui.Element {
	items := []ui.Element{
		sectionHeader("Effects (Phase G)"),
	}

	// --- Shadows ---
	items = append(items,
		display.Text("Soft Shadows (None / Low / Med / High):"),
		display.Spacer(8),
	)

	type shadowLevel struct {
		label string
		shadow draw.Shadow
		bg     draw.Color // card background — distinct per level, visible in both themes
	}
	levels := []shadowLevel{
		{"None", draw.Shadow{}, draw.Hex("#6366f1")},       // indigo
		{"Low", draw.Shadow{Color: draw.Color{R: 0, G: 0, B: 0, A: 0.14}, BlurRadius: 12, OffsetY: 3, Radius: 8}, draw.Hex("#3b82f6")},  // blue
		{"Med", draw.Shadow{Color: draw.Color{R: 0, G: 0, B: 0, A: 0.20}, BlurRadius: 24, OffsetY: 6, Radius: 8}, draw.Hex("#0ea5e9")}, // sky
		{"High", draw.Shadow{Color: draw.Color{R: 0, G: 0, B: 0, A: 0.28}, BlurRadius: 40, OffsetY: 10, Radius: 8}, draw.Hex("#14b8a6")}, // teal
	}
	shadowCards := make([]ui.Element, len(levels))
	for i, lv := range levels {
		shadowCards[i] = layout.Pad(ui.UniformInsets(20),
			effects.ShadowBox(lv.shadow, 8,
				layout.NewStack(
					display.GradientRect(132, 82, 8, draw.SolidPaint(lv.bg)),
					layout.Pad(ui.UniformInsets(16),
						layout.Sized(100, 50,
							display.Text(lv.label),
						),
					),
				),
			),
		)
	}
	items = append(items, layout.Row(shadowCards...))

	// --- Opacity ---
	items = append(items,
		display.Spacer(16),
		display.Text("Opacity (1.0, 0.75, 0.5, 0.25):"),
		display.Spacer(8),
	)

	alphas := []float32{1.0, 0.75, 0.5, 0.25}
	opacityBoxes := make([]ui.Element, len(alphas))
	for i, a := range alphas {
		opacityBoxes[i] = layout.Pad(ui.UniformInsets(4),
			effects.OpacityBox(a,
				layout.NewStack(
					display.GradientRect(104, 64, 6, draw.SolidPaint(draw.Hex("#3b82f6"))),
					layout.Pad(ui.UniformInsets(12),
						layout.Sized(80, 40,
							display.Text(fmt.Sprintf("%.0f%%", a*100)),
						),
					),
				),
			),
		)
	}
	items = append(items, layout.Row(opacityBoxes...))

	// --- Frosted Glass ---
	items = append(items,
		display.Spacer(16),
		display.Text("Frosted Glass (blur backdrop + sharp overlay panel):"),
		display.Spacer(8),
	)

	items = append(items,
		layout.NewStack(
			// Complex background: colorful checkerboard pattern makes blur effect obvious
			display.CheckerRect(420, 160, 16),
			// Frosted glass panel overlaid on the pattern
			layout.Pad(draw.Insets{Top: 24, Left: 50, Right: 50, Bottom: 24},
				effects.FrostedGlass(16, draw.Color{R: 1, G: 1, B: 1, A: 0.18},
					layout.Sized(320, 112,
						layout.Pad(ui.UniformInsets(16),
							layout.Column(
								display.Text("Frosted Glass Panel"),
								display.Spacer(4),
								display.Text("Background is blurred, text stays sharp."),
							),
						),
					),
				),
			),
		),
	)

	// --- Inner Shadow (Tier 2) ---
	items = append(items,
		display.Spacer(16),
		display.Text("Inner Shadow (light inset, deeper inset):"),
		display.Spacer(8),
	)

	innerShadowCards := []ui.Element{
		layout.Pad(ui.UniformInsets(12),
			effects.InnerShadowBox(
				draw.Shadow{Color: draw.Color{R: 0, G: 0, B: 0, A: 0.5}, BlurRadius: 10},
				8,
				layout.NewStack(
					display.GradientRect(132, 82, 8, draw.SolidPaint(draw.Hex("#e2e8f0"))),
					layout.Pad(ui.UniformInsets(16),
						layout.Sized(100, 50, display.Text("Light Inset")),
					),
				),
			),
		),
		layout.Pad(ui.UniformInsets(12),
			effects.InnerShadowBox(
				draw.Shadow{Color: draw.Color{R: 0, G: 0, B: 0, A: 0.85}, BlurRadius: 20},
				8,
				layout.NewStack(
					display.GradientRect(132, 82, 8, draw.SolidPaint(draw.Hex("#cbd5e1"))),
					layout.Pad(ui.UniformInsets(16),
						layout.Sized(100, 50, display.Text("Deep Inset")),
					),
				),
			),
		),
	}
	items = append(items, layout.Row(innerShadowCards...))

	// --- Elevation / Hover-Responsive Shadows (Tier 2) ---
	items = append(items,
		display.Spacer(16),
		display.Text("Elevation (hover to see shadow lift):"),
		display.Spacer(8),
	)

	elevationCards := []ui.Element{
		layout.Pad(ui.UniformInsets(16),
			effects.ElevationCard(nil,
				layout.NewStack(
					display.GradientRect(132, 82, 8, draw.SolidPaint(draw.Hex("#6366f1"))),
					layout.Pad(ui.UniformInsets(16),
						layout.Sized(100, 50, display.Text("Card A")),
					),
				),
			),
		),
		layout.Pad(ui.UniformInsets(16),
			effects.ElevationCard(nil,
				layout.NewStack(
					display.GradientRect(132, 82, 8, draw.SolidPaint(draw.Hex("#3b82f6"))),
					layout.Pad(ui.UniformInsets(16),
						layout.Sized(100, 50, display.Text("Card B")),
					),
				),
			),
		),
		layout.Pad(ui.UniformInsets(16),
			effects.ElevationCard(nil,
				layout.NewStack(
					display.GradientRect(132, 82, 8, draw.SolidPaint(draw.Hex("#0ea5e9"))),
					layout.Pad(ui.UniformInsets(16),
						layout.Sized(100, 50, display.Text("Card C")),
					),
				),
			),
		),
	}
	items = append(items, layout.Row(elevationCards...))

	// --- Vibrancy / Tinted Blur (Tier 2) ---
	items = append(items,
		display.Spacer(16),
		display.Text("Vibrancy vs Frosted Glass (accent-tinted blur — compare the color cast):"),
		display.Spacer(8),
	)

	items = append(items,
		layout.Row(
			// Frosted Glass reference (neutral white tint)
			layout.Pad(ui.UniformInsets(8),
				layout.NewStack(
					display.CheckerRect(210, 160, 16),
					layout.Pad(draw.Insets{Top: 20, Left: 16, Right: 16, Bottom: 20},
						effects.FrostedGlass(16, draw.Color{R: 1, G: 1, B: 1, A: 0.18},
							layout.Sized(178, 120,
								layout.Pad(ui.UniformInsets(12),
									layout.Column(
										display.Text("Frosted Glass"),
										display.Spacer(4),
										display.Text("Neutral white tint"),
									),
								),
							),
						),
					),
				),
			),
			// Vibrancy (accent-tinted, visibly colored)
			layout.Pad(ui.UniformInsets(8),
				layout.NewStack(
					display.CheckerRect(210, 160, 16),
					layout.Pad(draw.Insets{Top: 20, Left: 16, Right: 16, Bottom: 20},
						effects.Vibrancy(0.35,
							layout.Sized(178, 120,
								layout.Pad(ui.UniformInsets(12),
									layout.Column(
										display.Text("Vibrancy"),
										display.Spacer(4),
										display.Text("Accent-tinted blur"),
									),
								),
							),
						),
					),
				),
			),
		),
	)

	// --- Noise/Grain (Tier 3) ---
	items = append(items,
		display.Spacer(16),
		display.Text("Noise/Grain (always-on dither — zoom in to see noise preventing banding):"),
		display.Spacer(8),
	)
	items = append(items,
		layout.Row(
			layout.Pad(ui.UniformInsets(8),
				display.GradientRect(200, 80, 8, draw.LinearGradientPaint(
					draw.Pt(0, 0), draw.Pt(200, 0),
					draw.GradientStop{Offset: 0, Color: draw.Hex("#1e3a5f")},
					draw.GradientStop{Offset: 1, Color: draw.Hex("#4a90d9")},
				)),
			),
			layout.Pad(ui.UniformInsets(8),
				display.GradientRect(200, 80, 8, draw.LinearGradientPaint(
					draw.Pt(0, 0), draw.Pt(0, 80),
					draw.GradientStop{Offset: 0, Color: draw.Hex("#2d1b4e")},
					draw.GradientStop{Offset: 1, Color: draw.Hex("#7c3aed")},
				)),
			),
		),
	)

	// --- Glow (Tier 3) ---
	items = append(items,
		display.Spacer(16),
		display.Text("Glow (soft outer glow using shadow pipeline):"),
		display.Spacer(8),
	)
	glowCards := []ui.Element{
		layout.Pad(ui.UniformInsets(16),
			effects.Glow(12, 8,
				layout.Sized(120, 80,
					layout.Pad(ui.UniformInsets(12),
						layout.Column(
							display.Text("Accent"),
							display.Spacer(4),
							display.Text("blur=12"),
						),
					),
				),
			),
		),
		layout.Pad(ui.UniformInsets(16),
			effects.GlowBox(draw.Color{R: 0.2, G: 0.9, B: 0.4, A: 0.6}, 16, 8,
				layout.Sized(120, 80,
					layout.Pad(ui.UniformInsets(12),
						layout.Column(
							display.Text("Green"),
							display.Spacer(4),
							display.Text("blur=16"),
						),
					),
				),
			),
		),
		layout.Pad(ui.UniformInsets(16),
			effects.GlowBox(draw.Color{R: 0.9, G: 0.2, B: 0.2, A: 0.6}, 20, 8,
				layout.Sized(120, 80,
					layout.Pad(ui.UniformInsets(12),
						layout.Column(
							display.Text("Red"),
							display.Spacer(4),
							display.Text("blur=20"),
						),
					),
				),
			),
		),
	}
	items = append(items, layout.Row(glowCards...))

	return layout.Column(items...)
}

// ── Multi-Window Section (Phase F) ──────────────────────────────

func multiWindowSection(m Model) ui.Element {
	var btn ui.Element
	if m.SecondWindowOpen {
		btn = button.Text("Close Second Window", func() {
			app.Send(app.CloseWindowMsg{ID: 1})
		})
	} else {
		btn = button.Text("Open Second Window", func() {
			app.Send(app.OpenWindowMsg{
				ID: 1,
				Config: app.WindowConfig{
					Title:     "Lux — Second Window",
					Type:      app.WindowTypeNormal,
					Width:     400,
					Height:    300,
					Resizable: true,
				},
			})
		})
	}

	return layout.Column(
		sectionHeader("Multi-Window (Phase F)"),
		display.Text("Multi-window support:"),
		display.Spacer(8),
		layout.Pad(ui.UniformInsets(8), btn),
		display.Spacer(4),
		display.Text(fmt.Sprintf("Second window open: %v", m.SecondWindowOpen)),
	)
}

// ── Image Helpers ────────────────────────────────────────────────

// generateColorChecker creates a checkerboard image with the given two colors.
func generateColorChecker(store *luximage.Store, w, h, cellSize int, r1, g1, b1, r2, g2, b2 byte) draw.ImageID {
	rgba := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			off := (y*w + x) * 4
			if ((x/cellSize)+(y/cellSize))%2 == 0 {
				rgba[off], rgba[off+1], rgba[off+2], rgba[off+3] = r1, g1, b1, 255
			} else {
				rgba[off], rgba[off+1], rgba[off+2], rgba[off+3] = r2, g2, b2, 255
			}
		}
	}
	id, err := store.LoadFromRGBA(w, h, rgba)
	if err != nil {
		log.Printf("generateColorChecker: %v", err)
		return 0
	}
	return id
}

// ── Images Section ───────────────────────────────────────────────

func imagesSection(m Model) ui.Element {
	return layout.Column(
		sectionHeader("Images"),
		display.Text("The ui.Image widget renders loaded images with size, scale mode, and opacity options."),
		display.Text("Images are loaded via image.Store and referenced by draw.ImageID."),
		display.Spacer(12),

		// 1. Basic image display — blue/gray checkerboard stretched to various sizes
		display.TextStyled("Blue Checker (64×64 source → stretched to various sizes)", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
		display.Spacer(4),
		layout.Row(
			display.Image(m.ImgChecker1, display.WithImageSize(64, 64), display.WithImageScaleMode(draw.ImageScaleStretch)),
			display.Spacer(8),
			display.Image(m.ImgChecker1, display.WithImageSize(128, 64), display.WithImageScaleMode(draw.ImageScaleStretch)),
			display.Spacer(8),
			display.Image(m.ImgChecker1, display.WithImageSize(48, 48), display.WithImageScaleMode(draw.ImageScaleStretch)),
		),
		display.Spacer(16),

		// 2. Scale modes — orange/teal checkerboard (128×64 → 100×100 box)
		display.TextStyled("Scale Modes (orange/teal 128×64 → 100×100)", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
		display.Spacer(4),
		layout.Row(
			layout.Column(
				display.Text("Fit"),
				display.Image(m.ImgChecker2, display.WithImageSize(100, 100), display.WithImageScaleMode(draw.ImageScaleFit)),
			),
			display.Spacer(12),
			layout.Column(
				display.Text("Fill"),
				display.Image(m.ImgChecker2, display.WithImageSize(100, 100), display.WithImageScaleMode(draw.ImageScaleFill)),
			),
			display.Spacer(12),
			layout.Column(
				display.Text("Stretch"),
				display.Image(m.ImgChecker2, display.WithImageSize(100, 100), display.WithImageScaleMode(draw.ImageScaleStretch)),
			),
		),
		display.Spacer(16),

		// 3. Opacity control — pink/green checkerboard
		display.TextStyled("Opacity (pink/green checker)", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
		display.Spacer(4),
		layout.Row(
			display.Image(m.ImgChecker3, display.WithImageSize(120, 60), display.WithImageScaleMode(draw.ImageScaleStretch), display.WithImageOpacity(m.ImageOpacity)),
			display.Spacer(12),
			display.Text(fmt.Sprintf("%.0f%%", m.ImageOpacity*100)),
		),
		form.NewSlider(m.ImageOpacity, func(v float32) { app.Send(SetImageOpacityMsg{v}) }),
		display.Spacer(16),

		// 4. Alt text — blue checker again
		display.TextStyled("Accessibility", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
		display.Spacer(4),
		display.Image(m.ImgChecker1,
			display.WithImageSize(64, 64),
			display.WithImageScaleMode(draw.ImageScaleStretch),
			display.WithImageAlt("Blue and white checkerboard pattern"),
		),
		display.Text("  Alt: \"Blue and white checkerboard pattern\""),
		display.Spacer(16),

		// 5. API reference
		display.TextStyled("API", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
		display.Spacer(4),
		display.Text("  store := image.NewStore()"),
		display.Text("  id, _ := store.LoadFromFile(\"photo.png\")"),
		display.Text("  id, _ := store.LoadFromBytes(data)"),
		display.Text("  id, _ := store.LoadFromRGBA(w, h, rgba)"),
		display.Text("  display.Image(id, display.WithImageSize(200, 150))"),
		display.Text("  display.Image(id, display.WithImageScaleMode(draw.ImageScaleFit))"),
		display.Text("  display.Image(id, display.WithImageOpacity(0.5))"),
	)
}

// ── Shader Effects Section ───────────────────────────────────────

func shaderEffectsSection() ui.Element {
	// Create Paint variant examples for display.
	noisePaint := draw.ShaderEffectPaint(draw.ShaderEffectNoise, 8.0)
	plasmaPaint := draw.ShaderEffectPaint(draw.ShaderEffectPlasma, 2.0)
	voronoiPaint := draw.ShaderEffectPaint(draw.ShaderEffectVoronoi, 12.0)
	_ = noisePaint
	_ = plasmaPaint
	_ = voronoiPaint

	return layout.Column(
		sectionHeader("Shader Effects"),
		display.Text("GPU shader-based visual effects via the Paint system."),
		display.Text("Requires WGPU backend for rendering."),
		display.Spacer(12),

		// Built-in effects
		display.TextStyled("Built-in Shader Effects", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
		display.Spacer(4),
		display.Text("  ShaderEffectNoise   — Simplex/Perlin noise pattern"),
		display.Text("  ShaderEffectPlasma  — Animated plasma effect"),
		display.Text("  ShaderEffectVoronoi — Voronoi cell pattern"),
		display.Spacer(12),

		// Paint API
		display.TextStyled("Paint Variants for Backgrounds", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
		display.Spacer(4),
		display.Text("Image-based paints:"),
		display.Text("  draw.ImagePaint(id, draw.ImageScaleFit)     — stretched/fitted image fill"),
		display.Text("  draw.PatternPaint(id, draw.Size{32, 32})    — tiled image fill"),
		display.Spacer(8),
		display.Text("Shader paints:"),
		display.Text("  draw.ShaderEffectPaint(draw.ShaderEffectNoise, 8.0)"),
		display.Text("  draw.ShaderEffectPaint(draw.ShaderEffectPlasma, 2.0)"),
		display.Text("  draw.ShaderEffectPaint(draw.ShaderEffectVoronoi, 12.0)"),
		display.Spacer(8),
		display.Text("Custom WGSL shader:"),
		display.Text("  draw.ShaderPaint(wgslSource, params...)"),
		display.Spacer(8),
		display.Text("Shader + image texture:"),
		display.Text("  draw.ShaderImagePaint(imgID, wgslSource, params...)"),
		display.Spacer(16),

		// Integration notes
		display.TextStyled("Integration", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
		display.Spacer(4),
		display.Text("Paints are used as fill styles for surfaces and backgrounds."),
		display.Text("Custom WGSL fragments receive uniforms via Params[0..7] and"),
		display.Text("an optional image texture for PaintShaderImage."),
		display.Text("Built-in effects are pre-compiled and cached by the GPU renderer."),
	)
}

// ── Accessibility Sections ────────────────────────────────────────

// buildA11yTreeDemo builds an AccessTree from a sample widget tree
// and returns an indented text representation.
func buildA11yTreeDemo(m Model) string {
	// Build a sample UI to demonstrate AccessTree construction.
	sampleUI := layout.Column(
		button.Text("Save", func() {}),
		form.NewCheckbox("Accept Terms", m.CheckA, nil),
		form.NewSlider(m.SliderVal, nil),
		form.NewTextField(m.TextValue, "Enter text..."),
		form.NewProgressBar(m.Progress),
		form.NewToggle(m.ToggleOn, nil),
	)

	tree := ui.RenderToAccessTree(sampleUI)
	return formatAccessTree(&tree)
}

// formatAccessTree renders an AccessTree as indented text.
func formatAccessTree(tree *a11y.AccessTree) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("AccessTree: %d nodes\n", len(tree.Nodes)))
	sb.WriteString("─────────────────────────────\n")

	for i := range tree.Nodes {
		node := &tree.Nodes[i]
		depth := 0
		pidx := node.ParentIndex
		for pidx >= 0 {
			depth++
			pidx = tree.Nodes[pidx].ParentIndex
		}

		indent := strings.Repeat("  ", depth)
		role := roleName(node.Node.Role)
		label := node.Node.Label
		if label == "" {
			label = "(no label)"
		}

		line := fmt.Sprintf("%s[%s] %s", indent, role, label)

		// Add state annotations.
		var states []string
		if node.Node.States.Focused {
			states = append(states, "focused")
		}
		if node.Node.States.Checked {
			states = append(states, "checked")
		}
		if node.Node.States.Disabled {
			states = append(states, "disabled")
		}
		if node.Node.States.Expanded {
			states = append(states, "expanded")
		}
		if node.Node.NumericValue != nil {
			states = append(states, fmt.Sprintf("value=%.2f", node.Node.NumericValue.Current))
		}
		if node.Node.TextState != nil {
			states = append(states, fmt.Sprintf("len=%d", node.Node.TextState.Length))
		}
		if len(states) > 0 {
			line += " {" + strings.Join(states, ", ") + "}"
		}

		sb.WriteString(line + "\n")
	}
	return sb.String()
}

// roleName returns a human-readable name for an AccessRole.
func roleName(role a11y.AccessRole) string {
	switch role {
	case a11y.RoleButton:
		return "Button"
	case a11y.RoleCheckbox:
		return "Checkbox"
	case a11y.RoleCombobox:
		return "Combobox"
	case a11y.RoleDialog:
		return "Dialog"
	case a11y.RoleGrid:
		return "Grid"
	case a11y.RoleGroup:
		return "Group"
	case a11y.RoleHeading:
		return "Heading"
	case a11y.RoleImage:
		return "Image"
	case a11y.RoleLink:
		return "Link"
	case a11y.RoleListbox:
		return "Listbox"
	case a11y.RoleMenu:
		return "Menu"
	case a11y.RoleProgressBar:
		return "ProgressBar"
	case a11y.RoleScrollBar:
		return "ScrollBar"
	case a11y.RoleSlider:
		return "Slider"
	case a11y.RoleSpinButton:
		return "SpinButton"
	case a11y.RoleTab:
		return "Tab"
	case a11y.RoleTable:
		return "Table"
	case a11y.RoleTextInput:
		return "TextInput"
	case a11y.RoleToggle:
		return "Toggle"
	case a11y.RoleTree:
		return "Tree"
	default:
		if role >= a11y.RoleCustomBase {
			return fmt.Sprintf("Custom(%d)", role-a11y.RoleCustomBase)
		}
		return fmt.Sprintf("Role(%d)", role)
	}
}

func a11yTreeSection(m Model) ui.Element {
	children := []ui.Element{
		sectionHeader("AccessTree Inspector"),
		display.Text("Build an AccessTree from a sample widget tree and inspect the result."),
		display.Text("Uses RenderToAccessTree() — the test helper from 6.2c."),
		display.Spacer(12),

		display.TextStyled("Sample Widgets", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
		display.Spacer(4),
		display.Text("  Button(\"Save\")"),
		display.Text("  Checkbox(\"Accept Terms\")"),
		display.Text("  Slider(0.5)"),
		display.Text("  TextField(\"Enter text...\")"),
		display.Text("  ProgressBar(0.0)"),
		display.Text("  Toggle(on/off)"),
		display.Spacer(12),

		button.Text("Build AccessTree", func() { app.Send(BuildA11yTreeMsg{}) }),
	}

	if m.A11yTreeText != "" {
		children = append(children,
			display.Spacer(12),
			display.TextStyled("Result", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
			display.Spacer(4),
		)
		for _, line := range strings.Split(m.A11yTreeText, "\n") {
			if line != "" {
				children = append(children, display.Text(line))
			}
		}
	}

	return layout.Column(children...)
}

func a11yFocusTrapSection(m Model) ui.Element {
	children := []ui.Element{
		sectionHeader("FocusTrap Demo"),
		display.Text("Modal dialogs trap Tab/Shift+Tab navigation within the dialog."),
		display.Text("Focus is restored to the trigger when the dialog closes."),
		display.Spacer(12),

		display.TextStyled("RFC-001 §11.7 — Focus Trapping Rules", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
		display.Spacer(4),
		display.Text("  1. Modal opens → Focus moves into dialog"),
		display.Text("  2. Tab at last widget → Wraps to first widget"),
		display.Text("  3. Shift+Tab at first → Wraps to last widget"),
		display.Text("  4. Escape → Dialog closes, focus restores"),
		display.Text("  5. Background content hidden from AccessTree"),
		display.Spacer(12),

		button.Text("Open Modal Dialog", func() { app.Send(ToggleA11yTrapMsg{}) }),
	}

	if m.A11yTrapResult != "" {
		children = append(children,
			display.Spacer(8),
			display.Text(fmt.Sprintf("Result: %s", m.A11yTrapResult)),
		)
	}

	// Render the modal dialog with FocusTrap.
	if m.A11yTrapOpen {
		children = append(children, ui.Overlay{
			ID:          "a11y-trap-demo",
			Placement:   ui.PlacementCenter,
			Dismissable: true,
			OnDismiss:   func() { app.Send(DismissA11yTrapMsg{}) },
			Backdrop:    true,
			FocusTrap:   &ui.FocusTrap{RestoreFocus: true, TrapID: "a11y-trap-demo"},
			Content: layout.Sized(360, 0,
				layout.Column(
					display.TextStyled("FocusTrap Demo Dialog", draw.TextStyle{Size: 16, Weight: draw.FontWeightSemiBold}),
					display.Spacer(12),
					display.Text("Tab/Shift+Tab should cycle within this dialog only."),
					display.Spacer(12),
					form.NewTextField(m.A11yTrapText, "Enter value...", form.WithOnChange(func(v string) { app.Send(SetA11yTrapTextMsg{Value: v}) })),
					display.Spacer(8),
					form.NewCheckbox("Option A", m.A11yTrapCheckA, func(v bool) { app.Send(SetA11yTrapCheckAMsg{Value: v}) }),
					display.Spacer(4),
					form.NewCheckbox("Option B", m.A11yTrapCheckB, func(v bool) { app.Send(SetA11yTrapCheckBMsg{Value: v}) }),
					display.Spacer(16),
					layout.Row(
						display.Spacer(0),
						button.OutlinedText("Cancel", func() { app.Send(DismissA11yTrapMsg{}) }),
						display.Spacer(8),
						button.Text("Confirm", func() { app.Send(A11yTrapConfirmMsg{}) }),
					),
				),
			),
		})
	}

	return layout.Column(children...)
}

func a11yBridgeSection() ui.Element {
	os := runtime.GOOS
	var bridgeName, bridgeDesc string
	switch os {
	case "windows":
		bridgeName = "UIA (UI Automation)"
		bridgeDesc = "Windows platform bridge via CGo/COM. Exposes AccessTree " +
			"as UIA element providers (IRawElementProviderSimple). " +
			"Handles WM_GETOBJECT, structural change events, focus events."
	case "darwin":
		bridgeName = "NSAccessibility"
		bridgeDesc = "macOS platform bridge via CGo/ObjC. Creates " +
			"LuxAccessibilityElement objects mapped to AccessTreeNodes. " +
			"Posts AXFocusedUIElementChanged and announcement notifications."
	case "linux":
		bridgeName = "AT-SPI2"
		bridgeDesc = "Linux platform bridge via D-Bus (godbus). No CGo required. " +
			"Exposes accessible objects on D-Bus implementing " +
			"org.a11y.atspi.Accessible, Component, Action, Value, and Text interfaces. " +
			"Works on X11, Wayland, and DRM/KMS (bare metal)."
	default:
		bridgeName = "None"
		bridgeDesc = "No accessibility bridge available for this platform."
	}

	return layout.Column(
		sectionHeader("Platform A11y Bridge"),
		display.Text("Lux provides platform-specific accessibility bridges that expose"),
		display.Text("the AccessTree to native screen readers and assistive technology."),
		display.Spacer(12),

		display.TextStyled("Current Platform", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
		display.Spacer(4),
		display.Text(fmt.Sprintf("  OS:     %s/%s", runtime.GOOS, runtime.GOARCH)),
		display.Text(fmt.Sprintf("  Bridge: %s", bridgeName)),
		display.Spacer(8),
		display.Text(bridgeDesc),
		display.Spacer(16),

		display.TextStyled("Bridge Interface (a11y.A11yBridge)", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
		display.Spacer(4),
		display.Text("  UpdateTree(tree AccessTree)"),
		display.Text("    → Replaces the access tree snapshot after each reconcile pass"),
		display.Text("  NotifyFocus(nodeID AccessNodeID)"),
		display.Text("    → Informs the bridge that keyboard focus moved"),
		display.Text("  NotifyLiveRegion(nodeID AccessNodeID, text string)"),
		display.Text("    → Announces dynamic content changes"),
		display.Spacer(16),

		display.TextStyled("Available Bridges", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
		display.Spacer(4),
		display.Text("  Windows  — UIA (UI Automation) via CGo/COM"),
		display.Text("  macOS    — NSAccessibility via CGo/ObjC"),
		display.Text("  Linux    — AT-SPI2 via D-Bus (godbus, no CGo)"),
		display.Text("  DRM/KMS  — AT-SPI2 via System D-Bus (bare metal HMI)"),
	)
}

// ── Main ─────────────────────────────────────────────────────────

func main() {
	initial := Model{
		Dark:               true,
		RadioChoice:        "alpha",
		SliderVal:          0.5,
		Progress:           0.0,
		SelectVal:          "Option 1",
		SelectState:        &form.SelectState{},
		ValRoleState:       &form.SelectState{},
		Scroll:             &ui.ScrollState{},
		ToggleAnim:         form.NewToggleState(),
		NavTree:            ui.NewTreeState(),
		ActiveSection:      "typography",
		VListScroll:        &ui.ScrollState{},
		DemoTree:           ui.NewTreeState(),
		PagedContacts:      data.NewPagedDataset[int](20),
		PagedScroll:        &ui.ScrollState{},
		StreamLog:          data.NewStreamDataset[int](data.StreamAppend),
		StreamScroll:       &ui.ScrollState{},
		// DataTable demos
		DTSliceState:  data.NewDataTableState(),
		DTPagedState:  data.NewDataTableState(),
		DTPaged:       data.NewPagedDataset[int](20),
		DTStreamState: data.NewDataTableState(),
		DTStream:      data.NewStreamDataset[int](data.StreamAppend),
		DTSelectedRow: -1,
		AccordionState:     nav.NewAccordionState(),
		MenuBarState:       menu.NewMenuBarState(),
		KineticScroll:      ui.NewKineticScroll(theme.Default.Tokens().Scroll),
		NavSplitRatio:      0.25,
		SplitHorizontal:    0.5,
		SplitVertical:      0.5,
		SplitNested1:       0.25,
		SplitNested2:       0.65,
		SplitThreeColLeft:  0.2,
		SplitThreeColRight: 0.5,
		SpringPreset:       "gentle",
		BezierPreset:       "ease",
		MotionPreset:       "standard",
		LayoutGap:          30,
		CurrentLocale:      "en",
		Pyramid:            NewPyramidSurface(),
		ImageStore:          luximage.NewStore(),
		ImageOpacity:        1.0,
		TextAreaScroll:      &ui.ScrollState{},
		// Toolbar demo defaults
		ToolbarDoc: richtext.Build(
			richtext.S("Format this text using the toolbar above.\n"),
			richtext.S("Try toggling "),
			richtext.S("Bold", richtext.SpanStyle{Bold: true}),
			richtext.S(", "),
			richtext.S("Italic", richtext.SpanStyle{Italic: true}),
			richtext.S(", and Underline."),
		),
		ToolbarDocScroll: &ui.ScrollState{},
		// Pickers & Numeric defaults
		DateVal:    time.Now(),
		DateState:  &form.DatePickerState{},
		ColorVal:   draw.Color{R: 0.25, G: 0.32, B: 0.71, A: 1}, // Indigo
		ColorState: &form.ColorPickerState{},
		TimeHour:   14,
		TimeMinute: 30,
		TimeState:  &form.TimePickerState{},
		NumericVal:         42,
		FilePickerState:    form.NewFilePickerState(""),
		FilePickerDirState: form.NewFilePickerState(""),
	}
	// Generate procedural demo images — each with distinct colors for easy identification.
	initial.ImgChecker1 = generateColorChecker(initial.ImageStore, 64, 64, 8,
		220, 220, 240, // light blue-gray
		59, 130, 246,  // blue
	)
	initial.ImgChecker2 = generateColorChecker(initial.ImageStore, 128, 64, 12,
		255, 160, 50, // orange
		30, 180, 160, // teal
	)
	initial.ImgChecker3 = generateColorChecker(initial.ImageStore, 120, 60, 10,
		230, 80, 160, // pink
		80, 200, 80,  // green
	)

	// SVG demo images — rasterized at load time.
	initial.SvgStarID = loadDemoSVG(initial.ImageStore, svgStar, 120, 120)
	initial.SvgCirclesID = loadDemoSVG(initial.ImageStore, svgCircles, 200, 100)

	initial.FadeOpacity.SetImmediate(1.0)

	// Global handler that logs key events (Phase 1: §2.8).
	keyLogger := func(ev ui.InputEvent) bool {
		if ev.Kind == ui.EventKey && ev.Key != nil {
			app.Send(SetHandlerLogMsg{Text: fmt.Sprintf("Key=%d Action=%d", ev.Key.Key, ev.Key.Action)})
		}
		return false // don't consume — let events pass through
	}

	// Persistence config: save/restore selected fields as JSON.
	type persistedState struct {
		Dark          bool   `json:"dark"`
		Count         int    `json:"count"`
		ActiveSection string `json:"active_section"`
		SubCounter    int    `json:"sub_counter"`
	}
	persistence := app.WithPersistence(app.PersistenceConfig[Model]{
		StorageKey: "kitchen-sink-state",
		Encode: func(m Model) ([]byte, error) {
			return json.Marshal(persistedState{
				Dark:          m.Dark,
				Count:         m.Count,
				ActiveSection: m.ActiveSection,
				SubCounter:    m.SubCounter,
			})
		},
		Decode: func(data []byte) (Model, error) {
			var ps persistedState
			if err := json.Unmarshal(data, &ps); err != nil {
				return Model{}, err
			}
			// Restore into a fresh model with proper widget states.
			restored := initial
			restored.Dark = ps.Dark
			restored.Count = ps.Count
			restored.ActiveSection = ps.ActiveSection
			restored.SubCounter = ps.SubCounter
			return restored, nil
		},
	})

	// Phase E: Connect PyramidSurface to WGPU renderer (if available).
	runOpts := []app.Option{
		app.WithTheme(theme.Default),
		app.WithTitle("Lux Kitchen Sink"),
		app.WithSize(900, 700),
		// Phase 1: Keyboard Shortcuts (RFC-002 §2.5)
		app.WithShortcut(input.Shortcut{Key: input.KeyI, Modifiers: input.ModCtrl}, "incr"),
		app.WithShortcut(input.Shortcut{Key: input.KeyD, Modifiers: input.ModCtrl}, "decr"),
		// Phase 1: Global Handler Layer (RFC-002 §2.8)
		app.WithGlobalHandler(keyLogger),
		// Phase 2: State Persistence (RFC §3.4)
		persistence,
		// Image store for GPU texture sync
		app.WithImageStore(initial.ImageStore),
	}
	if rf := pyramidRendererFactory(initial.Pyramid); rf != nil {
		runOpts = append(runOpts, app.WithRenderer(rf))
	}
	if err := app.RunMultiViewWithCmd(initial, update, multiView, runOpts...); err != nil {
		log.Fatal(err)
	}
}

// multiView returns element trees for all active windows.
// The main window always gets the full kitchen-sink view.
// Secondary windows get their own lightweight content.
func multiView(m Model) map[app.WindowID]ui.Element {
	views := map[app.WindowID]ui.Element{
		app.MainWindow: view(m),
	}
	if m.SecondWindowOpen {
		views[1] = secondWindowView(m)
	}
	return views
}

// secondWindowView renders the content for the second window.
func secondWindowView(m Model) ui.Element {
	return layout.Pad(ui.UniformInsets(16),
		layout.Column(
			display.Text("Lux — Second Window"),
			display.Spacer(8),
			display.Text(fmt.Sprintf("Counter: %d", m.Count)),
			display.Spacer(8),
			button.Text("Close This Window", func() {
				app.Send(app.CloseWindowMsg{ID: 1})
			}),
		),
	)
}

// ── SVG Rendering Section ─────────────────────────────────────────

// Demo SVG content.
const svgStar = `<svg xmlns="http://www.w3.org/2000/svg" width="120" height="120" viewBox="0 0 120 120">
  <polygon points="60,5 73,40 110,40 80,62 90,97 60,78 30,97 40,62 10,40 47,40" fill="#f59e0b" stroke="#d97706" stroke-width="2"/>
</svg>`

const svgCircles = `<svg xmlns="http://www.w3.org/2000/svg" width="200" height="100" viewBox="0 0 200 100">
  <circle cx="40" cy="50" r="30" fill="#3b82f6"/>
  <circle cx="100" cy="50" r="30" fill="#ef4444"/>
  <circle cx="160" cy="50" r="30" fill="#22c55e"/>
  <rect x="5" y="5" width="190" height="90" fill="none" stroke="#94a3b8" stroke-width="1"/>
</svg>`

func loadDemoSVG(store *luximage.Store, svgData string, w, h int) draw.ImageID {
	svgID, err := store.LoadSVG([]byte(svgData))
	if err != nil {
		return 0
	}
	rasterID, err := store.RasterizeSVG(svgID, w, h)
	if err != nil {
		return 0
	}
	return rasterID
}

// svgPathElement is a custom element that renders a draw.Path using FillPath/StrokePath.
type svgPathElement struct {
	ui.BaseElement
	path      draw.Path
	fillColor draw.Color
	hasFill   bool
	stroke    draw.Stroke
	hasStroke bool
	width     float32
	height    float32
}

func (n svgPathElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	w := int(n.width)
	h := int(n.height)
	if w > ctx.Area.W {
		w = ctx.Area.W
	}
	if h > ctx.Area.H {
		h = ctx.Area.H
	}

	// Offset the path to the layout position.
	ox := float32(ctx.Area.X)
	oy := float32(ctx.Area.Y)
	ctx.Canvas.PushOffset(ox, oy)

	if n.hasFill {
		ctx.Canvas.FillPath(n.path, draw.SolidPaint(n.fillColor))
	}
	if n.hasStroke {
		ctx.Canvas.StrokePath(n.path, n.stroke)
	}

	ctx.Canvas.PopTransform()
	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: w, H: h, Baseline: h}
}

func (n svgPathElement) TreeEqual(other ui.Element) bool {
	_, ok := other.(svgPathElement)
	return ok
}

func (n svgPathElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

func (n svgPathElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {}

func svgRenderingSection(m Model) ui.Element {
	// Build demo paths for FillPath/StrokePath showcase.
	// 1. A filled triangle.
	triangle := draw.NewPath().
		MoveTo(draw.Pt(10, 70)).
		LineTo(draw.Pt(60, 5)).
		LineTo(draw.Pt(110, 70)).
		Close().Build()

	// 2. A stroked diamond.
	diamond := draw.NewPath().
		MoveTo(draw.Pt(50, 5)).
		LineTo(draw.Pt(95, 40)).
		LineTo(draw.Pt(50, 75)).
		LineTo(draw.Pt(5, 40)).
		Close().Build()

	// 3. A curved path (quadratic Bezier wave).
	wave := draw.NewPath().
		MoveTo(draw.Pt(0, 30)).
		QuadTo(draw.Pt(30, 0), draw.Pt(60, 30)).
		QuadTo(draw.Pt(90, 60), draw.Pt(120, 30)).
		QuadTo(draw.Pt(150, 0), draw.Pt(180, 30)).
		Build()

	// 4. A filled star using the path builder.
	// Built as a 10-vertex concave polygon alternating between outer and
	// inner radius so the polygon is non-self-intersecting (ear-clipping
	// cannot triangulate self-intersecting pentagrams).
	const outerR = 45.0
	innerR := outerR * math.Cos(2*math.Pi/5) / math.Cos(math.Pi/5)
	star := draw.NewPath()
	for i := 0; i < 10; i++ {
		a := float64(i)*math.Pi/5 - math.Pi/2
		r := outerR
		if i%2 == 1 {
			r = innerR
		}
		px := float32(50 + r*math.Cos(a))
		py := float32(50 + r*math.Sin(a))
		if i == 0 {
			star.MoveTo(draw.Pt(px, py))
		} else {
			star.LineTo(draw.Pt(px, py))
		}
	}
	starPath := star.Close().Build()

	return layout.Column(
		sectionHeader("SVG Rendering"),
		display.Text("GPU-accelerated path rendering via CPU tessellation \u2192 triangle pipeline."),
		display.Text("SVG loading and rasterization via golang.org/x/image/vector."),
		display.Spacer(16),

		// ── FillPath / StrokePath demos ──
		display.TextStyled("FillPath / StrokePath (GPU triangles)", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
		display.Spacer(4),
		layout.Row(
			layout.Column(
				display.Text("Filled triangle"),
				svgPathElement{
					path: triangle, fillColor: draw.Hex("#3b82f6"), hasFill: true,
					width: 120, height: 80,
				},
			),
			display.Spacer(16),
			layout.Column(
				display.Text("Stroked diamond"),
				svgPathElement{
					path: diamond, hasStroke: true,
					stroke: draw.Stroke{
						Paint: draw.SolidPaint(draw.Hex("#ef4444")),
						Width: 3,
					},
					width: 100, height: 80,
				},
			),
			display.Spacer(16),
			layout.Column(
				display.Text("Filled star"),
				svgPathElement{
					path: starPath, fillColor: draw.Hex("#f59e0b"), hasFill: true,
					width: 100, height: 100,
				},
			),
		),
		display.Spacer(12),
		layout.Row(
			layout.Column(
				display.Text("Bezier wave (stroked)"),
				svgPathElement{
					path: wave, hasStroke: true,
					stroke: draw.Stroke{
						Paint: draw.SolidPaint(draw.Hex("#8b5cf6")),
						Width: 2,
						Cap:   draw.StrokeCapRound,
					},
					width: 190, height: 65,
				},
			),
		),

		display.Spacer(24),

		// ── SVG rasterization demos ──
		display.TextStyled("SVG Rasterization (LoadSVG \u2192 RasterizeSVG \u2192 DrawImage)", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
		display.Spacer(4),
		display.Text("SVGs are parsed from XML, rasterized to bitmaps at target resolution, then rendered as images."),
		display.Spacer(8),
		layout.Row(
			layout.Column(
				display.Text("Star (polygon)"),
				display.Image(m.SvgStarID, display.WithImageSize(120, 120)),
			),
			display.Spacer(16),
			layout.Column(
				display.Text("Circles + rect"),
				display.Image(m.SvgCirclesID, display.WithImageSize(200, 100)),
			),
		),

		display.Spacer(24),
		display.TextStyled("Supported SVG Elements", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
		display.Spacer(4),
		display.Text("  <path d=\"...\"> \u2014 full SVG path command set (M/L/H/V/C/S/Q/T/A/Z)"),
		display.Text("  <rect>, <circle>, <ellipse>, <line>, <polygon>, <polyline>"),
		display.Text("  <g> groups with fill/stroke inheritance"),
		display.Text("  fill, stroke, stroke-width attributes"),
		display.Text("  viewBox scaling to target resolution"),
	)
}
