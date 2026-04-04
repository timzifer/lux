package form

import "testing"

func TestTimeInput_ColumnCount(t *testing.T) {
	ti := TimeInput{Format: TimeFormatHHMM}
	if got := ti.columnCount(); got != 2 {
		t.Errorf("HHMM columnCount() = %d, want 2", got)
	}

	ti.Format = TimeFormatHHMMSS
	if got := ti.columnCount(); got != 3 {
		t.Errorf("HHMMSS columnCount() = %d, want 3", got)
	}

	ti.Format = TimeFormat12h
	if got := ti.columnCount(); got != 2 {
		t.Errorf("12h columnCount() = %d, want 2", got)
	}
}
