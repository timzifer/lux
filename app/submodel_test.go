//go:build nogui

package app

import "testing"

type parentModel struct {
	Name    string
	Counter childCounter
	Other   childCounter
}

type childCounter struct {
	Count int
}

type childIncrMsg struct{}
type childDecrMsg struct{}

func childUpdate(m childCounter, msg Msg) childCounter {
	switch msg.(type) {
	case childIncrMsg:
		m.Count++
	case childDecrMsg:
		m.Count--
	}
	return m
}

func TestDelegateForwardsToChild(t *testing.T) {
	sm := SubModel[parentModel, childCounter]{
		Get:    func(p parentModel) childCounter { return p.Counter },
		Set:    func(p parentModel, c childCounter) parentModel { p.Counter = c; return p },
		Update: childUpdate,
	}

	p := parentModel{Name: "test", Counter: childCounter{Count: 5}}
	p = Delegate(sm, p, childIncrMsg{})

	if p.Counter.Count != 6 {
		t.Errorf("Counter.Count = %d, want 6", p.Counter.Count)
	}
	if p.Name != "test" {
		t.Errorf("Name changed unexpectedly: %q", p.Name)
	}
}

func TestDelegateDecrement(t *testing.T) {
	sm := SubModel[parentModel, childCounter]{
		Get:    func(p parentModel) childCounter { return p.Counter },
		Set:    func(p parentModel, c childCounter) parentModel { p.Counter = c; return p },
		Update: childUpdate,
	}

	p := parentModel{Counter: childCounter{Count: 3}}
	p = Delegate(sm, p, childDecrMsg{})

	if p.Counter.Count != 2 {
		t.Errorf("Counter.Count = %d, want 2", p.Counter.Count)
	}
}

func TestDelegateMultipleSubModels(t *testing.T) {
	sm1 := SubModel[parentModel, childCounter]{
		Get:    func(p parentModel) childCounter { return p.Counter },
		Set:    func(p parentModel, c childCounter) parentModel { p.Counter = c; return p },
		Update: childUpdate,
	}
	sm2 := SubModel[parentModel, childCounter]{
		Get:    func(p parentModel) childCounter { return p.Other },
		Set:    func(p parentModel, c childCounter) parentModel { p.Other = c; return p },
		Update: childUpdate,
	}

	p := parentModel{}
	p = Delegate(sm1, p, childIncrMsg{})
	p = Delegate(sm1, p, childIncrMsg{})
	p = Delegate(sm2, p, childIncrMsg{})

	if p.Counter.Count != 2 {
		t.Errorf("Counter.Count = %d, want 2", p.Counter.Count)
	}
	if p.Other.Count != 1 {
		t.Errorf("Other.Count = %d, want 1", p.Other.Count)
	}
}

type asyncChildDoneMsg struct{ Value string }

func childUpdateWithCmd(m childCounter, msg Msg) (childCounter, Cmd) {
	switch msg.(type) {
	case childIncrMsg:
		m.Count++
		return m, func() Msg {
			return asyncChildDoneMsg{Value: "incremented"}
		}
	case childDecrMsg:
		m.Count--
		return m, nil
	}
	return m, nil
}

func TestDelegateWithCmdReturnsCmd(t *testing.T) {
	sm := SubModelWithCmd[parentModel, childCounter]{
		Get:    func(p parentModel) childCounter { return p.Counter },
		Set:    func(p parentModel, c childCounter) parentModel { p.Counter = c; return p },
		Update: childUpdateWithCmd,
	}

	p := parentModel{Counter: childCounter{Count: 0}}
	p, cmd := DelegateWithCmd(sm, p, childIncrMsg{})

	if p.Counter.Count != 1 {
		t.Errorf("Counter.Count = %d, want 1", p.Counter.Count)
	}
	if cmd == nil {
		t.Fatal("expected non-nil Cmd")
	}
	result := cmd()
	if done, ok := result.(asyncChildDoneMsg); !ok || done.Value != "incremented" {
		t.Errorf("Cmd result = %v, want asyncChildDoneMsg{incremented}", result)
	}
}

func TestDelegateWithCmdNilCmd(t *testing.T) {
	sm := SubModelWithCmd[parentModel, childCounter]{
		Get:    func(p parentModel) childCounter { return p.Counter },
		Set:    func(p parentModel, c childCounter) parentModel { p.Counter = c; return p },
		Update: childUpdateWithCmd,
	}

	p := parentModel{Counter: childCounter{Count: 5}}
	p, cmd := DelegateWithCmd(sm, p, childDecrMsg{})

	if p.Counter.Count != 4 {
		t.Errorf("Counter.Count = %d, want 4", p.Counter.Count)
	}
	if cmd != nil {
		t.Error("expected nil Cmd for decrement")
	}
}
