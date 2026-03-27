// Package richtext provides the RichTextEditor widget (RFC-003 §5.6).
//
// The editor extends the read-only RichText (Level 2) with full editability:
// cursor navigation, text selection, undo/redo, and inline formatting.
// It lives in a separate package because its WidgetState is considerably
// heavier than core widgets and its dependencies (clipboard, IME, undo stack)
// should not burden the framework core.
//
// The document model is based on AttributedString — a plain text string
// paired with run-length-encoded style attributes (inspired by Apple's
// NSAttributedString). This design aligns naturally with byte-offset-based
// InputState and enables lossless style preservation across edits.
//
// The editor's internal state (cursor, selection, undo stack) lives in
// the framework's WidgetState; the AttributedString is user-model owned.
package richtext
