package richtext

import (
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/layout"
	"github.com/timzifer/lux/ui/nav"
)

// ── Widget State ────────────────────────────────────────────────

// editorState is the internal state for RichTextEditorWidget,
// persisted across frames by the Reconciler.
type editorState struct {
	// PendingMods are style transformations queued when the user
	// toggles a formatting button with no selection. They are
	// applied to the next inserted text and then cleared.
	PendingMods []func(SpanStyle) SpanStyle
}

// ── Widget ──────────────────────────────────────────────────────

// RichTextEditorWidget is a Widget that composes a formatting toolbar
// with a RichTextEditor. The toolbar buttons reflect the selection's
// current style and toggle formatting on click.
//
// Usage:
//
//	richtext.NewEditorWithToolbar(doc,
//	    richtext.WithWidgetOnChange(func(as richtext.AttributedString) { ... }),
//	    richtext.WithWidgetFocus(fm),
//	)
type RichTextEditorWidget struct {
	// Value is the current document content (user-model owned).
	Value AttributedString

	// OnChange is called when the document changes (from typing or
	// toolbar actions).
	OnChange func(AttributedString)

	// Commands defines the toolbar buttons. Defaults to
	// DefaultCommands() (Bold, Italic, Underline).
	Commands []ToolbarCommand

	// ReadOnly disables editing but allows selection and copy.
	ReadOnly bool

	// Rows controls the visible row count of the editor (default 4).
	Rows int

	// Focus links the editor to a FocusManager for keyboard input.
	Focus *ui.FocusManager

	// Scroll links the editor to a ScrollState for internal scrolling.
	Scroll *ui.ScrollState

	// Placeholder is shown when the editor content is empty.
	Placeholder string
}

// Render implements ui.Widget.
func (w RichTextEditorWidget) Render(ctx ui.RenderCtx, rawState ui.WidgetState) (ui.Element, ui.WidgetState) {
	state := ui.AdoptState[editorState](rawState)

	// Read selection from the previous frame's FocusManager state.
	selStart, selEnd := w.selectionRange()

	// Capture the editor's focus UID so toolbar clicks can restore it.
	// The app loop calls fm.Blur() on every mouse-down before dispatching
	// hit targets, so without this the editor would lose focus when the
	// user clicks a toolbar button.
	var editorFocusUID ui.UID
	if w.Focus != nil {
		editorFocusUID = w.Focus.FocusedUID()
	}

	// Build toolbar items from commands.
	items := make([]nav.ToolbarItem, len(w.Commands))
	for i, cmd := range w.Commands {
		cmd := cmd // capture loop variable
		active := cmd.IsActive(w.Value, selStart, selEnd)

		items[i] = nav.ToolbarItem{
			Element: cmd.Icon(),
			Toggle:  true,
			Active:  active,
			OnClick: w.makeCommandHandler(cmd, selStart, selEnd, state, editorFocusUID),
		}
	}

	// Wrap OnChange to apply pending style mods to inserted text.
	wrappedOnChange := w.makeWrappedOnChange(state)

	// Build the element tree: toolbar on top, editor below.
	toolbar := nav.NewToolbar(items)

	var editorOpts []Option
	if wrappedOnChange != nil {
		editorOpts = append(editorOpts, WithOnChange(wrappedOnChange))
	}
	if w.ReadOnly {
		editorOpts = append(editorOpts, WithReadOnly())
	}
	if w.Rows > 0 {
		editorOpts = append(editorOpts, WithRows(w.Rows))
	}
	if w.Focus != nil {
		editorOpts = append(editorOpts, WithFocus(w.Focus))
	}
	if w.Scroll != nil {
		editorOpts = append(editorOpts, WithScroll(w.Scroll))
	}
	if w.Placeholder != "" {
		editorOpts = append(editorOpts, WithPlaceholder(w.Placeholder))
	}

	editor := New(w.Value, editorOpts...)

	return layout.Column(toolbar, editor), state
}

// selectionRange returns the current selection (or cursor position) from
// the FocusManager. Returns (cursor, cursor) when there is no selection.
func (w RichTextEditorWidget) selectionRange() (int, int) {
	if w.Focus == nil || w.Focus.Input == nil {
		return 0, 0
	}
	inp := w.Focus.Input
	if inp.HasSelection() {
		return inp.SelectionRange()
	}
	return inp.CursorOffset, inp.CursorOffset
}

