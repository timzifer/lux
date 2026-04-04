package chart

import "math"

// AxisKind distinguishes axis placement.
type AxisKind uint8

const (
	AxisBottom AxisKind = iota
	AxisLeft
)

// Axis configures one chart axis.
type Axis struct {
	Label     string              // axis title
	Min       *float64            // nil = auto-range from data
	Max       *float64            // nil = auto-range from data
	TickCount int                 // hint; 0 = auto (~5-8)
	Format    func(float64) string // tick label formatter; nil = default
	GridLines bool               // draw grid lines from this axis
}

// niceNum finds a "nice" number approximately equal to x.
// If round is true, it rounds; otherwise it takes the ceiling.
func niceNum(x float64, round bool) float64 {
	if x == 0 {
		return 0
	}
	exp := math.Floor(math.Log10(math.Abs(x)))
	frac := x / math.Pow(10, exp)

	var nice float64
	if round {
		switch {
		case frac < 1.5:
			nice = 1
		case frac < 3:
			nice = 2
		case frac < 7:
			nice = 5
		default:
			nice = 10
		}
	} else {
		switch {
		case frac <= 1:
			nice = 1
		case frac <= 2:
			nice = 2
		case frac <= 5:
			nice = 5
		default:
			nice = 10
		}
	}
	return nice * math.Pow(10, exp)
}

// computeTicks generates "nice" tick positions for a given data range.
func computeTicks(min, max float64, count int) []float64 {
	if count <= 0 {
		count = 6
	}
	if min == max {
		return []float64{min}
	}
	if min > max {
		min, max = max, min
	}

	rang := niceNum(max-min, false)
	step := niceNum(rang/float64(count-1), true)
	if step == 0 {
		return []float64{min}
	}

	lo := math.Floor(min/step) * step
	hi := math.Ceil(max/step) * step

	var ticks []float64
	for v := lo; v <= hi+step*0.5; v += step {
		ticks = append(ticks, v)
	}
	return ticks
}

// dataRange computes the min/max of a slice of data points.
func dataRange(points []DataPoint, axis byte) (float64, float64) {
	if len(points) == 0 {
		return 0, 1
	}
	val := func(p DataPoint) float64 {
		if axis == 'x' {
			return p.X
		}
		return p.Y
	}
	mn := val(points[0])
	mx := mn
	for _, p := range points[1:] {
		v := val(p)
		if v < mn {
			mn = v
		}
		if v > mx {
			mx = v
		}
	}
	if mn == mx {
		mn -= 1
		mx += 1
	}
	return mn, mx
}

// multiSeriesRange computes the combined min/max across multiple series.
func multiSeriesRange(series []Series, axis byte) (float64, float64) {
	if len(series) == 0 {
		return 0, 1
	}
	mn, mx := dataRange(series[0].Points, axis)
	for _, s := range series[1:] {
		sMin, sMax := dataRange(s.Points, axis)
		if sMin < mn {
			mn = sMin
		}
		if sMax > mx {
			mx = sMax
		}
	}
	return mn, mx
}

// formatTick is the default tick label formatter.
func formatTick(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e9 {
		return strconvFormatFloat(v)
	}
	return strconvFormatFloat(v)
}

func strconvFormatFloat(v float64) string {
	// Use compact formatting: no trailing zeros.
	s := math.Abs(v)
	switch {
	case s == 0:
		return "0"
	case s >= 1 && s == math.Trunc(s):
		return formatInt(v)
	default:
		return formatDecimal(v)
	}
}

func formatInt(v float64) string {
	neg := ""
	if v < 0 {
		neg = "-"
		v = -v
	}
	n := int64(v)
	if n == 0 {
		return neg + "0"
	}
	// Simple int-to-string.
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return neg + string(buf[i:])
}

func formatDecimal(v float64) string {
	neg := ""
	if v < 0 {
		neg = "-"
		v = -v
	}
	// Format to 2 decimal places, strip trailing zeros.
	whole := int64(v)
	frac := v - float64(whole)
	f := int64(math.Round(frac * 100))
	if f >= 100 {
		whole++
		f -= 100
	}

	ws := formatInt(float64(whole))
	if f == 0 {
		return neg + ws
	}
	d1 := byte('0' + f/10)
	d2 := byte('0' + f%10)
	if d2 == '0' {
		return neg + ws + "." + string(d1)
	}
	return neg + ws + "." + string(d1) + string(d2)
}
