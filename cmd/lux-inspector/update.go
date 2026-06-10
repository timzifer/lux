package main

import (
	"log"

	"github.com/timzifer/lux/app"
	"github.com/timzifer/lux/internal/vellum"
)

// Messages for the Inspector update loop.

// FrameReceivedMsg is sent when a new frame arrives from the Vellum client.
type FrameReceivedMsg struct {
	Frame *vellum.DecodedFrame
}

// ControlReceivedMsg is sent when a control message arrives.
type ControlReceivedMsg struct {
	Data []byte
}

// SelectWidgetMsg selects a widget in the tree panel.
type SelectWidgetMsg struct {
	UID uint64
}

// SetPanelMsg switches the active debug panel.
type SetPanelMsg struct {
	Panel Panel
}

// ToggleOverlayMsg toggles the layout overlay.
type ToggleOverlayMsg struct{}

// ToggleDirtyMsg toggles paint highlighting.
type ToggleDirtyMsg struct{}

// TogglePauseMsg pauses/resumes the frame stream.
type TogglePauseMsg struct{}

// startFrameListener returns a Cmd that reads the next frame from the client.
// A decode error skips the bad frame and keeps reading, so a single malformed
// frame doesn't tear down the stream. A nil result is only returned when the
// connection is gone, which stops the listen loop.
func startFrameListener(client *vellum.Client) app.Cmd {
	return func() app.Msg {
		for {
			data, err := client.NextFrame()
			if err != nil {
				log.Printf("lux-inspector: frame error: %v", err)
				return nil
			}
			frame, err := vellum.DecodeFrame(data)
			if err != nil {
				log.Printf("lux-inspector: decode error: %v", err)
				continue
			}
			return FrameReceivedMsg{Frame: frame}
		}
	}
}

// startControlListener returns a Cmd that reads control messages.
func startControlListener(client *vellum.Client) app.Cmd {
	return func() app.Msg {
		data, err := client.NextControl()
		if err != nil {
			return nil
		}
		return ControlReceivedMsg{Data: data}
	}
}

// inspectorUpdate handles messages for the Inspector (RFC-012 §6.2).
//
// The frame and control streams are pumped via blocking Cmds: each listener
// reads one message, returns it as a Msg, and is re-armed here so the next read
// starts immediately. Without this re-arming the Inspector would receive a
// single frame and then go silent — the cause of the blank remote view.
func inspectorUpdate(model InspectorModel, msg app.Msg) (InspectorModel, app.Cmd) {
	switch m := msg.(type) {
	case app.TickMsg:
		// Bootstrap the listeners exactly once. TickMsg is the first message
		// delivered after the loop is running, so app.Send/Cmd dispatch is live.
		if !model.started {
			model.started = true
			return model, app.Batch(
				startFrameListener(model.Client),
				startControlListener(model.Client),
			)
		}
		return model, nil

	case FrameReceivedMsg:
		// Always re-arm the frame listener, even while paused, so the stream
		// keeps draining and resumes instantly when unpaused.
		reArm := startFrameListener(model.Client)
		if model.Paused {
			return model, reArm
		}
		model.CurrentFrame = m.Frame
		if m.Frame.FrameInfo != nil {
			model.FrameInfo = m.Frame.FrameInfo
			// Append to frame history.
			metric := FrameMetric{
				FrameID:       m.Frame.FrameInfo.FrameID,
				FrameTime:     m.Frame.FrameInfo.FrameTime,
				UpdateTime:    m.Frame.FrameInfo.UpdateTime,
				ReconcileTime: m.Frame.FrameInfo.ReconcileTime,
				PaintTime:     m.Frame.FrameInfo.PaintTime,
				WidgetCount:   m.Frame.FrameInfo.WidgetCount,
			}
			if len(model.FrameHistory) >= maxFrameHistory {
				model.FrameHistory = append(model.FrameHistory[1:], metric)
			} else {
				model.FrameHistory = append(model.FrameHistory, metric)
			}
		}
		return model, reArm

	case ControlReceivedMsg:
		// Re-arm the control listener regardless of parse outcome.
		reArm := startControlListener(model.Client)
		if len(m.Data) == 0 {
			return model, reArm
		}
		opcode, payload, _, err := vellum.ReadOp(m.Data, 0)
		if err != nil {
			return model, reArm
		}
		switch opcode {
		case vellum.OpDebugWidgetTree:
			tree, err := vellum.DecodeDebugWidgetTree(payload)
			if err == nil {
				model.WidgetTree = tree
			}
		case vellum.OpDebugEventLog:
			evLog, err := vellum.DecodeDebugEventLog(payload)
			if err == nil {
				model.EventLog = append(model.EventLog, evLog.Events...)
				if len(model.EventLog) > maxEventLog {
					model.EventLog = model.EventLog[len(model.EventLog)-maxEventLog:]
				}
			}
		}
		return model, reArm

	case SelectWidgetMsg:
		model.SelectedUID = m.UID
		return model, nil

	case SetPanelMsg:
		model.ActivePanel = m.Panel
		return model, nil

	case ToggleOverlayMsg:
		model.ShowOverlay = !model.ShowOverlay
		return model, nil

	case ToggleDirtyMsg:
		model.ShowDirty = !model.ShowDirty
		return model, nil

	case TogglePauseMsg:
		model.Paused = !model.Paused
		return model, nil
	}

	return model, nil
}
