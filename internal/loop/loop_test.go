package loop

import (
	"testing"
	"time"
)

func TestClampDt_Normal(t *testing.T) {
	l := New()
	dt := l.ClampDt(16 * time.Millisecond)
	if dt != 16*time.Millisecond {
		t.Errorf("expected 16ms, got %v", dt)
	}
}

func TestClampDt_ExceedsMax(t *testing.T) {
	l := New()
	// Simulate a 5-second freeze.
	dt := l.ClampDt(5 * time.Second)
	if dt != DefaultMaxFrameDelta {
		t.Errorf("expected %v, got %v", DefaultMaxFrameDelta, dt)
	}
}

func TestClampDt_Zero(t *testing.T) {
	l := New()
	dt := l.ClampDt(0)
	if dt <= 0 {
		t.Errorf("dt must be > 0, got %v", dt)
	}
}

func TestClampDt_Negative(t *testing.T) {
	l := New()
	dt := l.ClampDt(-10 * time.Millisecond)
	if dt <= 0 {
		t.Errorf("dt must be > 0, got %v", dt)
	}
}

func TestClampDt_CustomMax(t *testing.T) {
	l := New(WithMaxFrameDelta(50 * time.Millisecond))
	dt := l.ClampDt(200 * time.Millisecond)
	if dt != 50*time.Millisecond {
		t.Errorf("expected 50ms, got %v", dt)
	}
}

func TestClampDt_ExactlyMax(t *testing.T) {
	l := New()
	dt := l.ClampDt(DefaultMaxFrameDelta)
	if dt != DefaultMaxFrameDelta {
		t.Errorf("expected %v, got %v", DefaultMaxFrameDelta, dt)
	}
}

func TestSendAndDrain(t *testing.T) {
	l := New()

	// Send 3 messages.
	l.Send("a")
	l.Send("b")
	l.Send("c")

	var received []string
	l.DrainMessages(func(msg any) bool {
		received = append(received, msg.(string))
		return true
	})

	if len(received) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(received))
	}
	if received[0] != "a" || received[1] != "b" || received[2] != "c" {
		t.Errorf("unexpected order: %v", received)
	}
}

func TestTrySend_Full(t *testing.T) {
	l := &Loop{
		msgCh:         make(chan any, 1),
		maxFrameDelta: DefaultMaxFrameDelta,
	}
	// First send succeeds.
	if !l.TrySend("a") {
		t.Error("first TrySend should succeed")
	}
	// Second send fails (buffer full).
	if l.TrySend("b") {
		t.Error("second TrySend should fail (buffer full)")
	}
}

func TestDrainMessages_Empty(t *testing.T) {
	l := New()
	changed := l.DrainMessages(func(msg any) bool {
		t.Error("should not be called")
		return true
	})
	if changed {
		t.Error("should return false when no messages")
	}
}

func TestDrainMessages_ReturnsChanged(t *testing.T) {
	l := New()
	l.Send("x")

	changed := l.DrainMessages(func(msg any) bool {
		return false // update returns no change
	})
	if changed {
		t.Error("should return false when update returns false")
	}

	l.Send("y")
	changed = l.DrainMessages(func(msg any) bool {
		return true // update returns change
	})
	if !changed {
		t.Error("should return true when update returns true")
	}
}
