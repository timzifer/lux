package richtext

import (
	"testing"

	"github.com/timzifer/lux/ui"
)

func TestNewEditorWithToolbar(t *testing.T) {
	doc := NewAttributedString("Hello World")
	el := NewEditorWithToolbar(doc)
	if el == nil {
		t.Fatal("expected non-nil element")
	}
	// Should produce a WidgetElement.
	if _, ok := el.(ui.WidgetElement); !ok {
		t.Fatalf("expected WidgetElement, got %T", el)
	}
}

func TestNewEditorWithToolbar_Options(t *testing.T) {
	doc := NewAttributedString("Test")
	var changed bool
	el := NewEditorWithToolbar(doc,
		WithWidgetOnChange(func(AttributedString) { changed = true }),
		WithWidgetRows(10),
		WithWidgetPlaceholder("Enter text..."),
		WithWidgetReadOnly(),
	)
	if el == nil {
		t.Fatal("expected non-nil element")
	}
	we, ok := el.(ui.WidgetElement)
	if !ok {
		t.Fatalf("expected WidgetElement, got %T", el)
	}
	w := we.W.(RichTextEditorWidget)
	if w.Rows != 10 {
		t.Errorf("expected 10 rows, got %d", w.Rows)
	}
	if w.Placeholder != "Enter text..." {
		t.Errorf("unexpected placeholder: %q", w.Placeholder)
	}
	if !w.ReadOnly {
		t.Error("expected ReadOnly")
	}
	if w.OnChange == nil {
		t.Error("expected non-nil OnChange")
	}
	w.OnChange(doc)
	if !changed {
		t.Error("OnChange not called")
	}
}

func TestNewEditorWithToolbar_CustomCommands(t *testing.T) {
	doc := NewAttributedString("Test")
	cmds := []ToolbarCommand{BoldCommand{}}
	el := NewEditorWithToolbar(doc, WithWidgetCommands(cmds))
	we := el.(ui.WidgetElement)
	w := we.W.(RichTextEditorWidget)
	if len(w.Commands) != 1 {
		t.Errorf("expected 1 command, got %d", len(w.Commands))
	}
}

func TestEditorState_PendingMods(t *testing.T) {
	state := &editorState{}
	if len(state.PendingMods) != 0 {
		t.Fatal("expected empty pending mods")
	}

	// Simulate adding a pending mod.
	state.PendingMods = append(state.PendingMods, func(s SpanStyle) SpanStyle {
		s.Bold = true
		return s
	})
	if len(state.PendingMods) != 1 {
		t.Fatal("expected 1 pending mod")
	}

	// Apply and clear.
	style := state.PendingMods[0](SpanStyle{})
	if !style.Bold {
		t.Error("expected Bold from pending mod")
	}
	state.PendingMods = nil
	if len(state.PendingMods) != 0 {
		t.Fatal("expected empty after clear")
	}
}
