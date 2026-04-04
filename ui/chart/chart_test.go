package chart

import (
	"math"
	"testing"

	"github.com/timzifer/lux/draw"
)

// ── niceNum ─────────────────────────────────────────────────────

func TestNiceNum(t *testing.T) {
	tests := []struct {
		x     float64
		round bool
		want  float64
	}{
		{1.0, true, 1},
		{2.3, true, 2},
		{4.5, true, 5},
		{7.8, true, 10},
		{0.07, true, 0.1},
		{35, true, 50},
		{0, true, 0},
	}
	for _, tc := range tests {
		got := niceNum(tc.x, tc.round)
		if math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("niceNum(%v, %v) = %v, want %v", tc.x, tc.round, got, tc.want)
		}
	}
}

// ── computeTicks ────────────────────────────────────────────────

func TestComputeTicks(t *testing.T) {
	ticks := computeTicks(0, 100, 6)
	if len(ticks) == 0 {
		t.Fatal("expected ticks, got none")
	}
	// First tick should be <= 0.
	if ticks[0] > 0 {
		t.Errorf("first tick %v > 0", ticks[0])
	}
	// Last tick should be >= 100.
	if ticks[len(ticks)-1] < 100 {
		t.Errorf("last tick %v < 100", ticks[len(ticks)-1])
	}
	// Ticks should be ascending.
	for i := 1; i < len(ticks); i++ {
		if ticks[i] <= ticks[i-1] {
			t.Errorf("ticks not ascending at %d: %v <= %v", i, ticks[i], ticks[i-1])
		}
	}
}

func TestComputeTicksEqual(t *testing.T) {
	ticks := computeTicks(5, 5, 6)
	if len(ticks) != 1 {
		t.Errorf("expected 1 tick for equal min/max, got %d", len(ticks))
	}
}

// ── transform ───────────────────────────────────────────────────

func TestTransformToScreen(t *testing.T) {
	tr := transform{
		xMin: 0, xMax: 100,
		yMin: 0, yMax: 100,
		plot: draw.R(50, 10, 200, 100),
	}

	// Bottom-left corner.
	sp := tr.toScreen(DataPoint{X: 0, Y: 0})
	if math.Abs(float64(sp.X-50)) > 0.5 || math.Abs(float64(sp.Y-110)) > 0.5 {
		t.Errorf("(0,0) → (%v,%v), want (50,110)", sp.X, sp.Y)
	}

	// Top-right corner.
	sp = tr.toScreen(DataPoint{X: 100, Y: 100})
	if math.Abs(float64(sp.X-250)) > 0.5 || math.Abs(float64(sp.Y-10)) > 0.5 {
		t.Errorf("(100,100) → (%v,%v), want (250,10)", sp.X, sp.Y)
	}

	// Center.
	sp = tr.toScreen(DataPoint{X: 50, Y: 50})
	if math.Abs(float64(sp.X-150)) > 0.5 || math.Abs(float64(sp.Y-60)) > 0.5 {
		t.Errorf("(50,50) → (%v,%v), want (150,60)", sp.X, sp.Y)
	}
}

func TestTransformFromScreen(t *testing.T) {
	tr := transform{
		xMin: 0, xMax: 100,
		yMin: 0, yMax: 100,
		plot: draw.R(50, 10, 200, 100),
	}

	dp := tr.fromScreen(draw.Pt(150, 60))
	if math.Abs(dp.X-50) > 0.5 || math.Abs(dp.Y-50) > 0.5 {
		t.Errorf("fromScreen(150,60) = (%v,%v), want (50,50)", dp.X, dp.Y)
	}
}

// ── RingBuffer ──────────────────────────────────────────────────

func TestRingBufferPush(t *testing.T) {
	rb := NewRingBuffer(3)
	rb.Push(DataPoint{X: 1, Y: 10})
	rb.Push(DataPoint{X: 2, Y: 20})
	rb.Push(DataPoint{X: 3, Y: 30})

	if rb.Len() != 3 {
		t.Errorf("Len() = %d, want 3", rb.Len())
	}

	// Overflow: oldest should be dropped.
	rb.Push(DataPoint{X: 4, Y: 40})
	if rb.Len() != 3 {
		t.Errorf("Len() = %d, want 3 after overflow", rb.Len())
	}

	s := rb.Slice()
	if s[0].X != 2 || s[1].X != 3 || s[2].X != 4 {
		t.Errorf("Slice() = %v, want [{2,20},{3,30},{4,40}]", s)
	}
}

