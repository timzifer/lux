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
	"time"

	"github.com/timzifer/lux/anim"
	"github.com/timzifer/lux/app"
	"github.com/timzifer/lux/dialog"
	"github.com/timzifer/lux/draw"
	luximage "github.com/timzifer/lux/image"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/platform"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/icons"
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
	"group-text",
	"group-images",
	"group-animation",
	"group-theming",
	"group-i18n",
	"group-platform",
	"group-rendering",
	"group-architecture",
}

// sectionGroupChildren maps each group to its leaf section IDs.
var sectionGroupChildren = map[string][]string{
	"group-basics":       {"typography", "buttons"},
	"group-input":        {"form-controls", "range-progress", "selection"},
	"group-layout":       {"layout", "split-view", "custom-layout"},
	"group-data":         {"virtual-list", "tree", "cards", "tabs", "accordion", "badges-chips"},
	"group-navigation":   {"menus", "shortcuts"},
	"group-overlays":     {"overlays", "dialogs"},
	"group-text":         {"rich-text", "text-shaping"},
	"group-images":       {"images", "shader-effects"},
	"group-animation":    {"spring-anim", "cubic-bezier", "motion-spec", "animation-id", "anim-group-seq"},
	"group-theming":      {"scoped-themes", "gradients", "effects", "blur"},
	"group-i18n":         {"rtl-layout", "locale", "ime-compose"},
	"group-platform":     {"platform-info", "window-controls", "clipboard", "gpu-backend", "multi-window"},
	"group-rendering":    {"canvas-paints", "surfaces"},
	"group-architecture": {"commands", "sub-models"},
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
	case "group-text":
		return "Text & Content"
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
	case "form-controls":
		return "Form Controls"
	case "range-progress":
		return "Range & Progress"
	case "selection":
		return "Selection"
	case "layout":
		return "Layout"
	case "split-view":
		return "SplitView"
	case "rich-text":
		return "RichText"
	case "virtual-list":
		return "VirtualList"
	case "tree":
		return "Tree"
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
	case "overlays":
		return "Overlays"
	case "canvas-paints":
		return "Canvas & Paints"
	case "scoped-themes":
		return "Scoped Themes"
	case "text-shaping":
		return "Text Shaping"
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
	case "custom-layout":
		return "Custom Layout"
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
	TextValue      string
	Scroll         *ui.ScrollState
	AnimTime       float64
	NavTree        *ui.TreeState
	ActiveSection  string
	ToggleAnim     *ui.ToggleState
	VListScroll    *ui.ScrollState
	DemoTree       *ui.TreeState
	TabIndex       int
	AccordionState *ui.AccordionState
	ChipASelected  bool
	ChipBSelected  bool
	ChipCSelected  bool
	ChipDismissed  bool
	LastMenuAction string
	MenuBarState   *ui.MenuBarState
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
	nav := ui.Tree(ui.TreeConfig{
		RootIDs:  sectionIDs,
		Children: sectionChildren,
		BuildNode: func(id string, depth int, _, selected bool) ui.Element {
			label := sectionLabel(id)
			if depth == 0 {
				// Group nodes rendered bold
				return ui.TextStyled(label, draw.TextStyle{
					Size:   13,
					Weight: draw.FontWeightSemiBold,
				})
			}
			return ui.Text(label)
		},
		NodeHeight: 28,
		MaxHeight:  0,
		State:      m.NavTree,
		OnSelect:   func(id string) { app.Send(SelectSectionMsg{id}) },
	})

	// Right panel: active section content (maxHeight 0 = fill available space)
	content := ui.ScrollView(sectionContent(m), 0, m.Scroll)

	return ui.Padding(ui.UniformInsets(16), ui.Flex(
		[]ui.Element{
			// SplitView: nav on the left, content on the right — Expanded fills remaining height
			ui.Expanded(ui.SplitView(
				nav,
				content,
				m.NavSplitRatio,
				func(r float32) { app.Send(SetNavSplitMsg{r}) },
			)),
			// Footer
			ui.Spacer(12),
			ui.Row(
				ui.ButtonText(themeLabel, func() { app.Send(ToggleThemeMsg{}) }),
			),
		},
		ui.WithDirection(ui.FlexColumn),
	))
}

func sectionContent(m Model) ui.Element {
	switch m.ActiveSection {
	case "typography":
		return typographySection()
	case "buttons":
		return buttonsSection(m)
	case "form-controls":
		return formControlsSection(m)
	case "range-progress":
		return rangeProgressSection(m)
	case "selection":
		return selectionSection(m)
	case "layout":
		return layoutSection()
	case "split-view":
		return splitViewSection(m)
	case "rich-text":
		return richTextSection()
	case "virtual-list":
		return virtualListSection(m)
	case "tree":
		return treeSection(m)
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
	case "overlays":
		return overlaysSection(m)
	case "canvas-paints":
		return canvasPaintsSection()
	case "scoped-themes":
		return scopedThemesSection()
	case "text-shaping":
		return textShapingSection()
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
	default:
		// Group nodes show a hint to expand
		if children := sectionGroupChildren[m.ActiveSection]; len(children) > 0 {
			items := make([]ui.Element, 0, len(children)+2)
			items = append(items, sectionHeader(sectionLabel(m.ActiveSection)))
			items = append(items, ui.Text("Expand this group in the tree to see:"))
			items = append(items, ui.Spacer(8))
			for _, child := range children {
				items = append(items, ui.Text("  "+sectionLabel(child)))
			}
			return ui.Column(items...)
		}
		return ui.Column(
			ui.Spacer(24),
			ui.Text("Select a section from the tree on the left."),
		)
	}
}

// ── Section Views ────────────────────────────────────────────────

func sectionHeader(title string) ui.Element {
	return ui.Column(
		ui.Spacer(8),
		ui.TextStyled(title, draw.TextStyle{
			Size:   16,
			Weight: draw.FontWeightSemiBold,
		}),
		ui.Spacer(4),
	)
}

func typographySection() ui.Element {
	return ui.Column(
		sectionHeader("Typography"),
		ui.TextStyled("Heading 1 (H1)", theme.Default.Tokens().Typography.H1),
		ui.TextStyled("Heading 2 (H2)", theme.Default.Tokens().Typography.H2),
		ui.TextStyled("Heading 3 (H3)", theme.Default.Tokens().Typography.H3),
		ui.Text("Body text — the quick brown fox jumps over the lazy dog."),
		ui.TextStyled("Body Small — metadata and captions", theme.Default.Tokens().Typography.BodySmall),
	)
}

func buttonsSection(m Model) ui.Element {
	noop := func() {}
	return ui.Column(
		sectionHeader("Buttons & Icons"),

		// Counter
		ui.Text(fmt.Sprintf("Counter: %d", m.Count)),
		ui.Row(
			ui.ButtonText("-", func() { app.Send(DecrMsg{}) }),
			ui.ButtonText("+", func() { app.Send(IncrMsg{}) }),
		),

		// Filled Buttons
		ui.Spacer(8),
		ui.Text("Filled (default):"),
		ui.Row(
			ui.ButtonText("Action", noop),
			ui.ButtonText("Save", noop),
			ui.Button(ui.Row(ui.Icon(icons.Download), ui.Text("Download")), noop),
		),

		// Outlined Buttons
		ui.Spacer(8),
		ui.Text("Outlined:"),
		ui.Row(
			ui.ButtonOutlinedText("Cancel", noop),
			ui.ButtonOutlinedText("Details", noop),
			ui.ButtonVariantOf(ui.ButtonOutlined, ui.Row(ui.Icon(icons.Share), ui.Text("Share")), noop),
		),

		// Text (Ghost) Buttons
		ui.Spacer(8),
		ui.Text("Text (chromeless):"),
		ui.Row(
			ui.ButtonGhostText("Learn more", noop),
			ui.ButtonGhostText("Skip", noop),
			ui.ButtonVariantOf(ui.ButtonGhost, ui.Row(ui.Icon(icons.ArrowRight), ui.Text("Next")), noop),
		),

		// Tonal Buttons
		ui.Spacer(8),
		ui.Text("Tonal:"),
		ui.Row(
			ui.ButtonTonalText("Draft", noop),
			ui.ButtonTonalText("Archive", noop),
			ui.ButtonVariantOf(ui.ButtonTonal, ui.Row(ui.Icon(icons.Copy), ui.Text("Duplicate")), noop),
		),

		// Icon Buttons
		ui.Spacer(8),
		ui.Text("Icon Buttons:"),
		ui.Row(
			ui.IconButton(icons.Heart, noop),
			ui.IconButton(icons.Star, noop),
			ui.IconButton(icons.Trash, noop),
			ui.IconButtonVariant(ui.ButtonOutlined, icons.Pencil, noop),
			ui.IconButtonVariant(ui.ButtonOutlined, icons.Share, noop),
			ui.IconButtonVariant(ui.ButtonGhost, icons.DotsThreeVertical, noop),
			ui.IconButtonVariant(ui.ButtonTonal, icons.Play, noop),
		),

		// Split Button
		ui.Spacer(8),
		ui.Text("Split Button:"),
		ui.Row(
			ui.SplitButton("Merge", noop, noop, []ui.SplitButtonItem{
				{Label: "Merge commit", OnClick: noop},
				{Label: "Squash and merge", OnClick: noop},
				{Label: "Rebase and merge", OnClick: noop},
			}),
		),

		// Segmented Buttons
		ui.Spacer(8),
		ui.Text("Segmented Buttons:"),
		ui.SegmentedButtons([]ui.SegmentedItem{
			{Label: "Day", OnClick: noop},
			{Label: "Week", OnClick: noop},
			{Label: "Month", OnClick: noop},
			{Label: "Year", OnClick: noop},
		}, 1),
		ui.Spacer(4),
		ui.SegmentedButtons([]ui.SegmentedItem{
			{Icon: icons.SortAscending, Label: "Sort", OnClick: noop},
			{Icon: icons.FunnelSimple, Label: "Filter", OnClick: noop},
			{Icon: icons.MagnifyingGlass, Label: "Search", OnClick: noop},
		}, 0),
		ui.Spacer(4),
		ui.SegmentedButtons([]ui.SegmentedItem{
			{Icon: icons.Play, OnClick: noop},
			{Icon: icons.Pause, OnClick: noop},
		}, 0),

		// Icons
		ui.Spacer(8),
		ui.Text("Icons (Phosphor):"),
		ui.Row(
			ui.Icon(icons.Star),
			ui.Icon(icons.ArrowRight),
			ui.Icon(icons.Heart),
			ui.Icon(icons.Gear),
			ui.Icon(icons.Eye),
			ui.Icon(icons.Sun),
			ui.Icon(icons.Moon),
		),
		ui.Row(
			ui.Icon(icons.Download),
			ui.Icon(icons.Upload),
			ui.Icon(icons.Share),
			ui.Icon(icons.Copy),
			ui.Icon(icons.Link),
			ui.Icon(icons.Play),
			ui.Icon(icons.Pause),
		),
	)
}

