package chart

import "math"

// ChartViewportMsg is sent when the user pans or zooms the chart.
type ChartViewportMsg struct {
	Viewport Viewport
}

// ChartHoverMsg is sent when the user hovers over a data point.
type ChartHoverMsg struct {
	Hit *SeriesHit // nil = mouse left the chart
}

// nearestPoint finds the closest data point to screenX within a series.
// Uses linear scan — suitable for up to ~10K points per series.
func nearestPoint(points []DataPoint, t transform, screenX, screenY float32) (int, float32) {
	if len(points) == 0 {
		return -1, math.MaxFloat32
	}

	bestIdx := 0
	bestDist := float32(math.MaxFloat32)
	for i, p := range points {
		sp := t.toScreen(p)
		dx := sp.X - screenX
		dy := sp.Y - screenY
		d := dx*dx + dy*dy
		if d < bestDist {
			bestDist = d
			bestIdx = i
		}
	}
	return bestIdx, float32(math.Sqrt(float64(bestDist)))
}

// nearestPointBinaryX finds the closest data point using binary search on X.
// Requires points to be sorted by X (typical for time-series).
func nearestPointBinaryX(points []DataPoint, targetX float64) int {
	if len(points) == 0 {
		return -1
	}
	lo, hi := 0, len(points)-1
	for lo < hi {
		mid := (lo + hi) / 2
		if points[mid].X < targetX {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	// Check neighbors.
	best := lo
	if lo > 0 && math.Abs(points[lo-1].X-targetX) < math.Abs(points[lo].X-targetX) {
		best = lo - 1
	}
	return best
}
