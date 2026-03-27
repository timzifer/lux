package input

// IMEComposeMsg carries pre-edit (composition) text from an IME (RFC-002 §2.2).
// The compose text is the in-progress text being composed by the user via the
// input method editor, displayed inline but not yet committed.
type IMEComposeMsg struct {
	// Text is the current pre-edit string. May be empty when composition ends.
	Text string
	// CursorStart is the cursor position within the compose text (in runes).
	CursorStart int
	// CursorEnd is the selection end within the compose text (in runes).
	// When equal to CursorStart, there is no selection.
	CursorEnd int
}

// IMECommitMsg carries the final committed text from an IME (RFC-002 §2.2).
// After receiving this message, the composition is complete and the text
// should be inserted at the cursor position.
type IMECommitMsg struct {
	Text string
}
