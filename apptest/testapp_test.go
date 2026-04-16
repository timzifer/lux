//go:build nogui

package apptest

import (
	"fmt"
	"testing"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/app"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/button"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/form"
	"github.com/timzifer/lux/ui/layout"
)

// ── Counter App ─────────────────────────────────────────────────

type counterModel struct {
	Count int
}

type incrMsg struct{}
type decrMsg struct{}

func counterUpdate(m counterModel, msg any) counterModel {
	switch msg.(type) {
	case incrMsg:
		m.Count++
	case decrMsg:
		m.Count--
	}
	return m
}

func counterView(m counterModel) ui.Element {
	return layout.Column(
		display.Text(fmt.Sprintf("Count: %d", m.Count)),
		layout.Row(
			button.Text("−", func() { app.Send(decrMsg{}) }),
			button.Text("+", func() { app.Send(incrMsg{}) }),
		),
	)
}

func TestCounterClickIncrement(t *testing.T) {
	ta := New(counterModel{Count: 0}, counterUpdate, counterView)
	defer ta.Close()

	// Verify initial state.
	countEl := ta.Query(ByText("Count: 0"))
	countEl.AssertExists(t)
	countEl.AssertText(t, "Count: 0")

	// Click the + button.
	plusBtn := ta.Query(ByLabel("+"))
	plusBtn.AssertExists(t)
	plusBtn.AssertRole(t, a11y.RoleButton)
	plusBtn.Click()

	// Verify the model updated.
	if ta.Model().Count != 1 {
		t.Errorf("expected count 1, got %d", ta.Model().Count)
	}

	// Verify the UI updated.
	ta.Query(ByText("Count: 1")).AssertExists(t)
}

func TestCounterClickDecrement(t *testing.T) {
	ta := New(counterModel{Count: 5}, counterUpdate, counterView)
	defer ta.Close()

	ta.Query(ByText("Count: 5")).AssertExists(t)

	minusBtn := ta.Query(ByLabel("−"))
	minusBtn.AssertExists(t)
	minusBtn.Click()

	if ta.Model().Count != 4 {
		t.Errorf("expected count 4, got %d", ta.Model().Count)
	}
	ta.Query(ByText("Count: 4")).AssertExists(t)
}

func TestCounterMultipleClicks(t *testing.T) {
	ta := New(counterModel{Count: 0}, counterUpdate, counterView)
	defer ta.Close()

	plusBtn := ta.Query(ByLabel("+"))
	plusBtn.Click()
	plusBtn = ta.Query(ByLabel("+")) // re-query after tree rebuild
	plusBtn.Click()
	plusBtn = ta.Query(ByLabel("+"))
	plusBtn.Click()

	if ta.Model().Count != 3 {
		t.Errorf("expected count 3, got %d", ta.Model().Count)
	}
}

// ── Checkbox App ────────────────────────────────────────────────

type checkboxModel struct {
	Agreed bool
}

func checkboxUpdate(m checkboxModel, msg any) checkboxModel {
	switch v := msg.(type) {
	case toggleMsg:
		m.Agreed = v.checked
	}
	return m
}

type toggleMsg struct{ checked bool }

func checkboxView(m checkboxModel) ui.Element {
	return layout.Column(
		form.NewCheckbox("I agree", m.Agreed, func(checked bool) {
			app.Send(toggleMsg{checked})
		}),
		display.Text(fmt.Sprintf("Agreed: %v", m.Agreed)),
	)
}

func TestCheckboxToggle(t *testing.T) {
	ta := New(checkboxModel{Agreed: false}, checkboxUpdate, checkboxView)
	defer ta.Close()

	cb := ta.Query(ByRole(a11y.RoleCheckbox))
	cb.AssertExists(t)
	cb.AssertNotChecked(t)
	cb.AssertText(t, "I agree")

	// Click to check.
	cb.Click()

	if !ta.Model().Agreed {
		t.Error("expected Agreed=true after click")
	}

	// Re-query and verify checked state.
	cb = ta.Query(ByRole(a11y.RoleCheckbox))
	cb.AssertChecked(t)
}

