// Test-only element types and constructors for ui package tests.
// Production code should use sub-package constructors (display.Text, layout.Row, etc.).
package ui

import (
	"math"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/validation"
)

// ── TextElement (test-only) ──────────────────────────────────────

type TextElement struct {
	BaseElement
	Content string
	Style   draw.TextStyle
}

func (TextElement) isElement() {}
func (n TextElement) ElementLabel() string { return n.Content }

func (n TextElement) LayoutSelf(ctx *LayoutContext) Bounds {
	style := ctx.Tokens.Typography.Body
	if n.Style.Size > 0 {
		style = n.Style
	}
	metrics := ctx.Canvas.MeasureText(n.Content, style)
	w := int(math.Ceil(float64(metrics.Width)))
	h := int(math.Ceil(float64(metrics.Ascent)))
	ctx.Canvas.DrawText(n.Content, draw.Pt(float32(ctx.Area.X), float32(ctx.Area.Y)), style, ctx.Tokens.Colors.Text.Primary)
	return Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: w, H: h, Baseline: h}
}

func (n TextElement) TreeEqual(other Element) bool {
	nb, ok := other.(TextElement)
	return ok && n.Content == nb.Content && n.Style == nb.Style
}

func (n TextElement) ResolveChildren(resolve func(Element, int) Element) Element { return n }
func (n TextElement) WalkAccess(b *AccessTreeBuilder, parentIdx int32)          {}

func Text(content string) Element                            { return TextElement{Content: content} }
func TextStyled(content string, style draw.TextStyle) Element { return TextElement{Content: content, Style: style} }

// ── ButtonElement (test-only) ────────────────────────────────────

type ButtonElement struct {
	BaseElement
	Content  Element
	OnClick  func()
	Variant  ButtonVariant
	Disabled bool
}

func (ButtonElement) isElement() {}

func (n ButtonElement) LayoutSelf(ctx *LayoutContext) Bounds {
	cb := ctx.MeasureChild(n.Content, Bounds{X: 0, Y: 0, W: ctx.Area.W, H: ctx.Area.H})
	w := cb.W + ButtonPadX*2
	h := cb.H + ButtonPadY*2
	buttonRect := draw.R(float32(ctx.Area.X), float32(ctx.Area.Y), float32(w), float32(h))
	fill, border, textColor := ButtonVariantColors(n.Variant, ctx.Tokens, 0)
	_ = border
	ctx.Canvas.FillRoundRect(buttonRect, ctx.Tokens.Radii.Button, draw.SolidPaint(fill))
	if txt, ok := n.Content.(TextElement); ok {
		style := ctx.Tokens.Typography.Label
		ctx.Canvas.DrawText(txt.Content, draw.Pt(float32(ctx.Area.X+ButtonPadX), float32(ctx.Area.Y+ButtonPadY)), style, textColor)
	} else {
		ctx.LayoutChild(n.Content, Bounds{X: ctx.Area.X + ButtonPadX, Y: ctx.Area.Y + ButtonPadY, W: cb.W, H: cb.H})
	}
	if !n.Disabled && ctx.IX != nil {
		ctx.IX.RegisterHit(buttonRect, n.OnClick)
	}
	return Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: w, H: h, Baseline: ButtonPadY + cb.Baseline}
}

func (n ButtonElement) TreeEqual(other Element) bool                                    { _, ok := other.(ButtonElement); return ok }
func (n ButtonElement) ResolveChildren(resolve func(Element, int) Element) Element      { return n }
func (n ButtonElement) WalkAccess(b *AccessTreeBuilder, parentIdx int32)                {}

func ButtonText(label string, onClick func()) Element {
	return ButtonElement{Content: TextElement{Content: label}, OnClick: onClick, Variant: ButtonFilled}
}
func ButtonTextDisabled(label string) Element {
	return ButtonElement{Content: TextElement{Content: label}, Variant: ButtonFilled, Disabled: true}
}
func ButtonOutlinedText(label string, onClick func()) Element {
	return ButtonElement{Content: TextElement{Content: label}, OnClick: onClick, Variant: ButtonOutlined}
}
func ButtonGhostText(label string, onClick func()) Element {
	return ButtonElement{Content: TextElement{Content: label}, OnClick: onClick, Variant: ButtonGhost}
}
func ButtonTonalText(label string, onClick func()) Element {
	return ButtonElement{Content: TextElement{Content: label}, OnClick: onClick, Variant: ButtonTonal}
}
func Button(content Element, onClick func()) Element {
	return ButtonElement{Content: content, OnClick: onClick, Variant: ButtonFilled}
}

// ── BoxElement (test-only) ───────────────────────────────────────

type BoxElement struct {
	BaseElement
	Axis     LayoutAxis
	Children []Element
}

func (BoxElement) isElement() {}

