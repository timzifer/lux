package form

import (
	"testing"
	"time"
)

func TestDaysInMonth(t *testing.T) {
	tests := []struct {
		year  int
		month time.Month
		want  int
	}{
		{2024, time.January, 31},
		{2024, time.February, 29},  // leap year
		{2025, time.February, 28},  // non-leap
		{2024, time.April, 30},
		{2024, time.December, 31},
	}
	for _, tt := range tests {
		got := DaysInMonth(tt.year, tt.month)
		if got != tt.want {
			t.Errorf("DaysInMonth(%d, %s) = %d, want %d", tt.year, tt.month, got, tt.want)
		}
	}
}

func TestMonthNames(t *testing.T) {
	if MonthNames[1] != "Jan" {
		t.Errorf("MonthNames[1] = %q, want %q", MonthNames[1], "Jan")
	}
	if MonthNames[12] != "Dec" {
		t.Errorf("MonthNames[12] = %q, want %q", MonthNames[12], "Dec")
	}
}