func formControlsSection(m Model) ui.Element {
	return ui.Column(
		sectionHeader("Form Controls"),
		ui.TextField(m.TextValue, "Enter text...",
			ui.WithOnChange(func(v string) { app.Send(SetTextMsg{v}) }),
			ui.WithFocus(app.Focus()),
		),
		ui.Spacer(8),
		ui.Checkbox("Enable notifications", m.CheckA, func(v bool) { app.Send(SetCheckAMsg{v}) }),
		ui.Checkbox("Auto-save", m.CheckB, func(v bool) { app.Send(SetCheckBMsg{v}) }),
		ui.Spacer(8),
		ui.Radio("Alpha", m.RadioChoice == "alpha", func() { app.Send(SetRadioMsg{"alpha"}) }),
		ui.Radio("Beta", m.RadioChoice == "beta", func() { app.Send(SetRadioMsg{"beta"}) }),
		ui.Radio("Gamma", m.RadioChoice == "gamma", func() { app.Send(SetRadioMsg{"gamma"}) }),
		ui.Spacer(8),
		ui.Row(
			ui.Text("Dark mode:"),
			ui.Toggle(m.ToggleOn, func(v bool) { app.Send(SetToggleMsg{v}) }, m.ToggleAnim),
		),
	)
}

func rangeProgressSection(m Model) ui.Element {
	return ui.Column(
		sectionHeader("Range & Progress"),
		ui.Text(fmt.Sprintf("Slider value: %.0f%%", m.SliderVal*100)),
		ui.Slider(m.SliderVal, func(v float32) { app.Send(SetSliderMsg{v}) }),
		ui.Spacer(8),
		ui.Text(fmt.Sprintf("Progress: %.0f%%", m.Progress*100)),
		ui.ProgressBar(m.Progress),
		ui.Spacer(4),
		ui.Text("Indeterminate:"),
		ui.ProgressBarIndeterminate(float32(math.Mod(m.AnimTime*0.8, 1.0))),
	)
}

func selectionSection(m Model) ui.Element {
	return ui.Column(
		sectionHeader("Selection"),
		ui.Select(m.SelectVal, []string{"Option 1", "Option 2", "Option 3"}),
	)
}

func layoutSection() ui.Element {
	return ui.Column(
		sectionHeader("Layout"),

		// Row
		ui.Text("Row:"),
		ui.Row(ui.Text("A"), ui.Text("B"), ui.Text("C")),
		ui.Spacer(8),

		// Stack
		ui.Text("Stack (overlapping):"),
		ui.Stack(ui.Text("Bottom"), ui.Text("Top")),
		ui.Spacer(8),

		// Flex with Justify
		ui.Text("Flex (JustifySpaceBetween):"),
		ui.Flex([]ui.Element{
			ui.Text("Left"),
			ui.Text("Center"),
			ui.Text("Right"),
		}, ui.WithJustify(ui.JustifySpaceBetween)),
		ui.Spacer(8),

		// Flex with Expanded
		ui.Text("Flex with Expanded:"),
		ui.Flex([]ui.Element{
			ui.ButtonText("Fixed", nil),
			ui.Expanded(ui.Text("← takes remaining space →")),
			ui.ButtonText("Fixed", nil),
		}),
		ui.Spacer(8),

		// Grid
		ui.Text("Grid (3 columns):"),
		ui.Grid(3, []ui.Element{
			ui.Text("Cell 1"), ui.Text("Cell 2"), ui.Text("Cell 3"),
			ui.Text("Cell 4"), ui.Text("Cell 5"), ui.Text("Cell 6"),
		}, ui.WithColGap(12), ui.WithRowGap(8)),
		ui.Spacer(8),

		// Padding
		ui.Text("Padding (16dp):"),
		ui.Padding(ui.UniformInsets(16), ui.Text("Padded content")),
		ui.Spacer(8),

		// SizedBox
		ui.Text("SizedBox (100x50):"),
		ui.SizedBox(100, 50, ui.Text("Sized")),
	)
}

func splitViewSection(m Model) ui.Element {
	return ui.Column(
		sectionHeader("SplitView"),
		ui.Text("Draggable split panes — resize by dragging the divider."),

		// 1. Horizontal (side-by-side)
		ui.Spacer(12),
		ui.Text("Horizontal split (default):"),
		ui.Spacer(4),
		ui.SizedBox(0, 120, ui.SplitView(
			ui.Padding(ui.UniformInsets(8), ui.Column(
				ui.TextStyled("Left Pane", draw.TextStyle{Size: 14, Weight: draw.FontWeightSemiBold}),
				ui.Spacer(4),
				ui.Text("This panel resizes"),
				ui.Text("when you drag the divider."),
			)),
			ui.Padding(ui.UniformInsets(8), ui.Column(
				ui.TextStyled("Right Pane", draw.TextStyle{Size: 14, Weight: draw.FontWeightSemiBold}),
				ui.Spacer(4),
				ui.Text("Content is clipped"),
				ui.Text("at the pane boundary."),
			)),
			m.SplitHorizontal,
			func(r float32) { app.Send(SetSplitHorizontalMsg{r}) },
		)),

		// 2. Vertical (stacked)
		ui.Spacer(12),
		ui.Text("Vertical split (stacked, WithSplitAxis):"),
		ui.Spacer(4),
		ui.SizedBox(0, 160, ui.SplitView(
			ui.Padding(ui.UniformInsets(8), ui.Column(
				ui.TextStyled("Top", draw.TextStyle{Size: 14, Weight: draw.FontWeightSemiBold}),
				ui.Spacer(4),
				ui.Text("Vertical divider splits top/bottom."),
			)),
			ui.Padding(ui.UniformInsets(8), ui.Column(
				ui.TextStyled("Bottom", draw.TextStyle{Size: 14, Weight: draw.FontWeightSemiBold}),
				ui.Spacer(4),
				ui.Text("Drag the horizontal bar to resize."),
			)),
			m.SplitVertical,
			func(r float32) { app.Send(SetSplitVerticalMsg{r}) },
			ui.WithSplitAxis(ui.AxisColumn),
		)),

		// 3. Nested splits (editor-like layout)
		ui.Spacer(12),
		ui.Text("Nested splits (IDE-style layout):"),
		ui.Spacer(4),
		ui.SizedBox(0, 180, ui.SplitView(
			ui.Padding(ui.UniformInsets(8), ui.Column(
				ui.Row(ui.Icon(icons.Folder), ui.Text(" Explorer")),
				ui.Spacer(4),
				ui.Text("  src/"),
				ui.Text("  docs/"),
				ui.Text("  tests/"),
			)),
			ui.SplitView(
				ui.Padding(ui.UniformInsets(8), ui.Column(
					ui.Row(ui.Icon(icons.FileText), ui.Text(" main.go")),
					ui.Spacer(4),
					ui.Text("  func main() {"),
					ui.Text("    // ..."),
					ui.Text("  }"),
				)),
				ui.Padding(ui.UniformInsets(8), ui.Column(
					ui.Row(ui.Icon(icons.Play), ui.Text(" Terminal")),
					ui.Spacer(4),
					ui.Text("  $ go run ."),
				)),
				m.SplitNested2,
				func(r float32) { app.Send(SetSplitNested2Msg{r}) },
				ui.WithSplitAxis(ui.AxisColumn),
			),
			m.SplitNested1,
			func(r float32) { app.Send(SetSplitNested1Msg{r}) },
		)),

		// 4. Three-column layout
		ui.Spacer(12),
		ui.Text("Three columns (nested horizontal):"),
		ui.Spacer(4),
		ui.SizedBox(0, 120, ui.SplitView(
			ui.Padding(ui.UniformInsets(8), ui.Column(
				ui.TextStyled("Nav", draw.TextStyle{Size: 14, Weight: draw.FontWeightSemiBold}),
				ui.Text("Home"),
				ui.Text("Settings"),
			)),
			ui.SplitView(
				ui.Padding(ui.UniformInsets(8), ui.Column(
					ui.TextStyled("Content", draw.TextStyle{Size: 14, Weight: draw.FontWeightSemiBold}),
					ui.Text("Main area"),
				)),
				ui.Padding(ui.UniformInsets(8), ui.Column(
					ui.TextStyled("Details", draw.TextStyle{Size: 14, Weight: draw.FontWeightSemiBold}),
					ui.Text("Inspector"),
				)),
				m.SplitThreeColRight,
				func(r float32) { app.Send(SetSplitThreeColRightMsg{r}) },
			),
			m.SplitThreeColLeft,
			func(r float32) { app.Send(SetSplitThreeColLeftMsg{r}) },
		)),

		// 5. Fixed (non-draggable)
		ui.Spacer(12),
		ui.Text("Fixed split (no drag — nil onResize):"),
		ui.Spacer(4),
		ui.SizedBox(0, 80, ui.SplitView(
			ui.Padding(ui.UniformInsets(8), ui.Text("Fixed left (30%)")),
			ui.Padding(ui.UniformInsets(8), ui.Text("Fixed right (70%)")),
			0.3,
			nil,
		)),
	)
}