func (n BoxElement) LayoutSelf(ctx *LayoutContext) Bounds {
	if n.Axis == AxisRow {
		cursorX := ctx.Area.X
		maxH := 0
		for _, child := range n.Children {
			cb := ctx.LayoutChild(child, Bounds{X: cursorX, Y: ctx.Area.Y, W: ctx.Area.W, H: ctx.Area.H})
			cursorX += cb.W
			if cb.H > maxH {
				maxH = cb.H
			}
		}
		return Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: cursorX - ctx.Area.X, H: maxH}
	}
	cursorY := ctx.Area.Y
	maxW := 0
	for _, child := range n.Children {
		cb := ctx.LayoutChild(child, Bounds{X: ctx.Area.X, Y: cursorY, W: ctx.Area.W, H: ctx.Area.H})
		cursorY += cb.H
		if cb.W > maxW {
			maxW = cb.W
		}
	}
	return Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: maxW, H: cursorY - ctx.Area.Y}
}

func (n BoxElement) TreeEqual(other Element) bool {
	nb, ok := other.(BoxElement)
	if !ok || n.Axis != nb.Axis || len(n.Children) != len(nb.Children) {
		return false
	}
	return true
}

func (n BoxElement) ResolveChildren(resolve func(Element, int) Element) Element {
	out := n
	out.Children = make([]Element, len(n.Children))
	for i, c := range n.Children {
		out.Children[i] = resolve(c, i)
	}
	return out
}

func (n BoxElement) WalkAccess(b *AccessTreeBuilder, parentIdx int32) {
	for _, child := range n.Children {
		b.Walk(child, parentIdx)
	}
}

func Column(children ...Element) Element { return BoxElement{Axis: AxisColumn, Children: children} }
func Row(children ...Element) Element    { return BoxElement{Axis: AxisRow, Children: children} }

// ── Simple elements (test-only) ──────────────────────────────────

type SpacerElement struct {
	BaseElement
	Size float32
}

func (SpacerElement) isElement() {}
func (n SpacerElement) LayoutSelf(ctx *LayoutContext) Bounds {
	s := int(n.Size)
	return Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: s, H: s, Baseline: s}
}
func (n SpacerElement) TreeEqual(other Element) bool                               { _, ok := other.(SpacerElement); return ok }
func (n SpacerElement) ResolveChildren(resolve func(Element, int) Element) Element { return n }
func (n SpacerElement) WalkAccess(b *AccessTreeBuilder, parentIdx int32)           {}

func Spacer(size float32) Element { return SpacerElement{Size: size} }

type DividerElement struct{ BaseElement }

func (DividerElement) isElement() {}
func (n DividerElement) LayoutSelf(ctx *LayoutContext) Bounds {
	return Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: ctx.Area.W, H: 1, Baseline: 1}
}
func (n DividerElement) TreeEqual(other Element) bool                               { _, ok := other.(DividerElement); return ok }
func (n DividerElement) ResolveChildren(resolve func(Element, int) Element) Element { return n }
func (n DividerElement) WalkAccess(b *AccessTreeBuilder, parentIdx int32)           {}

func Divider() Element { return DividerElement{} }

type IconElement struct {
	BaseElement
	Name string
	Size float32
}

func (IconElement) isElement() {}
func (n IconElement) LayoutSelf(ctx *LayoutContext) Bounds {
	s := int(n.Size)
	if s == 0 {
		s = 24
	}
	return Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: s, H: s, Baseline: s}
}
func (n IconElement) TreeEqual(other Element) bool                               { _, ok := other.(IconElement); return ok }
func (n IconElement) ResolveChildren(resolve func(Element, int) Element) Element { return n }
func (n IconElement) WalkAccess(b *AccessTreeBuilder, parentIdx int32)           {}

func Icon(name string) Element           { return IconElement{Name: name} }
func IconSize(name string, s float32) Element { return IconElement{Name: name, Size: s} }

// ── Container wrappers (test-only) ───────────────────────────────

type PaddingElement struct {
	BaseElement
	Insets draw.Insets
	Child  Element
}

func (PaddingElement) isElement() {}
func (n PaddingElement) LayoutSelf(ctx *LayoutContext) Bounds {
	cb := ctx.LayoutChild(n.Child, ctx.Area)
	return cb
}
func (n PaddingElement) TreeEqual(other Element) bool {
	o, ok := other.(PaddingElement)
	return ok && n.Insets == o.Insets
}
func (n PaddingElement) ResolveChildren(resolve func(Element, int) Element) Element {
	n.Child = resolve(n.Child, 0)
	return n
}
func (n PaddingElement) WalkAccess(b *AccessTreeBuilder, parentIdx int32) { b.Walk(n.Child, parentIdx) }

func Padding(insets draw.Insets, child Element) Element { return PaddingElement{Insets: insets, Child: child} }

