// Package loop implements the single-threaded app loop (RFC §3.3).
// It processes messages, calls update/view, and manages frame timing with dt-clamping.
package loop

import (
	"sync"
	"time"
)

const (
	// DefaultMaxFrameDelta prevents spiral-of-death after freezes (RFC §3.3).
	DefaultMaxFrameDelta = 100 * time.Millisecond

	// DefaultMsgChannelSize is the capacity of the message buffer.
	DefaultMsgChannelSize = 256
)

// Loop is the core app loop that processes messages and drives rendering.
type Loop struct {
	msgCh         chan any
	maxFrameDelta time.Duration

	mu      sync.Mutex
	running bool

	// wakeFn is called after a message is enqueued via Send/TrySend to
	// wake the platform event loop when it is blocking in idle mode.
	// Set once before goroutines are spawned; read concurrently (safe
	// because the write happens-before goroutine creation).
	wakeFn func()
}

// New creates a new Loop with the given options.
func New(opts ...Option) *Loop {
	l := &Loop{
		msgCh:         make(chan any, DefaultMsgChannelSize),
		maxFrameDelta: DefaultMaxFrameDelta,
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Option configures a Loop.
type Option func(*Loop)

// WithMaxFrameDelta overrides the default dt clamp.
func WithMaxFrameDelta(d time.Duration) Option {
	return func(l *Loop) {
		if d > 0 {
			l.maxFrameDelta = d
		}
	}
}

// SetWakeFunc registers a callback that is invoked after every successful
// Send/TrySend to wake the platform event loop from idle blocking.
// Must be called before any concurrent Send.
func (l *Loop) SetWakeFunc(fn func()) {
	l.wakeFn = fn
}

// Send enqueues a message. Thread-safe, never blocks (drops if full).
func (l *Loop) Send(msg any) {
	select {
	case l.msgCh <- msg:
		if l.wakeFn != nil {
			l.wakeFn()
		}
	default:
		// Channel full — drop message to avoid blocking.
		// This matches RFC's "blockiert nie" requirement.
		// Channel full — message dropped.
	}
}

// TrySend attempts to enqueue a message. Returns false if the channel is full.
func (l *Loop) TrySend(msg any) bool {
	select {
	case l.msgCh <- msg:
		if l.wakeFn != nil {
			l.wakeFn()
		}
		return true
	default:
		return false
	}
}

// FrameCallbacks are invoked each frame by the loop.
type FrameCallbacks struct {
	// Update processes a single message and returns the new model changed flag.
	Update func(msg any) bool

	// Render is called once per frame to execute GPU commands.
	Render func(dt time.Duration)
}

// DrainMessages processes all pending messages by calling update for each.
// Returns true if any message caused a model change.
func (l *Loop) DrainMessages(update func(msg any) bool) bool {
	changed := false
	for {
		select {
		case msg := <-l.msgCh:
			if update(msg) {
				changed = true
			}
		default:
			return changed
		}
	}
}

// ClampDt applies dt-clamping per RFC §3.3.
func (l *Loop) ClampDt(dt time.Duration) time.Duration {
	if dt <= 0 {
		return time.Millisecond // Never zero or negative.
	}
	if dt > l.maxFrameDelta {
		return l.maxFrameDelta
	}
	return dt
}

// MaxFrameDelta returns the configured maximum frame delta.
func (l *Loop) MaxFrameDelta() time.Duration {
	return l.maxFrameDelta
}
