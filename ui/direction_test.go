//go:build nogui

package ui

import (
	"testing"

	"github.com/timzifer/lux/draw"
)

func TestSetAndGetDirection(t *testing.T) {
	// Default should be LTR.
	SetDirection(draw.DirLTR)
	if got := Direction(); got != draw.DirLTR {
		t.Errorf("Direction() = %d, want DirLTR", got)
	}

	SetDirection(draw.DirRTL)
	if got := Direction(); got != draw.DirRTL {
		t.Errorf("Direction() = %d, want DirRTL", got)
	}

	// Clean up.
	SetDirection(draw.DirLTR)
}

func TestInlineInsets(t *testing.T) {
	ins := InlineInsets(10, 20)
	if ins.Start != 10 || ins.End != 20 {
		t.Errorf("InlineInsets(10, 20) = %+v", ins)
	}
	if ins.Left != 0 || ins.Right != 0 {
		t.Errorf("InlineInsets should have zero Left/Right: %+v", ins)
	}
}

func TestBlockInsets(t *testing.T) {
	ins := BlockInsets(5, 15)
	if ins.Top != 5 || ins.Bottom != 15 {
		t.Errorf("BlockInsets(5, 15) = %+v", ins)
	}
}

func TestLogicalInsets(t *testing.T) {
	ins := LogicalInsets(1, 2, 3, 4)
	if ins.Top != 1 || ins.End != 2 || ins.Bottom != 3 || ins.Start != 4 {
		t.Errorf("LogicalInsets(1,2,3,4) = %+v", ins)
	}
}

func TestLogicalInsetsResolveLTR(t *testing.T) {
	ins := LogicalInsets(0, 20, 0, 10)
	left, right := ins.Resolve(draw.DirLTR)
	if left != 10 || right != 20 {
		t.Errorf("LTR: left=%v, right=%v; want 10, 20", left, right)
	}
}

func TestLogicalInsetsResolveRTL(t *testing.T) {
	ins := LogicalInsets(0, 20, 0, 10)
	left, right := ins.Resolve(draw.DirRTL)
	if left != 20 || right != 10 {
		t.Errorf("RTL: left=%v, right=%v; want 20, 10", left, right)
	}
}
