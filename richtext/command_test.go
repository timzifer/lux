package richtext

import "testing"

// ── BoldCommand ─────────────────────────────────────────────────

func TestBoldCommand_IsActive_Cursor(t *testing.T) {
	doc := Build(
		S("Hello ", SpanStyle{Bold: true}),
		S("World"),
	)
	cmd := BoldCommand{}
	if !cmd.IsActive(doc, 0, 0) {
		t.Error("expected active at offset 0 (bold run)")
	}
	if cmd.IsActive(doc, 8, 8) {
		t.Error("expected inactive at offset 8 (plain run)")
	}
}

func TestBoldCommand_IsActive_Selection(t *testing.T) {
	doc := Build(
		S("Hello ", SpanStyle{Bold: true}),
		S("World"),
	)
	cmd := BoldCommand{}
	// Selection entirely within bold run.
	if !cmd.IsActive(doc, 0, 5) {
		t.Error("expected active for selection within bold run")
	}
	// Selection spanning bold and non-bold.
	if cmd.IsActive(doc, 0, 8) {
		t.Error("expected inactive for selection spanning both runs")
	}
}

func TestBoldCommand_Execute_Selection(t *testing.T) {
	doc := NewAttributedString("Hello World")
	cmd := BoldCommand{}

	// Apply bold to "Hello".
	newDoc, pending := cmd.Execute(doc, 0, 5)
	if pending != nil {
		t.Error("expected nil pending for selection execute")
	}
	if !newDoc.RunAt(0).Bold {
		t.Error("expected Bold at offset 0 after execute")
	}
	if newDoc.RunAt(6).Bold {
		t.Error("expected non-Bold at offset 6 after execute")
	}

	// Toggle off: apply bold again to already-bold range.
	newDoc2, _ := cmd.Execute(newDoc, 0, 5)
	if newDoc2.RunAt(0).Bold {
		t.Error("expected Bold toggled off")
	}
}

func TestBoldCommand_Execute_NoSelection(t *testing.T) {
	doc := NewAttributedString("Hello")
	cmd := BoldCommand{}

	newDoc, pending := cmd.Execute(doc, 3, 3)
	if newDoc.Text != doc.Text {
		t.Error("document should not change for no-selection execute")
	}
	if pending == nil {
		t.Fatal("expected pending mod for no-selection execute")
	}

	// The pending mod should toggle Bold on.
	style := pending(SpanStyle{Italic: true})
	if !style.Bold {
		t.Error("expected Bold to be set by pending mod")
	}
	if !style.Italic {
		t.Error("expected Italic to be preserved by pending mod")
	}
}

// ── ItalicCommand ───────────────────────────────────────────────

func TestItalicCommand_Execute_Selection(t *testing.T) {
	doc := NewAttributedString("Hello World")
	cmd := ItalicCommand{}

	newDoc, _ := cmd.Execute(doc, 0, 5)
	if !newDoc.RunAt(0).Italic {
		t.Error("expected Italic at offset 0")
	}
	if newDoc.RunAt(6).Italic {
		t.Error("expected non-Italic at offset 6")
	}
}

func TestItalicCommand_IsActive_Cursor(t *testing.T) {
	doc := Styled("Hello", SpanStyle{Italic: true})
	cmd := ItalicCommand{}
	if !cmd.IsActive(doc, 2, 2) {
		t.Error("expected active at cursor in italic text")
	}
}

// ── UnderlineCommand ────────────────────────────────────────────

func TestUnderlineCommand_Execute_Selection(t *testing.T) {
	doc := NewAttributedString("Hello World")
	cmd := UnderlineCommand{}

	newDoc, _ := cmd.Execute(doc, 6, 11)
	if newDoc.RunAt(0).Underline {
		t.Error("expected non-Underline at offset 0")
	}
	if !newDoc.RunAt(7).Underline {
		t.Error("expected Underline at offset 7")
	}
}

func TestUnderlineCommand_Execute_NoSelection(t *testing.T) {
	doc := NewAttributedString("Hello")
	cmd := UnderlineCommand{}

	_, pending := cmd.Execute(doc, 0, 0)
	if pending == nil {
		t.Fatal("expected pending mod")
	}
	style := pending(SpanStyle{Bold: true})
	if !style.Underline || !style.Bold {
		t.Error("expected Underline toggled on and Bold preserved")
	}
}

// ── DefaultCommands ─────────────────────────────────────────────

func TestDefaultCommands_Icons(t *testing.T) {
	cmds := DefaultCommands()
	for i, cmd := range cmds {
		if cmd.Icon() == nil {
			t.Errorf("command %d returned nil Icon", i)
		}
	}
}

// ── ToggleStyleFunc ─────────────────────────────────────────────

func TestToggleStyleFunc_PreservesOtherAttrs(t *testing.T) {
	doc := Build(
		S("Hello ", SpanStyle{Bold: true, Italic: true}),
		S("World"),
	)

	// Toggle Bold off in the first run.
	result := doc.ToggleStyleFunc(0, 6, func(s SpanStyle) SpanStyle {
		s.Bold = false
		return s
	})

	style := result.RunAt(0)
	if style.Bold {
		t.Error("expected Bold to be toggled off")
	}
	if !style.Italic {
		t.Error("expected Italic to be preserved")
	}
}

