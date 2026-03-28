// Package vellum implements the Vellum remote-rendering protocol (RFC-011/RFC-012).
//
// It provides binary serialization of draw.Canvas operations, Unix-socket
// transport, and debug extensions for the Widget Inspector.
package vellum

// Opcodes for the Vellum canvas-command protocol (RFC-012 §7.2).
// Each opcode maps 1:1 to a draw.Canvas method or a control/debug message.
const (
	// Frame control.
	OpBeginFrame byte = 0x01
	OpEndFrame   byte = 0x02

	// Primitives (1:1 mapping to draw.Canvas).
	OpFillRect             byte = 0x10
	OpFillRoundRect        byte = 0x11
	OpFillRoundRectCorners byte = 0x12
	OpFillEllipse          byte = 0x13
	OpStrokeRect           byte = 0x14
	OpStrokeRoundRect      byte = 0x15
	OpStrokeRoundRectCorners byte = 0x16
	OpStrokeEllipse        byte = 0x17
	OpStrokeLine           byte = 0x18

	// Paths.
	OpFillPath   byte = 0x20
	OpStrokePath byte = 0x21

	// Text.
	OpDrawText       byte = 0x30
	OpDrawTextLayout byte = 0x31
	OpMeasureText    byte = 0x32

	// Images & Textures.
	OpDrawImage       byte = 0x40
	OpDrawImageScaled byte = 0x41
	OpDrawImageSlice  byte = 0x42
	OpDrawTexture     byte = 0x43

	// Shadows.
	OpDrawShadow byte = 0x50

	// Clipping & Transform.
	OpPushClip          byte = 0x60
	OpPushClipRoundRect byte = 0x61
	OpPushClipPath      byte = 0x62
	OpPopClip           byte = 0x63
	OpPushTransform     byte = 0x64
	OpPopTransform      byte = 0x65
	OpPushOffset        byte = 0x66
	OpPushScale         byte = 0x67

	// Effects.
	OpPushOpacity byte = 0x70
	OpPopOpacity  byte = 0x71
	OpPushBlur    byte = 0x72
	OpPopBlur     byte = 0x73
	OpPushLayer   byte = 0x74
	OpPopLayer    byte = 0x75

	// State.
	OpBounds  byte = 0x80
	OpDPR     byte = 0x81
	OpSave    byte = 0x82
	OpRestore byte = 0x83

	// Control (Channel 0).
	OpHandshake       byte = 0xC0
	OpAccessTreeUpdate byte = 0xC1

	// Debug extensions (0xD0–0xDF).
	OpDebugFrameInfo  byte = 0xD0
	OpDebugWidgetTree byte = 0xD1
	OpDebugEventLog   byte = 0xD2
)
