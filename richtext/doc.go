// Package richtext provides the RichTextEditor widget (RFC-003 §5.6).
//
// The editor extends the read-only RichText (Level 2) with full editability:
// cursor navigation, text selection, undo/redo, and inline formatting.
// It lives in a separate package because its WidgetState is considerably
// heavier than core widgets and its dependencies (clipboard, IME, undo stack)
// should not burden the framework core.
//
// The Document type is the serializable user-model value; the editor's
// internal state (cursor, selection, undo stack) lives in WidgetState.
package richtext
