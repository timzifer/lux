package chart

import "github.com/timzifer/lux/draw"

// DataPoint is a single (X, Y) observation.
type DataPoint struct {
	X float64
	Y float64
}

// Series is a named, styled collection of data points.
type Series struct {
	Name   string
	Points []DataPoint
	Color  draw.Color // zero value = auto-assign from palette
}

// PieSlice is a single segment for a pie chart.
type PieSlice struct {
	Label string
	Value float64
	Color draw.Color // zero value = auto-assign from palette
}

// SeriesHit describes which point the user is hovering.
type SeriesHit struct {
	SeriesIndex int
	PointIndex  int
	DataPoint   DataPoint
	ScreenX     float32
	ScreenY     float32
}