type SizedBoxElement struct {
	BaseElement
	Width, Height float32
	Child         Element
}

func (SizedBoxElement) isElement() {}
func (n SizedBoxElement) LayoutSelf(ctx *LayoutContext) Bounds {
	w, h := int(n.Width), int(n.Height)
	if n.Child != nil {
		ctx.LayoutChild(n.Child, Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: w, H: h})
	}
	return Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: w, H: h}
}
func (n SizedBoxElement) TreeEqual(other Element) bool {
	o, ok := other.(SizedBoxElement)
	return ok && n.Width == o.Width && n.Height == o.Height
}
func (n SizedBoxElement) ResolveChildren(resolve func(Element, int) Element) Element {
	if n.Child != nil {
		n.Child = resolve(n.Child, 0)
	}
	return n
}
func (n SizedBoxElement) WalkAccess(b *AccessTreeBuilder, parentIdx int32) {
	if n.Child != nil {
		b.Walk(n.Child, parentIdx)
	}
}

func SizedBox(width, height float32, child ...Element) Element {
	var c Element
	if len(child) > 0 {
		c = child[0]
	}
	return SizedBoxElement{Width: width, Height: height, Child: c}
}

type ExpandedElement struct {
	BaseElement
	Child Element
	Grow  float32
}

func (ExpandedElement) isElement() {}
func (n ExpandedElement) LayoutSelf(ctx *LayoutContext) Bounds { return ctx.LayoutChild(n.Child, ctx.Area) }
func (n ExpandedElement) TreeEqual(other Element) bool         { _, ok := other.(ExpandedElement); return ok }
func (n ExpandedElement) ResolveChildren(resolve func(Element, int) Element) Element {
	n.Child = resolve(n.Child, 0)
	return n
}
func (n ExpandedElement) WalkAccess(b *AccessTreeBuilder, parentIdx int32) { b.Walk(n.Child, parentIdx) }

func Expanded(child Element, flex ...float32) Element {
	grow := float32(1)
	if len(flex) > 0 {
		grow = flex[0]
	}
	return ExpandedElement{Child: child, Grow: grow}
}

type StackElement struct {
	BaseElement
	Children []Element
}

func (StackElement) isElement() {}
func (n StackElement) LayoutSelf(ctx *LayoutContext) Bounds {
	maxW, maxH := 0, 0
	for _, child := range n.Children {
		cb := ctx.LayoutChild(child, ctx.Area)
		if cb.W > maxW { maxW = cb.W }
		if cb.H > maxH { maxH = cb.H }
	}
	return Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: maxW, H: maxH}
}
func (n StackElement) TreeEqual(other Element) bool { _, ok := other.(StackElement); return ok }
func (n StackElement) ResolveChildren(resolve func(Element, int) Element) Element {
	out := n
	out.Children = make([]Element, len(n.Children))
	for i, c := range n.Children {
		out.Children[i] = resolve(c, i)
	}
	return out
}
func (n StackElement) WalkAccess(b *AccessTreeBuilder, parentIdx int32) {
	for _, child := range n.Children {
		b.Walk(child, parentIdx)
	}
}

func Stack(children ...Element) Element { return StackElement{Children: children} }

// ── Leaf widgets (test-only, minimal layout) ─────────────────────

type testLeafElement struct {
	BaseElement
	w, h int
}

func (testLeafElement) isElement() {}
func (n testLeafElement) LayoutSelf(ctx *LayoutContext) Bounds {
	return Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: n.w, H: n.h}
}
func (n testLeafElement) TreeEqual(other Element) bool                               { return false }
func (n testLeafElement) ResolveChildren(resolve func(Element, int) Element) Element { return n }
func (n testLeafElement) WalkAccess(b *AccessTreeBuilder, parentIdx int32)           {}

func Checkbox(label string, checked bool, onToggle func(bool)) Element   { return testLeafElement{w: 120, h: 20} }
func Radio(label string, selected bool, onSelect func()) Element         { return testLeafElement{w: 120, h: 20} }
type ToggleState struct{ Pos float32 }
func Toggle(on bool, onToggle func(bool), state ...*ToggleState) Element { return testLeafElement{w: 36, h: 20} }
func Slider(value float32, onChange func(float32)) Element               { return testLeafElement{w: 200, h: 20} }
func ProgressBar(value float32) Element                                  { return testLeafElement{w: 200, h: 8} }
func ProgressBarIndeterminate(phase ...float32) Element                   { return testLeafElement{w: 200, h: 8} }

func TextField(value, placeholder string, opts ...func(*testTextFieldCfg)) Element { return testLeafElement{w: 200, h: 36} }
func Select(value string, options []string, opts ...func(*testSelectCfg)) Element  { return testLeafElement{w: 200, h: 36} }

type testTextFieldCfg struct{}
type testSelectCfg struct{}

