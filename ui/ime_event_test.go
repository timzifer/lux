//go:build nogui

package ui

import (
	"testing"

	"github.com/timzifer/lux/input"
)

func TestIMEComposeEvent(t *testing.T) {
	msg := input.IMEComposeMsg{Text: "こん", CursorStart: 0, CursorEnd: 2}
	ev := IMEComposeEvent(msg)

	if ev.Kind != EventIMECompose {
		t.Errorf("Kind = %d, want EventIMECompose", ev.Kind)
	}
	if ev.IMECompose == nil {
		t.Fatal("IMECompose is nil")
	}
	if ev.IMECompose.Text != "こん" {
		t.Errorf("IMECompose.Text = %q", ev.IMECompose.Text)
	}
}

func TestIMECommitEvent(t *testing.T) {
	msg := input.IMECommitMsg{Text: "今日は"}
	ev := IMECommitEvent(msg)

	if ev.Kind != EventIMECommit {
		t.Errorf("Kind = %d, want EventIMECommit", ev.Kind)
	}
	if ev.IMECommit == nil {
		t.Fatal("IMECommit is nil")
	}
	if ev.IMECommit.Text != "今日は" {
		t.Errorf("IMECommit.Text = %q", ev.IMECommit.Text)
	}
}

func TestInputStateComposeFields(t *testing.T) {
	is := InputState{
		Value:              "hello",
		ComposeText:        "にほん",
		ComposeCursorStart: 0,
		ComposeCursorEnd:   3,
	}
	if is.ComposeText != "にほん" {
		t.Errorf("ComposeText = %q", is.ComposeText)
	}
	if is.ComposeCursorStart != 0 || is.ComposeCursorEnd != 3 {
		t.Errorf("Compose cursor: %d-%d", is.ComposeCursorStart, is.ComposeCursorEnd)
	}
}