// ── Query Tests ─────────────────────────────────────────────────

func TestQueryAll(t *testing.T) {
	ta := New(counterModel{Count: 0}, counterUpdate, counterView)
	defer ta.Close()

	buttons := ta.QueryAll(ByRole(a11y.RoleButton))
	if len(buttons) != 2 {
		t.Errorf("expected 2 buttons, got %d", len(buttons))
	}
}

func TestQueryByLabelAndRole(t *testing.T) {
	ta := New(counterModel{Count: 0}, counterUpdate, counterView)
	defer ta.Close()

	el := ta.Query(ByLabelAndRole("+", a11y.RoleButton))
	el.AssertExists(t)
	el.AssertText(t, "+")
	el.AssertRole(t, a11y.RoleButton)
}

func TestQueryNotFound(t *testing.T) {
	ta := New(counterModel{Count: 0}, counterUpdate, counterView)
	defer ta.Close()

	el := ta.Query(ByLabel("nonexistent"))
	if el != nil {
		t.Error("expected nil for nonexistent element")
	}
}

// ── Send Tests ──────────────────────────────────────────────────

func TestSendMessage(t *testing.T) {
	ta := New(counterModel{Count: 0}, counterUpdate, counterView)
	defer ta.Close()

	ta.Send(incrMsg{})
	ta.Step()

	if ta.Model().Count != 1 {
		t.Errorf("expected count 1 after Send+Step, got %d", ta.Model().Count)
	}
}

// ── StepUntilStable Tests ───────────────────────────────────────

func TestStepUntilStable(t *testing.T) {
	ta := New(counterModel{Count: 0}, counterUpdate, counterView)
	defer ta.Close()

	ta.Send(incrMsg{})
	ta.StepUntilStable()

	if ta.Model().Count != 1 {
		t.Errorf("expected count 1 after StepUntilStable, got %d", ta.Model().Count)
	}
}

// ── WithSize Tests ──────────────────────────────────────────────

func TestWithSize(t *testing.T) {
	ta := New(counterModel{Count: 0}, counterUpdate, counterView, WithSize(400, 300))
	defer ta.Close()

	// Just verify it doesn't panic and produces valid output.
	ta.Query(ByText("Count: 0")).AssertExists(t)
}

// ── Selector Matches Tests ──────────────────────────────────────

func TestSelectorByText(t *testing.T) {
	ta := New(counterModel{Count: 42}, counterUpdate, counterView)
	defer ta.Close()

	// ByText should match substring.
	el := ta.Query(ByText("Count:"))
	el.AssertExists(t)
	if el.Text() != "Count: 42" {
		t.Errorf("expected 'Count: 42', got %q", el.Text())
	}
}

// ── Element State Tests ─────────────────────────────────────────

func TestElementEnabled(t *testing.T) {
	ta := New(counterModel{Count: 0}, counterUpdate, counterView)
	defer ta.Close()

	btn := ta.Query(ByLabel("+"))
	btn.AssertEnabled(t)
}

func TestDisabledCheckbox(t *testing.T) {
	view := func(_ struct{}) ui.Element {
		return form.CheckboxDisabled("Accept", false)
	}
	ta := New(struct{}{}, func(m struct{}, msg any) struct{} { return m }, view)
	defer ta.Close()

	cb := ta.Query(ByRole(a11y.RoleCheckbox))
	cb.AssertExists(t)
	cb.AssertText(t, "Accept")
	// Note: Checkbox.WalkAccess doesn't report Disabled state yet.
	// The disabled checkbox has no "activate" action.
	if len(cb.node.Node.Actions) != 0 {
		t.Error("disabled checkbox should have no actions")
	}
}