// makeCommandHandler returns an OnClick handler for a toolbar command.
// When the command produces a new document, OnChange is called. When it
// produces a pending modifier (no selection), the mod is stored in state.
//
// The handler restores the editor's focus UID after execution because
// the app loop calls fm.Blur() on every mouse-down before hit targets
// fire. Without the restore, clicking a toolbar button would defocus
// the editor and lose the selection.
func (w RichTextEditorWidget) makeCommandHandler(cmd ToolbarCommand, selStart, selEnd int, state *editorState, editorFocusUID ui.UID) func() {
	return func() {
		// Restore editor focus (blurred by the app loop's mouse-down handler).
		if w.Focus != nil && editorFocusUID != 0 {
			w.Focus.SetFocusedUID(editorFocusUID)
		}

		newDoc, pendingMod := cmd.Execute(w.Value, selStart, selEnd)
		if pendingMod != nil {
			state.PendingMods = append(state.PendingMods, pendingMod)
		}
		if w.OnChange != nil && !newDoc.Equal(w.Value) {
			w.OnChange(newDoc)
		}
	}
}

// makeWrappedOnChange wraps the user's OnChange to apply pending style
// mods to freshly inserted text.
func (w RichTextEditorWidget) makeWrappedOnChange(state *editorState) func(AttributedString) {
	if w.OnChange == nil {
		return nil
	}
	return func(newDoc AttributedString) {
		// Apply pending mods to inserted text.
		if len(state.PendingMods) > 0 {
			oldText := w.Value.Text
			newText := newDoc.Text

			// Find the insertion range via common prefix/suffix.
			pfx := commonPrefixLen(oldText, newText)
			sfx := commonSuffixLen(oldText, newText, pfx)
			insEnd := len(newText) - sfx

			// Only apply if there was an insertion (not just deletion).
			if insEnd > pfx {
				for _, mod := range state.PendingMods {
					newDoc = newDoc.ToggleStyleFunc(pfx, insEnd, mod)
				}
			}
			state.PendingMods = nil
		}
		w.OnChange(newDoc)
	}
}

// ── Widget Options ──────────────────────────────────────────────

// WidgetOption configures a RichTextEditorWidget.
type WidgetOption func(*RichTextEditorWidget)

// WithWidgetOnChange sets the change callback.
func WithWidgetOnChange(fn func(AttributedString)) WidgetOption {
	return func(w *RichTextEditorWidget) { w.OnChange = fn }
}

// WithWidgetCommands sets the toolbar commands.
func WithWidgetCommands(cmds []ToolbarCommand) WidgetOption {
	return func(w *RichTextEditorWidget) { w.Commands = cmds }
}

// WithWidgetReadOnly sets read-only mode.
func WithWidgetReadOnly() WidgetOption {
	return func(w *RichTextEditorWidget) { w.ReadOnly = true }
}

// WithWidgetRows sets the number of visible rows.
func WithWidgetRows(n int) WidgetOption {
	return func(w *RichTextEditorWidget) { w.Rows = n }
}

// WithWidgetFocus links the widget to a FocusManager.
func WithWidgetFocus(fm *ui.FocusManager) WidgetOption {
	return func(w *RichTextEditorWidget) { w.Focus = fm }
}

// WithWidgetScroll links the widget to a ScrollState.
func WithWidgetScroll(s *ui.ScrollState) WidgetOption {
	return func(w *RichTextEditorWidget) { w.Scroll = s }
}

// WithWidgetPlaceholder sets the placeholder text.
func WithWidgetPlaceholder(p string) WidgetOption {
	return func(w *RichTextEditorWidget) { w.Placeholder = p }
}

// ── Constructor ─────────────────────────────────────────────────

// NewEditorWithToolbar creates a RichTextEditorWidget wrapped as a
// ui.Element. By default it includes Bold, Italic and Underline
// toolbar commands.
func NewEditorWithToolbar(doc AttributedString, opts ...WidgetOption) ui.Element {
	w := RichTextEditorWidget{
		Value:    doc,
		Commands: DefaultCommands(),
	}
	for _, opt := range opts {
		opt(&w)
	}
	return ui.Component(w)
}