func WithOnChange(fn func(string)) func(*testTextFieldCfg) { return func(*testTextFieldCfg) {} }
func WithFocus(fm *FocusManager) func(*testTextFieldCfg)    { return func(*testTextFieldCfg) {} }

type SelectState struct{ Open bool }
func WithSelectState(s *SelectState) func(*testSelectCfg) { return func(*testSelectCfg) {} }
func WithOnSelect(fn func(string)) func(*testSelectCfg)   { return func(*testSelectCfg) {} }

type ScrollViewElement struct {
	BaseElement
	Child     Element
	MaxHeight float32
	State     *ScrollState
}

func (ScrollViewElement) isElement() {}
func (n ScrollViewElement) LayoutSelf(ctx *LayoutContext) Bounds {
	return ctx.LayoutChild(n.Child, ctx.Area)
}
func (n ScrollViewElement) TreeEqual(other Element) bool {
	o, ok := other.(ScrollViewElement)
	return ok && n.MaxHeight == o.MaxHeight
}
func (n ScrollViewElement) ResolveChildren(resolve func(Element, int) Element) Element {
	n.Child = resolve(n.Child, 0)
	return n
}
func (n ScrollViewElement) WalkAccess(b *AccessTreeBuilder, parentIdx int32) { b.Walk(n.Child, parentIdx) }

func ScrollView(child Element, maxHeight float32, state ...*ScrollState) Element {
	var s *ScrollState
	if len(state) > 0 {
		s = state[0]
	}
	return ScrollViewElement{Child: child, MaxHeight: maxHeight, State: s}
}

func Card(children ...Element) Element {
	if len(children) == 1 {
		return testLeafElement{w: 200, h: 100}
	}
	return testLeafElement{w: 200, h: 100}
}

const badgeMinSize = 20

func Badge(content Element) Element                                          { return testLeafElement{w: 20, h: 20} }
func BadgeText(label string) Element                                         { return testLeafElement{w: 20, h: 20} }
func Chip(label Element, selected bool, onClick func()) Element              { return testLeafElement{w: 80, h: 32} }
func ChipDismissible(label Element, selected bool, onClick, onDismiss func()) Element { return testLeafElement{w: 80, h: 32} }
func Tooltip(trigger, content Element) Element                               { return testLeafElement{w: 100, h: 20} }
func TooltipVisible(trigger, content Element, visible bool) Element          { return testLeafElement{w: 100, h: 20} }

func Tabs(items []TabItem, selected int, onSelect func(int)) Element        { return testLeafElement{w: 400, h: 200} }

// Moved types re-declared for tests

type TabItem struct {
	Header  Element
	Content Element
}

type AccordionSection struct {
	Header  Element
	Content Element
}

type AccordionState struct {
	Expanded map[int]bool
}

func NewAccordionState() *AccordionState { return &AccordionState{Expanded: make(map[int]bool)} }

func Accordion(sections []AccordionSection, state *AccordionState) Element { return testLeafElement{w: 200, h: 100} }

type MenuItem struct {
	Label   Element
	OnClick func()
	Items   []MenuItem
}

type MenuBarState struct {
	OpenIndex int
}

func NewMenuBarState() *MenuBarState { return &MenuBarState{OpenIndex: -1} }

func MenuBar(items []MenuItem, state *MenuBarState) Element                { return testLeafElement{w: 400, h: 32} }
func ContextMenu(items []MenuItem, visible bool, x, y float32) Element     { return testLeafElement{w: 0, h: 0} }

// FormField test support
type FormFieldElement struct {
	BaseElement
	Child    Element
	Label    string
	Hint     string
	Result   validation.FieldResult
}

func (FormFieldElement) isElement() {}
func (n FormFieldElement) LayoutSelf(ctx *LayoutContext) Bounds { return ctx.LayoutChild(n.Child, ctx.Area) }
func (n FormFieldElement) TreeEqual(other Element) bool         { _, ok := other.(FormFieldElement); return ok }
func (n FormFieldElement) ResolveChildren(resolve func(Element, int) Element) Element {
	n.Child = resolve(n.Child, 0)
	return n
}
func (n FormFieldElement) WalkAccess(b *AccessTreeBuilder, parentIdx int32) { b.Walk(n.Child, parentIdx) }

type testFormFieldOpt func(*FormFieldElement)
func WithFormLabel(label string) testFormFieldOpt       { return func(f *FormFieldElement) { f.Label = label } }
func WithFormHint(hint string) testFormFieldOpt         { return func(f *FormFieldElement) { f.Hint = hint } }
func WithFormValidation(r validation.FieldResult) testFormFieldOpt { return func(f *FormFieldElement) { f.Result = r } }

func FormField(child Element, opts ...testFormFieldOpt) Element {
	el := FormFieldElement{Child: child}
	for _, opt := range opts {
		opt(&el)
	}
	return el
}
