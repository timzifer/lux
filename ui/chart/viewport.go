package chart

// Viewport tracks the visible data window for pan/zoom.
type Viewport struct {
	XMin, XMax float64
	YMin, YMax float64
}

// Pan shifts the viewport by a delta in data coordinates.
func (v *Viewport) Pan(dx, dy float64) {
	v.XMin += dx
	v.XMax += dx
	v.YMin += dy
	v.YMax += dy
}

// Zoom scales the viewport around a focal point by a factor.
// factor > 1 zooms out, factor < 1 zooms in.
func (v *Viewport) Zoom(focusX, focusY, factor float64) {
	v.XMin = focusX + (v.XMin-focusX)*factor
	v.XMax = focusX + (v.XMax-focusX)*factor
	v.YMin = focusY + (v.YMin-focusY)*factor
	v.YMax = focusY + (v.YMax-focusY)*factor
}

// RingBuffer is a fixed-capacity circular buffer for streaming data points.
type RingBuffer struct {
	points   []DataPoint
	head     int
	count    int
	capacity int
}

// NewRingBuffer creates a RingBuffer with the given capacity.
func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = 256
	}
	return &RingBuffer{
		points:   make([]DataPoint, capacity),
		capacity: capacity,
	}
}

// Push appends a data point, overwriting the oldest if at capacity.
func (rb *RingBuffer) Push(p DataPoint) {
	idx := (rb.head + rb.count) % rb.capacity
	rb.points[idx] = p
	if rb.count < rb.capacity {
		rb.count++
	} else {
		rb.head = (rb.head + 1) % rb.capacity
	}
}

// Len returns the number of points currently stored.
func (rb *RingBuffer) Len() int { return rb.count }

// Slice returns the points in chronological order.
func (rb *RingBuffer) Slice() []DataPoint {
	out := make([]DataPoint, rb.count)
	for i := 0; i < rb.count; i++ {
		out[i] = rb.points[(rb.head+i)%rb.capacity]
	}
	return out
}

// AutoScrollViewport returns a viewport that shows the last windowSize
// X-units of data, with Y auto-ranged to fit visible points.
func AutoScrollViewport(data []DataPoint, windowSize float64) Viewport {
	if len(data) == 0 {
		return Viewport{XMin: 0, XMax: windowSize, YMin: -1, YMax: 1}
	}
	xMax := data[len(data)-1].X
	xMin := xMax - windowSize

	// Find Y range for visible points.
	yMin, yMax := data[len(data)-1].Y, data[len(data)-1].Y
	for i := len(data) - 1; i >= 0; i-- {
		if data[i].X < xMin {
			break
		}
		if data[i].Y < yMin {
			yMin = data[i].Y
		}
		if data[i].Y > yMax {
			yMax = data[i].Y
		}
	}
	// Add 10% padding to Y.
	pad := (yMax - yMin) * 0.1
	if pad == 0 {
		pad = 1
	}
	return Viewport{XMin: xMin, XMax: xMax, YMin: yMin - pad, YMax: yMax + pad}
}