func TestToggleStyleFunc_PartialOverlap(t *testing.T) {
	doc := Build(
		S("AAA", SpanStyle{Bold: true}),
		S("BBB", SpanStyle{Italic: true}),
	)

	// Toggle Underline on for "AB" (offset 2-4).
	result := doc.ToggleStyleFunc(2, 4, func(s SpanStyle) SpanStyle {
		s.Underline = true
		return s
	})

	// "AA" (0-2) should be bold, no underline.
	s0 := result.RunAt(0)
	if !s0.Bold || s0.Underline {
		t.Errorf("unexpected style at 0: %+v", s0)
	}

	// "A" (2-3) should be bold + underline.
	s2 := result.RunAt(2)
	if !s2.Bold || !s2.Underline {
		t.Errorf("unexpected style at 2: %+v", s2)
	}

	// "B" (3-4) should be italic + underline.
	s3 := result.RunAt(3)
	if !s3.Italic || !s3.Underline {
		t.Errorf("unexpected style at 3: %+v", s3)
	}

	// "BB" (4-6) should be italic, no underline.
	s5 := result.RunAt(5)
	if !s5.Italic || s5.Underline {
		t.Errorf("unexpected style at 5: %+v", s5)
	}
}

// ── AllMatch ────────────────────────────────────────────────────

func TestAllMatch_AllBold(t *testing.T) {
	doc := Styled("Hello World", SpanStyle{Bold: true})
	if !doc.AllMatch(0, 11, func(s SpanStyle) bool { return s.Bold }) {
		t.Error("expected all bold")
	}
}

func TestAllMatch_Mixed(t *testing.T) {
	doc := Build(
		S("Hello ", SpanStyle{Bold: true}),
		S("World"),
	)
	if doc.AllMatch(0, 11, func(s SpanStyle) bool { return s.Bold }) {
		t.Error("expected not all bold")
	}
}

func TestAllMatch_EmptyRange(t *testing.T) {
	doc := NewAttributedString("Hello")
	if !doc.AllMatch(3, 3, func(s SpanStyle) bool { return s.Bold }) {
		t.Error("expected true for empty range")
	}
}

func TestAllMatch_SubRange(t *testing.T) {
	doc := Build(
		S("Hello ", SpanStyle{Bold: true}),
		S("World"),
	)
	// Only checking within the bold run.
	if !doc.AllMatch(0, 5, func(s SpanStyle) bool { return s.Bold }) {
		t.Error("expected all bold within first run")
	}
}

// ── StrikethroughCommand ───────────────────────────────────────

func TestStrikethroughCommand_IsActive_Cursor(t *testing.T) {
	doc := Build(
		S("Hello ", SpanStyle{Strikethrough: true}),
		S("World"),
	)
	cmd := StrikethroughCommand{}
	if !cmd.IsActive(doc, 0, 0) {
		t.Error("expected active at offset 0 (strikethrough run)")
	}
	if cmd.IsActive(doc, 8, 8) {
		t.Error("expected inactive at offset 8 (plain run)")
	}
}

func TestStrikethroughCommand_IsActive_Selection(t *testing.T) {
	doc := Build(
		S("Hello ", SpanStyle{Strikethrough: true}),
		S("World"),
	)
	cmd := StrikethroughCommand{}
	if !cmd.IsActive(doc, 0, 5) {
		t.Error("expected active for selection within strikethrough run")
	}
	if cmd.IsActive(doc, 0, 8) {
		t.Error("expected inactive for selection spanning both runs")
	}
}

func TestStrikethroughCommand_Execute_Selection(t *testing.T) {
	doc := NewAttributedString("Hello World")
	cmd := StrikethroughCommand{}

	// Apply strikethrough to "Hello".
	newDoc, pending := cmd.Execute(doc, 0, 5)
	if pending != nil {
		t.Error("expected nil pending for selection execute")
	}
	if !newDoc.RunAt(0).Strikethrough {
		t.Error("expected Strikethrough at offset 0 after execute")
	}
	if newDoc.RunAt(6).Strikethrough {
		t.Error("expected non-Strikethrough at offset 6 after execute")
	}

	// Toggle off.
	newDoc2, _ := cmd.Execute(newDoc, 0, 5)
	if newDoc2.RunAt(0).Strikethrough {
		t.Error("expected Strikethrough toggled off")
	}
}

func TestStrikethroughCommand_Execute_NoSelection(t *testing.T) {
	doc := NewAttributedString("Hello")
	cmd := StrikethroughCommand{}

	newDoc, pending := cmd.Execute(doc, 3, 3)
	if newDoc.Text != doc.Text {
		t.Error("document should not change for no-selection execute")
	}
	if pending == nil {
		t.Fatal("expected pending mod for no-selection execute")
	}

	// The pending mod should toggle Strikethrough on.
	style := pending(SpanStyle{Bold: true})
	if !style.Strikethrough {
		t.Error("expected Strikethrough to be set by pending mod")
	}
	if !style.Bold {
		t.Error("expected Bold to be preserved by pending mod")
	}
}

func TestStrikethroughCommand_PreservesOtherStyles(t *testing.T) {
	doc := Build(
		S("Hello", SpanStyle{Bold: true, Italic: true, Underline: true}),
	)
	cmd := StrikethroughCommand{}

	newDoc, _ := cmd.Execute(doc, 0, 5)
	s := newDoc.RunAt(0)
	if !s.Bold || !s.Italic || !s.Underline {
		t.Error("other styles should be preserved when applying strikethrough")
	}
	if !s.Strikethrough {
		t.Error("strikethrough should be applied")
	}
}

// ── DefaultCommands ────────────────────────────────────────────

func TestDefaultCommands_IncludesStrikethrough(t *testing.T) {
	cmds := DefaultCommands()
	if len(cmds) != 4 {
		t.Fatalf("expected 4 default commands, got %d", len(cmds))
	}
	if _, ok := cmds[3].(StrikethroughCommand); !ok {
		t.Errorf("expected 4th command to be StrikethroughCommand, got %T", cmds[3])
	}
}