func richTextSection() ui.Element {
	return ui.Column(
		sectionHeader("RichText"),
		ui.Text("Mixed styles in a single line:"),
		ui.Spacer(4),
		ui.RichTextSpans(
			ui.Span{Text: "Bold text ", Style: ui.SpanStyle{Style: draw.TextStyle{Weight: draw.FontWeightBold, Size: 13}}},
			ui.Span{Text: "and normal text "},
			ui.Span{Text: "with color", Style: ui.SpanStyle{Color: draw.Hex("#3b82f6")}},
		),
		ui.Spacer(12),
		ui.Text("Multiple paragraphs:"),
		ui.Spacer(4),
		ui.RichText(
			ui.RichParagraph{Spans: []ui.Span{
				{Text: "First paragraph with "},
				{Text: "bold", Style: ui.SpanStyle{Style: draw.TextStyle{Weight: draw.FontWeightBold, Size: 13}}},
				{Text: " and "},
				{Text: "colored", Style: ui.SpanStyle{Color: draw.Hex("#ef4444")}},
				{Text: " spans."},
			}},
			ui.RichParagraph{Spans: []ui.Span{
				{Text: "Second paragraph. Rich text supports per-span styling."},
			}},
		),
	)
}

func virtualListSection(m Model) ui.Element {
	return ui.Column(
		sectionHeader("VirtualList"),
		ui.Text("1000 items — only visible items are rendered:"),
		ui.Spacer(8),
		ui.VirtualList(ui.VirtualListConfig{
			ItemCount:  1000,
			ItemHeight: 24,
			BuildItem: func(i int) ui.Element {
				return ui.Text(fmt.Sprintf("  Item %d — virtualized row", i))
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
	return ui.Column(
		sectionHeader("Tree"),
		ui.Text("Hierarchical tree with expand/collapse:"),
		ui.Spacer(8),
		ui.Tree(ui.TreeConfig{
			RootIDs:  demoTreeRoots,
			Children: demoTreeChildren,
			BuildNode: func(id string, _ int, expanded, _ bool) ui.Element {
				kids := demoTreeChildren(id)
				if len(kids) > 0 {
					icon := icons.Folder
					if expanded {
						icon = icons.FolderOpen
					}
					return ui.Row(ui.Icon(icon), ui.Text(id))
				}
				return ui.Row(ui.Icon(icons.FileText), ui.Text(id))
			},
			NodeHeight: 24,
			MaxHeight:  200,
			State:      m.DemoTree,
		}),
	)
}

// ── Tier 3 Section Views ─────────────────────────────────────────

func cardsSection() ui.Element {
	return ui.Column(
		sectionHeader("Cards"),
		ui.Text("Card with text content:"),
		ui.Spacer(4),
		ui.Card(
			ui.Text("This content lives inside a Card."),
			ui.Text("Cards have elevation and borders."),
		),
		ui.Spacer(12),
		ui.Text("Nested cards:"),
		ui.Spacer(4),
		ui.Card(
			ui.Text("Outer card"),
			ui.Spacer(8),
			ui.Card(ui.Text("Inner nested card")),
		),
	)
}

func tabsSection(m Model) ui.Element {
	return ui.Column(
		sectionHeader("Tabs"),
		ui.Text("Tabs with rich headers (Icon + Text + Badge):"),
		ui.Spacer(4),
		ui.Tabs([]ui.TabItem{
			{
				Header:  ui.Row(ui.Icon(icons.Star), ui.Text("General")),
				Content: ui.Text("General settings content goes here."),
			},
			{
				Header:  ui.Row(ui.Icon(icons.Gear), ui.Text("Advanced"), ui.BadgeText("3")),
				Content: ui.Column(ui.Text("Advanced settings."), ui.Text("With multiple items.")),
			},
			{
				Header:  ui.Row(ui.Icon(icons.Eye), ui.Text("Preview")),
				Content: ui.Card(ui.Text("Preview content inside a Card.")),
			},
		}, m.TabIndex, func(idx int) { app.Send(SetTabMsg{idx}) }),
	)
}

func accordionSection(m Model) ui.Element {
	return ui.Column(
		sectionHeader("Accordion"),
		ui.Text("Collapsible sections (click to expand/collapse):"),
		ui.Spacer(4),
		ui.Accordion([]ui.AccordionSection{
			{
				Header:  ui.Text("Section 1 — Getting Started"),
				Content: ui.Text("Welcome! This section covers the basics."),
			},
			{
				Header:  ui.Text("Section 2 — Configuration"),
				Content: ui.Column(ui.Text("Configure your settings here."), ui.Text("Multiple widgets supported.")),
			},
			{
				Header:  ui.Text("Section 3 — Advanced Topics"),
				Content: ui.Card(ui.Text("Advanced content inside a Card.")),
			},
		}, m.AccordionState),
	)
}

func badgesChipsSection(m Model) ui.Element {
	tokens := theme.Default.Tokens()
	children := []ui.Element{
		sectionHeader("Badges & Chips"),

		ui.Text("Badges (colorful pill indicators):"),
		ui.Spacer(4),
		ui.Row(
			ui.BadgeText("3"),
			ui.BadgeColor(ui.Text("99+"), tokens.Colors.Status.Error),
			ui.BadgeColor(ui.Icon(icons.Star), tokens.Colors.Status.Warning),
			ui.BadgeColor(ui.Text("New"), tokens.Colors.Status.Success),
			ui.BadgeColor(ui.Row(ui.Icon(icons.Heart), ui.Text("Hot")), tokens.Colors.Accent.Secondary),
		),

		ui.Spacer(12),
		ui.Text("Chips (selectable):"),
		ui.Spacer(4),
		ui.Row(
			ui.Chip(ui.Text("Go"), m.ChipASelected, func() { app.Send(ToggleChipAMsg{}) }),
			ui.Chip(ui.Text("Rust"), m.ChipBSelected, func() { app.Send(ToggleChipBMsg{}) }),
			ui.Chip(ui.Text("Python"), m.ChipCSelected, func() { app.Send(ToggleChipCMsg{}) }),
		),
	}

	// Dismissible chip (shown until dismissed)
	if !m.ChipDismissed {
		children = append(children,
			ui.Spacer(8),
			ui.Text("Dismissible chip (click × to remove):"),
			ui.ChipDismissible(
				ui.Row(ui.Icon(icons.Star), ui.Text("Featured")),
				true,
				func() {},
				func() { app.Send(DismissChipMsg{}) },
			),
		)
	} else {
		children = append(children,
			ui.Spacer(8),
			ui.Text("Chip dismissed!"),
		)
	}

	children = append(children,
		ui.Spacer(12),
		ui.Text("Tooltip (hover to show):"),
		ui.Spacer(4),
		ui.Row(
			ui.Tooltip(
				ui.Text("← Hover me for tooltip"),
				ui.Text("This is a tooltip with arbitrary content!"),
			),
		),
	)

	return ui.Column(children...)
}

func menusSection(m Model) ui.Element {
	menuAction := func(action string) func() {
		return func() { app.Send(MenuActionMsg{action}) }
	}

	children := []ui.Element{
		sectionHeader("Menus"),

		ui.Text("MenuBar (click to open dropdown):"),
		ui.Spacer(4),
		ui.MenuBar([]ui.MenuItem{
			{Label: ui.Text("File"), Items: []ui.MenuItem{
				{Label: ui.Text("New"), OnClick: menuAction("File > New")},
				{Label: ui.Text("Open"), OnClick: menuAction("File > Open")},
				{Label: ui.Text("Save"), OnClick: menuAction("File > Save")},
			}},
			{Label: ui.Text("Edit"), Items: []ui.MenuItem{
				{Label: ui.Text("Undo"), OnClick: menuAction("Edit > Undo")},
				{Label: ui.Text("Redo"), OnClick: menuAction("Edit > Redo")},
				{Label: ui.Text("Cut"), OnClick: menuAction("Edit > Cut")},
				{Label: ui.Text("Copy"), OnClick: menuAction("Edit > Copy")},
				{Label: ui.Text("Paste"), OnClick: menuAction("Edit > Paste")},
			}},
			{Label: ui.Text("View"), Items: []ui.MenuItem{
				{Label: ui.Text("Zoom In"), OnClick: menuAction("View > Zoom In")},
				{Label: ui.Text("Zoom Out"), OnClick: menuAction("View > Zoom Out")},
			}},
			{Label: ui.Text("Help"), OnClick: menuAction("Help")},
		}, m.MenuBarState),
	}

	if m.LastMenuAction != "" {
		children = append(children,
			ui.Spacer(4),
			ui.Text(fmt.Sprintf("Last action: %s", m.LastMenuAction)),
		)
	}

	children = append(children,
		ui.Spacer(12),
		ui.Text("ContextMenu:"),
		ui.Spacer(4),
		ui.ContextMenu([]ui.MenuItem{
			{Label: ui.Text("Cut"), OnClick: menuAction("Cut")},
			{Label: ui.Text("Copy"), OnClick: menuAction("Copy")},
			{Label: ui.Text("Paste"), OnClick: menuAction("Paste")},
			{Label: ui.Text("Delete"), OnClick: menuAction("Delete")},
		}, true, 0, 0),
	)

	return ui.Column(children...)
}

// ── Phase 1 Sections ──────────────────────────────────────────────

func shortcutsSection(m Model) ui.Element {
	children := []ui.Element{
		sectionHeader("Keyboard Shortcuts"),
		ui.Text("Registered shortcuts:"),
		ui.Spacer(4),
		ui.Text("  Ctrl+I → Increment counter"),
		ui.Text("  Ctrl+D → Decrement counter"),
		ui.Spacer(8),
		ui.Text(fmt.Sprintf("Counter value: %d", m.Count)),
	}

	if m.ShortcutLog != "" {
		children = append(children,
			ui.Spacer(8),
			ui.Text(fmt.Sprintf("Last shortcut: %s", m.ShortcutLog)),
		)
	}

	children = append(children,
		ui.Spacer(16),
		sectionHeader("Global Handler Layer"),
		ui.Text("A global handler logs all key events before widget dispatch."),
	)
	if m.HandlerLog != "" {
		children = append(children,
			ui.Spacer(4),
			ui.Text(fmt.Sprintf("Handler log: %s", m.HandlerLog)),
		)
	}

	return ui.Column(children...)
}

func overlaysSection(m Model) ui.Element {
	children := []ui.Element{
		sectionHeader("Overlay System"),
		ui.Text("Click the button to toggle a dismissable overlay:"),
		ui.Spacer(4),
		ui.ButtonText("Toggle Overlay", func() { app.Send(ToggleOverlayMsg{}) }),
	}

	if m.OverlayOpen {
		children = append(children,
			ui.Spacer(8),
			ui.Text("Overlay is OPEN (click outside or press button to close)"),
			ui.Spacer(4),
			// The actual Overlay element rendered above normal flow.
			ui.Overlay{
				ID:          "demo-overlay",
				Anchor:      draw.R(300, 300, 100, 30),
				Placement:   ui.PlacementBelow,
				Dismissable: true,
				OnDismiss:   func() { app.Send(DismissOverlayMsg{}) },
				Content: ui.Card(ui.Column(
					ui.Text("This is an overlay!"),
					ui.Spacer(4),
					ui.Text("It renders above normal content."),
					ui.Spacer(8),
					ui.ButtonText("Close", func() { app.Send(DismissOverlayMsg{}) }),
				)),
			},
		)
	} else {
		children = append(children,
			ui.Spacer(8),
			ui.Text("Overlay is closed."),
		)
	}

	children = append(children,
		ui.Spacer(16),
		sectionHeader("Kinetic Scrolling"),
		ui.Text("KineticScroll with friction-decay physics is available."),
		ui.Text("Use trackpad for smooth kinetic scrolling or mouse wheel for discrete steps."),
	)

	return ui.Column(children...)
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

	return ui.Column(
		sectionHeader("Canvas & Paints (Phase 3)"),

		ui.Text("New Canvas API (GPU stubs — API validation):"),
		ui.Spacer(4),
		ui.Text("  PathBuilder.ArcTo — elliptical arc segments"),
		ui.Text("  PushClipRoundRect / PushClipPath — advanced clipping"),
		ui.Text("  PushBlur / PopBlur — backdrop blur effects"),
		ui.Text("  PushLayer / PopLayer — compositing layers"),
		ui.Text("  PushScale — uniform/non-uniform scaling"),
		ui.Text("  DrawTextLayout — rich text layout with alignment"),
		ui.Text("  DrawImageSlice — 9-slice image rendering"),
		ui.Text("  DrawTexture — external texture surfaces"),

		ui.Spacer(12),
		ui.Text("Paint Variants:"),
		ui.Spacer(4),
		ui.Text(fmt.Sprintf("  LinearGradientPaint: %d stops", 2)),
		ui.Text(fmt.Sprintf("  RadialGradientPaint: radius=%.0f", float64(50))),
		ui.Text("  PatternPaint: tiled image fills"),

		ui.Spacer(12),
		ui.Text("Theme-Lookup-Cache:"),
		ui.Spacer(4),
		ui.Text("  CachedTheme wraps Theme with lazy resolution"),
		ui.Text("  Auto-invalidation on SetThemeMsg / SetDarkModeMsg"),
		ui.Text("  Warm-up before first frame in app.Run"),
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
	return ui.Column(
		sectionHeader("Scoped Themes"),
		ui.Text("ui.Themed() overrides the theme for a subtree."),
		ui.Text("Buttons below inherit their accent color from scoped themes:"),
		ui.Spacer(12),

		// Default (no override)
		ui.Text("Default theme:"),
		ui.Spacer(4),
		ui.Row(
			ui.ButtonText("Save", nil),
			ui.ButtonText("Submit", nil),
		),
		ui.Spacer(12),

		// Danger theme
		ui.Text("Danger theme (red accent):"),
		ui.Spacer(4),
		ui.Themed(dangerTheme,
			ui.Row(
				ui.ButtonText("Delete", nil),
				ui.ButtonText("Reset All", nil),
			),
		),
		ui.Spacer(12),

		// Success theme
		ui.Text("Success theme (green accent):"),
		ui.Spacer(4),
		ui.Themed(successTheme,
			ui.Row(
				ui.ButtonText("Confirm", nil),
				ui.ButtonText("Approve", nil),
			),
		),
		ui.Spacer(12),

		// Warning theme
		ui.Text("Warning theme (amber accent):"),
		ui.Spacer(4),
		ui.Themed(warningTheme,
			ui.Row(
				ui.ButtonText("Proceed", nil),
				ui.ButtonText("Override", nil),
			),
		),
		ui.Spacer(12),

		// Mixed: default and themed side by side
		ui.Text("Mixed — default and danger in one row:"),
		ui.Spacer(4),
		ui.Row(
			ui.ButtonText("Normal", nil),
			ui.Themed(dangerTheme,
				ui.ButtonText("Danger", nil),
			),
			ui.ButtonText("Normal", nil),
		),
	)
}
func textShapingSection() ui.Element {
	return ui.Column(
		sectionHeader("Text Shaping (Phase 4)"),

		ui.Text("GoTextShaper — go-text/typesetting with full OpenType GSUB/GPOS:"),
		ui.Spacer(8),

		// Size comparison: MSDF vs bitmap threshold at 24px
		ui.Text("Size Rendering (MSDF >= 24px, Bitmap < 24px):"),
		ui.Spacer(4),
		ui.TextStyled("12px — bitmap rasterized, hinted", draw.TextStyle{Size: 12, Weight: draw.FontWeightRegular}),
		ui.TextStyled("18px — bitmap rasterized, hinted", draw.TextStyle{Size: 18, Weight: draw.FontWeightRegular}),
		ui.TextStyled("24px — MSDF rendered, scalable", draw.TextStyle{Size: 24, Weight: draw.FontWeightRegular}),
		ui.TextStyled("32px — MSDF rendered, scalable", draw.TextStyle{Size: 32, Weight: draw.FontWeightRegular}),

		ui.Spacer(12),
		ui.Text("Font Fallback Chain (RFC-003 §3.4):"),
		ui.Spacer(4),
		ui.Text("Latin: The quick brown fox jumps over the lazy dog"),
		ui.Text("Digits & Symbols: 0123456789 @#$%&*()[]{}"),
		ui.Text("Punctuation: .,;:!? - — ' \" ... /"),

		ui.Spacer(12),
		ui.Text("Per-Glyph Fallback:"),
		ui.Spacer(4),
		ui.Text("Primary font -> Fallback chain -> Embedded Noto Sans -> U+FFFD"),
		ui.Text("Missing glyphs are individually resolved, not entire runs."),

		ui.Spacer(12),
		ui.Text("Shaper Details:"),
		ui.Spacer(4),
		ui.Text("  Implementation: GoTextShaper (go-text/typesetting v0.3.4)"),
		ui.Text("  Shaping: HarfBuzz-compatible, pure Go"),
		ui.Text("  Scripts: Latin, Arabic, Devanagari, CJK (GSUB/GPOS)"),
		ui.Text("  Fallback: Noto Sans (embedded)"),
		ui.Text("  Rasterization: MSDF (>=24px) / Hinted Bitmap (<24px)"),
	)
}

// ── Commands Section ──────────────────────────────────────────────

func commandsSection(m Model) ui.Element {
	children := []ui.Element{
		sectionHeader("Commands (Async Side Effects)"),
		ui.Text("Commands let your update function trigger async work."),
		ui.Text("The result is sent back as a message when done."),
		ui.Spacer(12),
		ui.ButtonText("Run Async", func() { app.Send(StartAsyncMsg{}) }),
	}

	if m.AsyncLoading {
		children = append(children,
			ui.Spacer(8),
			ui.Text("Loading..."),
		)
	}
	if m.AsyncResult != "" {
		children = append(children,
			ui.Spacer(8),
			ui.Text(fmt.Sprintf("Result: %s", m.AsyncResult)),
		)
	}

	return ui.Column(children...)
}

// ── Sub-Models Section ───────────────────────────────────────────

func subModelsSection(m Model) ui.Element {
	return ui.Column(
		sectionHeader("Sub-Models (Delegated State)"),
		ui.Text("SubModel delegates messages to a child model."),
		ui.Text("The counter below is managed by a separate update function:"),
		ui.Spacer(12),
		ui.Text(fmt.Sprintf("Sub-Counter: %d", m.SubCounter)),
		ui.Row(
			ui.ButtonText("-", func() { app.Send(SubCounterDecrMsg{}) }),
			ui.ButtonText("+", func() { app.Send(SubCounterIncrMsg{}) }),
		),
	)
}

// ── Phase 2 Section Views ──────────────────────────────────────────

func springAnimSection(m Model) ui.Element {
	presetLabel := "gentle"
	if m.SpringPreset != "" {
		presetLabel = m.SpringPreset
	}

	return ui.Column(
		sectionHeader("Spring Animation (Phase 2)"),
		ui.Text("SpringAnim[T] — physics-based spring-damper system."),
		ui.Text("No fixed duration — converges asymptotically."),
		ui.Spacer(8),

		ui.Text("Select preset:"),
		ui.Row(
			ui.Radio("Gentle", presetLabel == "gentle", func() { app.Send(SetSpringPresetMsg{"gentle"}) }),
			ui.Radio("Snappy", presetLabel == "snappy", func() { app.Send(SetSpringPresetMsg{"snappy"}) }),
			ui.Radio("Bouncy", presetLabel == "bouncy", func() { app.Send(SetSpringPresetMsg{"bouncy"}) }),
		),
		ui.Spacer(8),

		ui.ButtonText("Animate Spring", func() { app.Send(StartSpringMsg{}) }),
		ui.Spacer(8),
		ui.Text(fmt.Sprintf("Value: %.3f", m.SpringVal.Value())),
		ui.ProgressBar(m.SpringVal.Value()),
		ui.Spacer(4),
		ui.Text(fmt.Sprintf("Done: %v", m.SpringVal.IsDone())),
	)
}

func cubicBezierSection(m Model) ui.Element {
	preset := m.BezierPreset
	if preset == "" {
		preset = "ease"
	}

	return ui.Column(
		sectionHeader("Cubic Bezier Easing (Phase 2)"),
		ui.Text("CubicBezier(x1, y1, x2, y2) — CSS-compatible easing."),
		ui.Spacer(8),

		ui.Text("Select CSS preset:"),
		ui.Row(
			ui.Radio("ease", preset == "ease", func() { app.Send(SetBezierPresetMsg{"ease"}) }),
			ui.Radio("ease-in", preset == "ease-in", func() { app.Send(SetBezierPresetMsg{"ease-in"}) }),
			ui.Radio("ease-out", preset == "ease-out", func() { app.Send(SetBezierPresetMsg{"ease-out"}) }),
			ui.Radio("ease-in-out", preset == "ease-in-out", func() { app.Send(SetBezierPresetMsg{"ease-in-out"}) }),
		),
		ui.Spacer(8),

		ui.ButtonText("Animate", func() { app.Send(StartBezierMsg{}) }),
		ui.Spacer(8),
		ui.Text(fmt.Sprintf("Value: %.3f (preset: %s)", m.BezierAnim.Value(), preset)),
		ui.ProgressBar(m.BezierAnim.Value()),
	)
}

func motionSpecSection(m Model) ui.Element {
	tokens := theme.Default.Tokens()
	preset := m.MotionPreset
	if preset == "" {
		preset = "standard"
	}

	return ui.Column(
		sectionHeader("Motion Spec (Phase 2)"),
		ui.Text("DurationEasing — theme tokens pair duration with easing."),
		ui.Spacer(8),
		ui.Text(fmt.Sprintf("Standard:   %v + OutCubic", tokens.Motion.Standard.Duration)),
		ui.Text(fmt.Sprintf("Emphasized: %v + InOutCubic", tokens.Motion.Emphasized.Duration)),
		ui.Text(fmt.Sprintf("Quick:      %v + OutExpo", tokens.Motion.Quick.Duration)),
		ui.Spacer(12),

		ui.Text("Select preset:"),
		ui.Row(
			ui.Radio("Standard", preset == "standard", func() { app.Send(SetMotionPresetMsg{"standard"}) }),
			ui.Radio("Emphasized", preset == "emphasized", func() { app.Send(SetMotionPresetMsg{"emphasized"}) }),
			ui.Radio("Quick", preset == "quick", func() { app.Send(SetMotionPresetMsg{"quick"}) }),
		),
		ui.Spacer(8),

		ui.ButtonText("Animate", func() { app.Send(StartMotionMsg{}) }),
		ui.Spacer(8),
		ui.Text(fmt.Sprintf("Value: %.3f", m.MotionAnim.Value())),
		ui.ProgressBar(m.MotionAnim.Value()),
	)
}

func animationIDSection(m Model) ui.Element {
	children := []ui.Element{
		sectionHeader("Animation ID (Phase 2)"),
		ui.Text("SetTargetWithID — fires AnimationEnded{ID} on completion."),
		ui.Text("The user update loop receives the message — no callbacks."),
		ui.Spacer(8),
		ui.ButtonText("Start Fade (500ms)", func() { app.Send(StartFadeMsg{}) }),
	}

	if m.FadeActive {
		children = append(children,
			ui.Spacer(8),
			ui.Text(fmt.Sprintf("Fading... opacity: %.2f", m.FadeOpacity.Value())),
			ui.ProgressBar(m.FadeOpacity.Value()),
		)
	}

	if m.AnimIDResult != "" {
		children = append(children,
			ui.Spacer(8),
			ui.Text(fmt.Sprintf("Received: %s", m.AnimIDResult)),
		)
	}

	return ui.Column(children...)
}

func animGroupSeqSection(m Model) ui.Element {
	return ui.Column(
		sectionHeader("AnimGroup & AnimSeq (Phase 2)"),
		ui.Text("AnimGroup — parallel animations. AnimSeq — sequential."),
		ui.Spacer(8),

		ui.Text("Parallel (AnimGroup):"),
		ui.Row(
			ui.ButtonText("Run Group", func() { app.Send(StartGroupMsg{}) }),
		),
		ui.Spacer(4),
		ui.Text(fmt.Sprintf("  A: %.2f  B: %.2f", m.GroupA.Value(), m.GroupB.Value())),
		ui.Spacer(8),

		ui.Text("Sequential (AnimSeq):"),
		ui.Row(
			ui.ButtonText("Run Seq", func() { app.Send(StartSeqMsg{}) }),
		),
		ui.Spacer(4),
		ui.Text(fmt.Sprintf("  A: %.2f  B: %.2f", m.SeqA.Value(), m.SeqB.Value())),
		ui.Spacer(8),

		ui.Text(fmt.Sprintf("Status: %s", m.GroupSeqStatus)),
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

	return ui.Column(
		sectionHeader("Custom Layout (Phase 2)"),
		ui.Text("Layout interface — user-defined layout algorithms."),
		ui.Text("LayoutCtx provides Measure/Place callbacks."),
		ui.Spacer(8),

		ui.Text(fmt.Sprintf("Stair gap: %.0f dp", gap)),
		ui.Slider(gap/100, func(v float32) { app.Send(SetLayoutGapMsg{v * 100}) }),
		ui.Spacer(8),

		ui.Text("Stair layout demo:"),
		ui.Spacer(4),
		ui.CustomLayout(stairLayout{Gap: gap},
			ui.ButtonText("Step 1", nil),
			ui.ButtonText("Step 2", nil),
			ui.ButtonText("Step 3", nil),
			ui.ButtonText("Step 4", nil),
		),

		ui.Spacer(16),
		ui.Text("Layout Cache:"),
		ui.Spacer(4),
		ui.Text("  LayoutCache stores constraints + size + childRects"),
		ui.Text("  Invalidated when props or constraints change"),
		ui.Text("  O(dirty subtree) re-layout, not O(n)"),
	)
}

// ── Phase 4b Section Views ────────────────────────────────────────

func rtlLayoutSection() ui.Element {
	return ui.Column(
		sectionHeader("RTL Layout (Phase 4b)"),
		ui.Text("Insets now support Start/End for direction-aware spacing."),
		ui.Spacer(8),

		// Demonstrate InlineInsets — Start=40, End=8
		ui.Text("InlineInsets(40, 8) — Start has more padding:"),
		ui.Padding(ui.InlineInsets(40, 8),
			ui.Card(ui.Column(
				ui.Text("This card uses logical Start/End insets."),
				ui.Text("In LTR: Start=Left, End=Right."),
				ui.Text("In RTL: Start=Right, End=Left."),
			)),
		),
		ui.Spacer(12),

		// Demonstrate LogicalInsets
		ui.Text("LogicalInsets(8, 40, 8, 16) — top/end/bottom/start:"),
		ui.Padding(ui.LogicalInsets(8, 40, 8, 16),
			ui.Card(ui.Text("Four-sided logical insets.")),
		),
		ui.Spacer(12),

		// FlexRow automatically mirrors in RTL
		ui.Text("FlexRow mirrors child order in RTL:"),
		ui.Spacer(4),
		ui.Flex([]ui.Element{
			ui.BadgeText("First"),
			ui.BadgeText("Second"),
			ui.BadgeText("Third"),
		}, ui.WithDirection(ui.FlexRow), ui.WithGap(8)),
		ui.Spacer(8),

		// JustifyStart resolves to left in LTR, right in RTL
		ui.Text("JustifyStart — left-aligned in LTR, right-aligned in RTL:"),
		ui.Spacer(4),
		ui.Flex([]ui.Element{
			ui.ButtonText("Start", nil),
		}, ui.WithDirection(ui.FlexRow), ui.WithJustify(ui.JustifyStart)),
		ui.Spacer(8),

		ui.Text("Switch to Arabic locale (in 'Locale' section) to see RTL mirroring."),
	)
}

func localeSection(m Model) ui.Element {
	currentLocale := m.CurrentLocale
	if currentLocale == "" {
		currentLocale = "en (default)"
	}
	return ui.Column(
		sectionHeader("Locale / i18n (Phase 4b)"),
		ui.Text("app.WithLocale() sets the BCP 47 locale at startup."),
		ui.Text("app.SetLocaleMsg switches locale at runtime."),
		ui.Spacer(8),

		ui.Text(fmt.Sprintf("Current locale: %s", currentLocale)),
		ui.Spacer(8),

		ui.Text("Switch locale:"),
		ui.Spacer(4),
		ui.Row(
			ui.ButtonText("English (LTR)", func() { app.Send(SetLocaleChoiceMsg{Locale: "en"}) }),
			ui.ButtonText("العربية (RTL)", func() { app.Send(SetLocaleChoiceMsg{Locale: "ar"}) }),
			ui.ButtonText("עברית (RTL)", func() { app.Send(SetLocaleChoiceMsg{Locale: "he"}) }),
			ui.ButtonText("Deutsch (LTR)", func() { app.Send(SetLocaleChoiceMsg{Locale: "de"}) }),
		),
		ui.Spacer(12),

		ui.Text("The layout direction is derived from the locale:"),
		ui.Text("  Arabic (ar) → RTL"),
		ui.Text("  Hebrew (he) → RTL"),
		ui.Text("  English (en), German (de) → LTR"),
		ui.Spacer(8),
		ui.Text("Switching triggers full layout invalidation."),
	)
}

func imeComposeSection(m Model) ui.Element {
	composeStatus := "No active composition"
	if m.IMEComposeText != "" {
		composeStatus = fmt.Sprintf("Composing: [%s]", m.IMEComposeText)
	}

	return ui.Column(
		sectionHeader("IME Compose (Phase 4b)"),
		ui.Text("IME composition support for CJK and other input methods."),
		ui.Spacer(8),

		ui.Text("New message types:"),
		ui.Text("  • IMEComposeMsg — pre-edit text (composition in progress)"),
		ui.Text("  • IMECommitMsg — final committed text"),
		ui.Spacer(8),

		ui.Text("Platform integration:"),
		ui.Text("  • Platform.SetIMECursorRect() — positions candidate window"),
		ui.Text("  • GLFW: awaiting 3.4 for glfwSetPreeditCallback"),
		ui.Text("  • Win32: IMM32 integration planned"),
		ui.Spacer(8),

		ui.Text("TextField composition state:"),
		ui.Text("  • InputState.ComposeText — current pre-edit string"),
		ui.Text("  • InputState.ComposeCursorStart/End — cursor range"),
		ui.Spacer(8),

		ui.Text(fmt.Sprintf("Status: %s", composeStatus)),
		ui.Spacer(8),

		ui.Text("EventDispatcher routes IME events to focused widget:"),
		ui.Text("  • EventIMECompose → focused widget via RenderCtx.Events"),
		ui.Text("  • EventIMECommit → focused widget via RenderCtx.Events"),
	)
}

// ── Phase 5 Section Views ──────────────────────────────────────────

func platformInfoSection() ui.Element {
	return ui.Column(
		sectionHeader("Platform Info (Phase 5)"),
		ui.Text("Platform-Interface erweitert (RFC §7.1):"),
		ui.Spacer(8),

		ui.Text("Neue Methoden:"),
		ui.Spacer(4),
		ui.Text("  SetSize(w, h int) — Fenstergröße ändern"),
		ui.Text("  SetFullscreen(bool) — Vollbildmodus umschalten"),
		ui.Text("  RequestFrame() — Nächsten Frame anfordern"),
		ui.Text("  SetClipboard(text) / GetClipboard() — Zwischenablage"),
		ui.Text("  CreateWGPUSurface(instance) — wgpu Surface erstellen"),
		ui.Spacer(12),

		ui.Text("Verfügbare Backends:"),
		ui.Spacer(4),
		ui.Text("  • GLFW (macOS/Linux, default) — OpenGL 3.3 Core"),
		ui.Text("  • Win32 (Windows, native) — GDI Software / wgpu"),
		ui.Text("  • Wayland (Linux, -tags wayland) — Native Wayland + wgpu"),
		ui.Text("  • X11 (Linux, -tags x11) — Native X11 + wgpu"),
		ui.Text("  • Cocoa (macOS, -tags cocoa) — Native AppKit + Metal"),
		ui.Text("  • DRM/KMS (Linux, -tags drm) — Direct framebuffer"),
		ui.Spacer(12),

		ui.Text("Backend-Auswahl via Build-Tags:"),
		ui.Spacer(4),
		ui.Text("  go build -tags wayland ./..."),
		ui.Text("  go build -tags x11 ./..."),
		ui.Text("  go build -tags cocoa ./..."),
		ui.Text("  go build -tags drm ./..."),
	)
}

func windowControlsSection(m Model) ui.Element {
	fsLabel := "Enter Fullscreen"
	if m.IsFullscreen {
		fsLabel = "Exit Fullscreen"
	}

	return ui.Column(
		sectionHeader("Window Controls (Phase 5)"),
		ui.Text("SetSize — resize the window programmatically:"),
		ui.Spacer(8),
		ui.Row(
			ui.ButtonText("800×600", func() { app.Send(ResizeWindowMsg{800, 600}) }),
			ui.ButtonText("1024×768", func() { app.Send(ResizeWindowMsg{1024, 768}) }),
			ui.ButtonText("1280×720", func() { app.Send(ResizeWindowMsg{1280, 720}) }),
		),
		ui.Spacer(12),
		ui.Text("SetFullscreen — toggle fullscreen mode:"),
		ui.Spacer(4),
		ui.ButtonText(fsLabel, func() { app.Send(ToggleFullscreenMsg{}) }),
		ui.Spacer(4),
		ui.Text(fmt.Sprintf("Fullscreen: %v", m.IsFullscreen)),
		ui.Spacer(12),
		ui.Text("RequestFrame — request immediate repaint:"),
		ui.Spacer(4),
		ui.Text("Used internally to trigger repaints outside the normal event loop."),
	)
}

func clipboardSection(m Model) ui.Element {
	return ui.Column(
		sectionHeader("Clipboard (Phase 5)"),
		ui.Text("SetClipboard / GetClipboard — system clipboard access:"),
		ui.Spacer(8),

		ui.Text("Text to copy:"),
		ui.Spacer(4),
		ui.TextField(m.ClipboardText, "Enter text to copy...",
			ui.WithOnChange(func(v string) { app.Send(SetClipboardTextMsg{v}) }),
			ui.WithFocus(app.Focus()),
		),
		ui.Spacer(8),

		ui.Row(
			ui.ButtonText("Copy to Clipboard", func() { app.Send(CopyToClipboardMsg{}) }),
			ui.ButtonText("Paste from Clipboard", func() { app.Send(PasteFromClipboardMsg{}) }),
		),
		ui.Spacer(8),
		ui.Text(fmt.Sprintf("Current text: %q", m.ClipboardText)),
		ui.Spacer(12),

		ui.Text("API:"),
		ui.Spacer(4),
		ui.Text("  app.SetClipboard(text) — set clipboard (package-level)"),
		ui.Text("  app.GetClipboard() — get clipboard (package-level)"),
		ui.Text("  platform.SetClipboard(text) — per-backend implementation"),
	)
}

func gpuBackendSection() ui.Element {
	return ui.Column(
		sectionHeader("GPU Backend (Phase 5)"),
		ui.Text("wgpu — WebGPU-basiertes GPU-Backend (RFC §6.1):"),
		ui.Spacer(8),

		ui.Text("Zwei Implementierungen:"),
		ui.Spacer(4),
		ui.Text("  1. wgpu-native (CGo, Default)"),
		ui.Text("     Wrapper für die C-Library wgpu-native"),
		ui.Text("     Backends: Vulkan (Linux), Metal (macOS), D3D12 (Windows)"),
		ui.Spacer(4),
		ui.Text("  2. gogpu (Pure Go, -tags gogpu)"),
		ui.Text("     Vollständig in Go, keine CGo-Abhängigkeit"),
		ui.Text("     Backend: Vulkan via pure-Go Bindings"),
		ui.Spacer(12),

		ui.Text("Shim-Interface (internal/wgpu/):"),
		ui.Spacer(4),
		ui.Text("  Instance, Adapter, Device, Surface, SwapChain"),
		ui.Text("  RenderPipeline, Buffer, Texture, CommandEncoder"),
		ui.Text("  RenderPass, ShaderModule, BindGroup, Queue"),
		ui.Spacer(12),

		ui.Text("wgpu-Renderer (internal/gpu/wgpu_renderer.go):"),
		ui.Spacer(4),
		ui.Text("  WGSL-Shader für:"),
		ui.Text("    • Rounded Rectangles (SDF-basiert, instanced)"),
		ui.Text("    • Atlas-basierte Bitmap-Glyphen (<24px)"),
		ui.Text("    • MSDF-Text (>=24px, Chlumsky-Methode)"),
		ui.Spacer(12),

		ui.Text("Migration von OpenGL 3.3:"),
		ui.Spacer(4),
		ui.Text("  • GLSL 330 → WGSL Shader-Konvertierung"),
		ui.Text("  • glScissor → wgpu RenderPass Clipping"),
		ui.Text("  • glDrawArraysInstanced → wgpu DrawInstanced"),
		ui.Text("  • glBufferData → wgpu Queue.WriteBuffer"),
	)
}

// ── Phase 6 Section Views ──────────────────────────────────────────

func surfacesSection(pyramid *PyramidSurface) ui.Element {
	return ui.Column(
		sectionHeader("External Surfaces (Phase 6)"),
		ui.Text("Surface Slots — RFC §8: Externe Surfaces"),
		ui.Spacer(8),

		ui.Text("SurfaceProvider Interface:"),
		ui.Spacer(4),
		ui.Text("  AcquireFrame(bounds) → (TextureID, FrameToken)"),
		ui.Text("  ReleaseFrame(token)"),
		ui.Text("  HandleMsg(msg) → consumed"),
		ui.Spacer(12),

		ui.Text("Zero-Copy Paths:"),
		ui.Spacer(4),
		ui.Text(fmt.Sprintf("  Preferred mode on this platform: %d", ui.PreferredZeroCopyMode())),
		ui.Spacer(4),
		ui.Text("  • macOS: IOSurface → wgpu Shared Texture"),
		ui.Text("  • Linux: DMA-buf → wgpu External Memory"),
		ui.Text("  • Windows: DXGI Shared Handle"),
		ui.Text("  • Fallback: OSR → CPU-Copy → Upload"),
		ui.Spacer(12),

		ui.Text("RGB Cube (drag to rotate):"),
		ui.Spacer(4),
		ui.Surface(1, pyramid, 400, 300),
		ui.Spacer(12),

		ui.Text("Input Routing:"),
		ui.Spacer(4),
		ui.Text("  Mouse/Key events in surface area → SurfaceMouseMsg/SurfaceKeyMsg"),
		ui.Text("  Routed via SurfaceProvider.HandleMsg()"),
	)
}

// ── Gradients Section (Phase E) ──────────────────────────────────

func gradientsSection() ui.Element {
	return ui.Column(
		sectionHeader("Gradients (Phase E)"),
		ui.Text("GPU-rendered gradient fills via the gradient pipeline:"),
		ui.Spacer(12),

		// Linear gradient: 2-stop blue→indigo
		ui.Text("Linear Gradient (2 stops):"),
		ui.Spacer(4),
		ui.GradientRect(200, 60, 8, draw.LinearGradientPaint(
			draw.Pt(0, 0), draw.Pt(200, 0),
			draw.GradientStop{Offset: 0, Color: draw.Hex("#3b82f6")},
			draw.GradientStop{Offset: 1, Color: draw.Hex("#6366f1")},
		)),
		ui.Spacer(12),

		// Linear gradient: 4-stop rainbow
		ui.Text("Linear Gradient (4 stops):"),
		ui.Spacer(4),
		ui.GradientRect(200, 60, 8, draw.LinearGradientPaint(
			draw.Pt(0, 0), draw.Pt(200, 0),
			draw.GradientStop{Offset: 0.0, Color: draw.Hex("#ef4444")},
			draw.GradientStop{Offset: 0.33, Color: draw.Hex("#eab308")},
			draw.GradientStop{Offset: 0.66, Color: draw.Hex("#22c55e")},
			draw.GradientStop{Offset: 1.0, Color: draw.Hex("#3b82f6")},
		)),
		ui.Spacer(12),

		// Radial gradient
		ui.Text("Radial Gradient:"),
		ui.Spacer(4),
		ui.GradientRect(200, 200, 8, draw.RadialGradientPaint(
			draw.Pt(100, 100), 100,
			draw.GradientStop{Offset: 0, Color: draw.Hex("#ffffff")},
			draw.GradientStop{Offset: 1, Color: draw.Hex("#09090b")},
		)),
		ui.Spacer(12),

		// Sharp rect (no radius)
		ui.Text("Sharp Linear Gradient (no radius):"),
		ui.Spacer(4),
		ui.GradientRect(200, 40, 0, draw.LinearGradientPaint(
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
		ui.Text("Framework-rendered dialog overlays with backdrop scrim:"),
		ui.Spacer(8),
		ui.Row(
			ui.ButtonText("Info", func() { app.Send(ShowMsgDialogMsg{Kind: platform.DialogInfo}) }),
			ui.Spacer(8),
			ui.ButtonText("Warning", func() { app.Send(ShowMsgDialogMsg{Kind: platform.DialogWarning}) }),
			ui.Spacer(8),
			ui.ButtonText("Error", func() { app.Send(ShowMsgDialogMsg{Kind: platform.DialogError}) }),
		),
		ui.Spacer(8),
		ui.Row(
			ui.ButtonText("Confirm", func() { app.Send(ShowConfirmDialogMsg{}) }),
			ui.Spacer(8),
			ui.ButtonText("Input", func() { app.Send(ShowInputDialogMsg{}) }),
			ui.Spacer(8),
			ui.ButtonOutlinedText("Native Confirm", func() { app.Send(NativeConfirmMsg{}) }),
		),
	}

	if m.DialogResult != "" {
		children = append(children,
			ui.Spacer(8),
			ui.Text(fmt.Sprintf("Result: %s", m.DialogResult)),
		)
	}

	// Render active dialog overlays.
	if m.ShowMsgDialog {
		children = append(children, ui.MessageDialog(
			"demo-msg-dialog",
			"Message",
			"This is a sample message dialog.",
			m.DialogMsgKind,
			func() { app.Send(DismissDialogMsg{}) },
		))
	}
	if m.ShowConfirmDialog {
		children = append(children, ui.ConfirmDialog(
			"demo-confirm-dialog",
			"Confirm Action",
			"Are you sure you want to proceed?",
			func() { app.Send(DialogConfirmedMsg{}) },
			func() { app.Send(DismissDialogMsg{}) },
		))
	}
	if m.ShowInputDialog {
		children = append(children, ui.InputDialog(
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

	return ui.Column(children...)
}

// ── Blur Section (Phase F) ───────────────────────────────────────

func blurSection() ui.Element {
	radii := []float32{4, 8, 16, 32, 64}
	items := make([]ui.Element, 0, len(radii)+3)
	items = append(items,
		sectionHeader("Blur (Phase F)"),
		ui.Text("Gaussian blur at various radii (PushBlur / PopBlur):"),
		ui.Spacer(8),
	)

	for _, r := range radii {
		radius := r
		items = append(items,
			ui.Spacer(4),
			ui.BlurBox(radius,
				ui.Stack(
					ui.GradientRect(200, 60, 8, draw.LinearGradientPaint(
						draw.Pt(0, 0), draw.Pt(200, 0),
						draw.GradientStop{Offset: 0, Color: draw.Hex("#3366cc")},
						draw.GradientStop{Offset: 1, Color: draw.Hex("#ffcc33")},
					)),
					ui.SizedBox(200, 60,
						ui.Padding(ui.UniformInsets(8),
							ui.Text(fmt.Sprintf("radius = %.0f", radius)),
						),
					),
				),
			),
		)
	}

	return ui.Column(items...)
}

// ── Effects Section (Phase G) ────────────────────────────────────

func effectsSection() ui.Element {
	items := []ui.Element{
		sectionHeader("Effects (Phase G)"),
	}

	// --- Shadows ---
	items = append(items,
		ui.Text("Soft Shadows (None / Low / Med / High):"),
		ui.Spacer(8),
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
		shadowCards[i] = ui.Padding(ui.UniformInsets(20),
			ui.ShadowBox(lv.shadow, 8,
				ui.Stack(
					ui.GradientRect(132, 82, 8, draw.SolidPaint(lv.bg)),
					ui.Padding(ui.UniformInsets(16),
						ui.SizedBox(100, 50,
							ui.Text(lv.label),
						),
					),
				),
			),
		)
	}
	items = append(items, ui.Row(shadowCards...))

	// --- Opacity ---
	items = append(items,
		ui.Spacer(16),
		ui.Text("Opacity (1.0, 0.75, 0.5, 0.25):"),
		ui.Spacer(8),
	)

	alphas := []float32{1.0, 0.75, 0.5, 0.25}
	opacityBoxes := make([]ui.Element, len(alphas))
	for i, a := range alphas {
		opacityBoxes[i] = ui.Padding(ui.UniformInsets(4),
			ui.OpacityBox(a,
				ui.Stack(
					ui.GradientRect(104, 64, 6, draw.SolidPaint(draw.Hex("#3b82f6"))),
					ui.Padding(ui.UniformInsets(12),
						ui.SizedBox(80, 40,
							ui.Text(fmt.Sprintf("%.0f%%", a*100)),
						),
					),
				),
			),
		)
	}
	items = append(items, ui.Row(opacityBoxes...))

	// --- Frosted Glass ---
	items = append(items,
		ui.Spacer(16),
		ui.Text("Frosted Glass (blur backdrop + sharp overlay panel):"),
		ui.Spacer(8),
	)

	items = append(items,
		ui.Stack(
			// Complex background: colorful checkerboard pattern makes blur effect obvious
			ui.CheckerRect(420, 160, 16),
			// Frosted glass panel overlaid on the pattern
			ui.Padding(draw.Insets{Top: 24, Left: 50, Right: 50, Bottom: 24},
				ui.FrostedGlass(16, draw.Color{R: 1, G: 1, B: 1, A: 0.18},
					ui.SizedBox(320, 112,
						ui.Padding(ui.UniformInsets(16),
							ui.Column(
								ui.Text("Frosted Glass Panel"),
								ui.Spacer(4),
								ui.Text("Background is blurred, text stays sharp."),
							),
						),
					),
				),
			),
		),
	)

	// --- Inner Shadow (Tier 2) ---
	items = append(items,
		ui.Spacer(16),
		ui.Text("Inner Shadow (light inset, deeper inset):"),
		ui.Spacer(8),
	)

	innerShadowCards := []ui.Element{
		ui.Padding(ui.UniformInsets(12),
			ui.InnerShadowBox(
				draw.Shadow{Color: draw.Color{R: 0, G: 0, B: 0, A: 0.5}, BlurRadius: 10},
				8,
				ui.Stack(
					ui.GradientRect(132, 82, 8, draw.SolidPaint(draw.Hex("#e2e8f0"))),
					ui.Padding(ui.UniformInsets(16),
						ui.SizedBox(100, 50, ui.Text("Light Inset")),
					),
				),
			),
		),
		ui.Padding(ui.UniformInsets(12),
			ui.InnerShadowBox(
				draw.Shadow{Color: draw.Color{R: 0, G: 0, B: 0, A: 0.85}, BlurRadius: 20},
				8,
				ui.Stack(
					ui.GradientRect(132, 82, 8, draw.SolidPaint(draw.Hex("#cbd5e1"))),
					ui.Padding(ui.UniformInsets(16),
						ui.SizedBox(100, 50, ui.Text("Deep Inset")),
					),
				),
			),
		),
	}
	items = append(items, ui.Row(innerShadowCards...))

	// --- Elevation / Hover-Responsive Shadows (Tier 2) ---
	items = append(items,
		ui.Spacer(16),
		ui.Text("Elevation (hover to see shadow lift):"),
		ui.Spacer(8),
	)

	elevationCards := []ui.Element{
		ui.Padding(ui.UniformInsets(16),
			ui.ElevationCard(nil,
				ui.Stack(
					ui.GradientRect(132, 82, 8, draw.SolidPaint(draw.Hex("#6366f1"))),
					ui.Padding(ui.UniformInsets(16),
						ui.SizedBox(100, 50, ui.Text("Card A")),
					),
				),
			),
		),
		ui.Padding(ui.UniformInsets(16),
			ui.ElevationCard(nil,
				ui.Stack(
					ui.GradientRect(132, 82, 8, draw.SolidPaint(draw.Hex("#3b82f6"))),
					ui.Padding(ui.UniformInsets(16),
						ui.SizedBox(100, 50, ui.Text("Card B")),
					),
				),
			),
		),
		ui.Padding(ui.UniformInsets(16),
			ui.ElevationCard(nil,
				ui.Stack(
					ui.GradientRect(132, 82, 8, draw.SolidPaint(draw.Hex("#0ea5e9"))),
					ui.Padding(ui.UniformInsets(16),
						ui.SizedBox(100, 50, ui.Text("Card C")),
					),
				),
			),
		),
	}
	items = append(items, ui.Row(elevationCards...))

	// --- Vibrancy / Tinted Blur (Tier 2) ---
	items = append(items,
		ui.Spacer(16),
		ui.Text("Vibrancy vs Frosted Glass (accent-tinted blur — compare the color cast):"),
		ui.Spacer(8),
	)

	items = append(items,
		ui.Row(
			// Frosted Glass reference (neutral white tint)
			ui.Padding(ui.UniformInsets(8),
				ui.Stack(
					ui.CheckerRect(210, 160, 16),
					ui.Padding(draw.Insets{Top: 20, Left: 16, Right: 16, Bottom: 20},
						ui.FrostedGlass(16, draw.Color{R: 1, G: 1, B: 1, A: 0.18},
							ui.SizedBox(178, 120,
								ui.Padding(ui.UniformInsets(12),
									ui.Column(
										ui.Text("Frosted Glass"),
										ui.Spacer(4),
										ui.Text("Neutral white tint"),
									),
								),
							),
						),
					),
				),
			),
			// Vibrancy (accent-tinted, visibly colored)
			ui.Padding(ui.UniformInsets(8),
				ui.Stack(
					ui.CheckerRect(210, 160, 16),
					ui.Padding(draw.Insets{Top: 20, Left: 16, Right: 16, Bottom: 20},
						ui.Vibrancy(0.35,
							ui.SizedBox(178, 120,
								ui.Padding(ui.UniformInsets(12),
									ui.Column(
										ui.Text("Vibrancy"),
										ui.Spacer(4),
										ui.Text("Accent-tinted blur"),
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
		ui.Spacer(16),
		ui.Text("Noise/Grain (always-on dither — zoom in to see noise preventing banding):"),
		ui.Spacer(8),
	)
	items = append(items,
		ui.Row(
			ui.Padding(ui.UniformInsets(8),
				ui.GradientRect(200, 80, 8, draw.LinearGradientPaint(
					draw.Pt(0, 0), draw.Pt(200, 0),
					draw.GradientStop{Offset: 0, Color: draw.Hex("#1e3a5f")},
					draw.GradientStop{Offset: 1, Color: draw.Hex("#4a90d9")},
				)),
			),
			ui.Padding(ui.UniformInsets(8),
				ui.GradientRect(200, 80, 8, draw.LinearGradientPaint(
					draw.Pt(0, 0), draw.Pt(0, 80),
					draw.GradientStop{Offset: 0, Color: draw.Hex("#2d1b4e")},
					draw.GradientStop{Offset: 1, Color: draw.Hex("#7c3aed")},
				)),
			),
		),
	)

	// --- Glow (Tier 3) ---
	items = append(items,
		ui.Spacer(16),
		ui.Text("Glow (soft outer glow using shadow pipeline):"),
		ui.Spacer(8),
	)
	glowCards := []ui.Element{
		ui.Padding(ui.UniformInsets(16),
			ui.Glow(12, 8,
				ui.SizedBox(120, 80,
					ui.Padding(ui.UniformInsets(12),
						ui.Column(
							ui.Text("Accent"),
							ui.Spacer(4),
							ui.Text("blur=12"),
						),
					),
				),
			),
		),
		ui.Padding(ui.UniformInsets(16),
			ui.GlowBox(draw.Color{R: 0.2, G: 0.9, B: 0.4, A: 0.6}, 16, 8,
				ui.SizedBox(120, 80,
					ui.Padding(ui.UniformInsets(12),
						ui.Column(
							ui.Text("Green"),
							ui.Spacer(4),
							ui.Text("blur=16"),
						),
					),
				),
			),
		),
		ui.Padding(ui.UniformInsets(16),
			ui.GlowBox(draw.Color{R: 0.9, G: 0.2, B: 0.2, A: 0.6}, 20, 8,
				ui.SizedBox(120, 80,
					ui.Padding(ui.UniformInsets(12),
						ui.Column(
							ui.Text("Red"),
							ui.Spacer(4),
							ui.Text("blur=20"),
						),
					),
				),
			),
		),
	}
	items = append(items, ui.Row(glowCards...))

	return ui.Column(items...)
}

// ── Multi-Window Section (Phase F) ──────────────────────────────

func multiWindowSection(m Model) ui.Element {
	var btn ui.Element
	if m.SecondWindowOpen {
		btn = ui.ButtonText("Close Second Window", func() {
			app.Send(app.CloseWindowMsg{ID: 1})
		})
	} else {
		btn = ui.ButtonText("Open Second Window", func() {
			app.Send(app.OpenWindowMsg{
				ID:     1,
				Config: app.WindowConfig{Title: "Lux — Second Window", Width: 400, Height: 300},
			})
		})
	}

	return ui.Column(
		sectionHeader("Multi-Window (Phase F)"),
		ui.Text("Multi-window support:"),
		ui.Spacer(8),
		ui.Padding(ui.UniformInsets(8), btn),
		ui.Spacer(4),
		ui.Text(fmt.Sprintf("Second window open: %v", m.SecondWindowOpen)),
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
	return ui.Column(
		sectionHeader("Images"),
		ui.Text("The ui.Image widget renders loaded images with size, scale mode, and opacity options."),
		ui.Text("Images are loaded via image.Store and referenced by draw.ImageID."),
		ui.Spacer(12),

		// 1. Basic image display — blue/gray checkerboard stretched to various sizes
		ui.TextStyled("Blue Checker (64×64 source → stretched to various sizes)", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
		ui.Spacer(4),
		ui.Row(
			ui.Image(m.ImgChecker1, ui.WithImageSize(64, 64), ui.WithImageScaleMode(draw.ImageScaleStretch)),
			ui.Spacer(8),
			ui.Image(m.ImgChecker1, ui.WithImageSize(128, 64), ui.WithImageScaleMode(draw.ImageScaleStretch)),
			ui.Spacer(8),
			ui.Image(m.ImgChecker1, ui.WithImageSize(48, 48), ui.WithImageScaleMode(draw.ImageScaleStretch)),
		),
		ui.Spacer(16),

		// 2. Scale modes — orange/teal checkerboard (128×64 → 100×100 box)
		ui.TextStyled("Scale Modes (orange/teal 128×64 → 100×100)", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
		ui.Spacer(4),
		ui.Row(
			ui.Column(
				ui.Text("Fit"),
				ui.Image(m.ImgChecker2, ui.WithImageSize(100, 100), ui.WithImageScaleMode(draw.ImageScaleFit)),
			),
			ui.Spacer(12),
			ui.Column(
				ui.Text("Fill"),
				ui.Image(m.ImgChecker2, ui.WithImageSize(100, 100), ui.WithImageScaleMode(draw.ImageScaleFill)),
			),
			ui.Spacer(12),
			ui.Column(
				ui.Text("Stretch"),
				ui.Image(m.ImgChecker2, ui.WithImageSize(100, 100), ui.WithImageScaleMode(draw.ImageScaleStretch)),
			),
		),
		ui.Spacer(16),

		// 3. Opacity control — pink/green checkerboard
		ui.TextStyled("Opacity (pink/green checker)", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
		ui.Spacer(4),
		ui.Row(
			ui.Image(m.ImgChecker3, ui.WithImageSize(120, 60), ui.WithImageScaleMode(draw.ImageScaleStretch), ui.WithImageOpacity(m.ImageOpacity)),
			ui.Spacer(12),
			ui.Text(fmt.Sprintf("%.0f%%", m.ImageOpacity*100)),
		),
		ui.Slider(m.ImageOpacity, func(v float32) { app.Send(SetImageOpacityMsg{v}) }),
		ui.Spacer(16),

		// 4. Alt text — blue checker again
		ui.TextStyled("Accessibility", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
		ui.Spacer(4),
		ui.Image(m.ImgChecker1,
			ui.WithImageSize(64, 64),
			ui.WithImageScaleMode(draw.ImageScaleStretch),
			ui.WithImageAlt("Blue and white checkerboard pattern"),
		),
		ui.Text("  Alt: \"Blue and white checkerboard pattern\""),
		ui.Spacer(16),

		// 5. API reference
		ui.TextStyled("API", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
		ui.Spacer(4),
		ui.Text("  store := image.NewStore()"),
		ui.Text("  id, _ := store.LoadFromFile(\"photo.png\")"),
		ui.Text("  id, _ := store.LoadFromBytes(data)"),
		ui.Text("  id, _ := store.LoadFromRGBA(w, h, rgba)"),
		ui.Text("  ui.Image(id, ui.WithImageSize(200, 150))"),
		ui.Text("  ui.Image(id, ui.WithImageScaleMode(draw.ImageScaleFit))"),
		ui.Text("  ui.Image(id, ui.WithImageOpacity(0.5))"),
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

	return ui.Column(
		sectionHeader("Shader Effects"),
		ui.Text("GPU shader-based visual effects via the Paint system."),
		ui.Text("Requires WGPU backend for rendering."),
		ui.Spacer(12),

		// Built-in effects
		ui.TextStyled("Built-in Shader Effects", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
		ui.Spacer(4),
		ui.Text("  ShaderEffectNoise   — Simplex/Perlin noise pattern"),
		ui.Text("  ShaderEffectPlasma  — Animated plasma effect"),
		ui.Text("  ShaderEffectVoronoi — Voronoi cell pattern"),
		ui.Spacer(12),

		// Paint API
		ui.TextStyled("Paint Variants for Backgrounds", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
		ui.Spacer(4),
		ui.Text("Image-based paints:"),
		ui.Text("  draw.ImagePaint(id, draw.ImageScaleFit)     — stretched/fitted image fill"),
		ui.Text("  draw.PatternPaint(id, draw.Size{32, 32})    — tiled image fill"),
		ui.Spacer(8),
		ui.Text("Shader paints:"),
		ui.Text("  draw.ShaderEffectPaint(draw.ShaderEffectNoise, 8.0)"),
		ui.Text("  draw.ShaderEffectPaint(draw.ShaderEffectPlasma, 2.0)"),
		ui.Text("  draw.ShaderEffectPaint(draw.ShaderEffectVoronoi, 12.0)"),
		ui.Spacer(8),
		ui.Text("Custom WGSL shader:"),
		ui.Text("  draw.ShaderPaint(wgslSource, params...)"),
		ui.Spacer(8),
		ui.Text("Shader + image texture:"),
		ui.Text("  draw.ShaderImagePaint(imgID, wgslSource, params...)"),
		ui.Spacer(16),

		// Integration notes
		ui.TextStyled("Integration", draw.TextStyle{Size: 13, Weight: draw.FontWeightSemiBold}),
		ui.Spacer(4),
		ui.Text("Paints are used as fill styles for surfaces and backgrounds."),
		ui.Text("Custom WGSL fragments receive uniforms via Params[0..7] and"),
		ui.Text("an optional image texture for PaintShaderImage."),
		ui.Text("Built-in effects are pre-compiled and cached by the GPU renderer."),
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
		Scroll:             &ui.ScrollState{},
		ToggleAnim:         ui.NewToggleState(),
		NavTree:            ui.NewTreeState(),
		ActiveSection:      "typography",
		VListScroll:        &ui.ScrollState{},
		DemoTree:           ui.NewTreeState(),
		AccordionState:     ui.NewAccordionState(),
		MenuBarState:       ui.NewMenuBarState(),
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
	if err := app.RunWithCmd(initial, update, view, runOpts...); err != nil {
		log.Fatal(err)
	}
}