func TestRingBufferSliceOrder(t *testing.T) {
	rb := NewRingBuffer(4)
	for i := 0; i < 10; i++ {
		rb.Push(DataPoint{X: float64(i), Y: float64(i * 10)})
	}
	s := rb.Slice()
	if len(s) != 4 {
		t.Fatalf("Slice() len = %d, want 4", len(s))
	}
	// Should contain the last 4 pushes: 6, 7, 8, 9.
	for i, p := range s {
		want := float64(i + 6)
		if p.X != want {
			t.Errorf("Slice()[%d].X = %v, want %v", i, p.X, want)
		}
	}
}

// ── nearestPoint ────────────────────────────────────────────────

func TestNearestPointBinaryX(t *testing.T) {
	points := []DataPoint{
		{X: 1}, {X: 3}, {X: 5}, {X: 7}, {X: 9},
	}
	tests := []struct {
		targetX float64
		want    int
	}{
		{1, 0},
		{3, 1},
		{4, 2},   // equidistant — binary search rounds to 5
		{3.9, 1}, // closer to 3 than 5
		{9, 4},
		{0, 0},
		{10, 4},
	}
	for _, tc := range tests {
		got := nearestPointBinaryX(points, tc.targetX)
		if got != tc.want {
			t.Errorf("nearestPointBinaryX(target=%v) = %d, want %d", tc.targetX, got, tc.want)
		}
	}
}

func TestNearestPointEmpty(t *testing.T) {
	got := nearestPointBinaryX(nil, 5)
	if got != -1 {
		t.Errorf("nearestPointBinaryX(nil) = %d, want -1", got)
	}
}

// ── Viewport ────────────────────────────────────────────────────

func TestViewportPan(t *testing.T) {
	vp := Viewport{XMin: 0, XMax: 10, YMin: 0, YMax: 10}
	vp.Pan(5, -3)
	if vp.XMin != 5 || vp.XMax != 15 || vp.YMin != -3 || vp.YMax != 7 {
		t.Errorf("Pan(5,-3) = %+v, want {5,15,-3,7}", vp)
	}
}

func TestViewportZoom(t *testing.T) {
	vp := Viewport{XMin: 0, XMax: 10, YMin: 0, YMax: 10}
	// Zoom in by 50% around center (5,5).
	vp.Zoom(5, 5, 0.5)
	if math.Abs(vp.XMin-2.5) > 0.01 || math.Abs(vp.XMax-7.5) > 0.01 {
		t.Errorf("Zoom(0.5) X = [%v,%v], want [2.5,7.5]", vp.XMin, vp.XMax)
	}
}

// ── AutoScrollViewport ──────────────────────────────────────────

func TestAutoScrollViewport(t *testing.T) {
	data := make([]DataPoint, 100)
	for i := range data {
		data[i] = DataPoint{X: float64(i), Y: float64(i % 10)}
	}
	vp := AutoScrollViewport(data, 20)
	if vp.XMax != 99 {
		t.Errorf("XMax = %v, want 99", vp.XMax)
	}
	if vp.XMin != 79 {
		t.Errorf("XMin = %v, want 79", vp.XMin)
	}
}

func TestAutoScrollViewportEmpty(t *testing.T) {
	vp := AutoScrollViewport(nil, 10)
	if vp.XMax != 10 {
		t.Errorf("empty data: XMax = %v, want 10", vp.XMax)
	}
}

// ── dataRange ───────────────────────────────────────────────────

func TestDataRange(t *testing.T) {
	pts := []DataPoint{{X: 3, Y: 10}, {X: 1, Y: 30}, {X: 5, Y: 20}}
	mn, mx := dataRange(pts, 'x')
	if mn != 1 || mx != 5 {
		t.Errorf("X range = [%v,%v], want [1,5]", mn, mx)
	}
	mn, mx = dataRange(pts, 'y')
	if mn != 10 || mx != 30 {
		t.Errorf("Y range = [%v,%v], want [10,30]", mn, mx)
	}
}

func TestDataRangeEmpty(t *testing.T) {
	mn, mx := dataRange(nil, 'x')
	if mn != 0 || mx != 1 {
		t.Errorf("empty range = [%v,%v], want [0,1]", mn, mx)
	}
}

// ── formatTick ──────────────────────────────────────────────────

func TestFormatTick(t *testing.T) {
	tests := []struct {
		v    float64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{100, "100"},
		{-5, "-5"},
		{2.5, "2.5"},
		{0.33, "0.33"},
	}
	for _, tc := range tests {
		got := formatTick(tc.v)
		if got != tc.want {
			t.Errorf("formatTick(%v) = %q, want %q", tc.v, got, tc.want)
		}
	}
}
