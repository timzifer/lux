package input

import "testing"

func TestIMEComposeMsgFields(t *testing.T) {
	msg := IMEComposeMsg{
		Text:        "にほん",
		CursorStart: 1,
		CursorEnd:   3,
	}
	if msg.Text != "にほん" {
		t.Errorf("Text = %q", msg.Text)
	}
	if msg.CursorStart != 1 || msg.CursorEnd != 3 {
		t.Errorf("Cursor: start=%d, end=%d", msg.CursorStart, msg.CursorEnd)
	}
}

func TestIMECommitMsgFields(t *testing.T) {
	msg := IMECommitMsg{Text: "日本"}
	if msg.Text != "日本" {
		t.Errorf("Text = %q", msg.Text)
	}
}

func TestIMEComposeEmptyText(t *testing.T) {
	msg := IMEComposeMsg{Text: "", CursorStart: 0, CursorEnd: 0}
	if msg.Text != "" {
		t.Errorf("empty compose text should be empty")
	}
}
