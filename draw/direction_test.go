package draw

import "testing"

func TestInsetsResolveLTR(t *testing.T) {
	ins := Insets{Start: 10, End: 20, Top: 5, Bottom: 5}
	left, right := ins.Resolve(DirLTR)
	if left != 10 || right != 20 {
		t.Errorf("LTR: got left=%v, right=%v; want 10, 20", left, right)
	}
}

func TestInsetsResolveRTL(t *testing.T) {
	ins := Insets{Start: 10, End: 20, Top: 5, Bottom: 5}
	left, right := ins.Resolve(DirRTL)
	if left != 20 || right != 10 {
		t.Errorf("RTL: got left=%v, right=%v; want 20, 10", left, right)
	}
}

func TestInsetsResolveFallbackToLeftRight(t *testing.T) {
	ins := Insets{Left: 15, Right: 25}
	left, right := ins.Resolve(DirLTR)
	if left != 15 || right != 25 {
		t.Errorf("fallback LTR: got left=%v, right=%v; want 15, 25", left, right)
	}
	left, right = ins.Resolve(DirRTL)
	if left != 15 || right != 25 {
		t.Errorf("fallback RTL: got left=%v, right=%v; want 15, 25 (no Start/End set)", left, right)
	}
}

func TestInsetsResolveStartEndOverridesLeftRight(t *testing.T) {
	ins := Insets{Left: 100, Right: 200, Start: 10, End: 20}
	left, right := ins.Resolve(DirLTR)
	if left != 10 || right != 20 {
		t.Errorf("Start/End should override Left/Right: got left=%v, right=%v", left, right)
	}
}

func TestLayoutDirectionConstants(t *testing.T) {
	if DirLTR != 0 {
		t.Errorf("DirLTR should be 0, got %d", DirLTR)
	}
	if DirRTL != 1 {
		t.Errorf("DirRTL should be 1, got %d", DirRTL)
	}
}
