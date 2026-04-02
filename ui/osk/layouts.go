package osk

import "strings"

// RowsForState returns the key rows for the current OSK state.
func RowsForState(state *OSKState) [][]OSKKey {
	if state == nil {
		return alphaRows(false)
	}
	switch state.Mode {
	case ModeAlpha:
		return alphaRows(state.Shifted)
	case ModeNumPad:
		return numpadRows(state.Layout, state.Shifted)
	case ModeFull:
		return fullRows(state.Shifted)
	case ModeCondensed:
		return condensedRows(state.Shifted)
	default:
		return alphaRows(state.Shifted)
	}
}

// ── Alpha Layout (QWERTZ with number row) ───────────────────────

func alphaRows(shifted bool) [][]OSKKey {
	if shifted {
		return alphaRowsShifted()
	}
	return alphaRowsLower()
}

func alphaRowsLower() [][]OSKKey {
	return [][]OSKKey{
		charRow("1234567890", 1.0),
		charRow("qwertzuiop", 1.0),
		charRow("asdfghjkl", 1.0),
		shiftRow("yxcvbnm", false),
		bottomRow(),
	}
}

func alphaRowsShifted() [][]OSKKey {
	return [][]OSKKey{
		symbolRow("!@#$%^&*()"),
		charRow("QWERTZUIOP", 1.0),
		charRow("ASDFGHJKL", 1.0),
		shiftRow("YXCVBNM", true),
		bottomRow(),
	}
}

// ── NumPad Layout ───────────────────────────────────────────────

func numpadRows(layout OSKLayout, shifted bool) [][]OSKKey {
	switch layout {
	case OSKLayoutPin:
		return pinRows()
	case OSKLayoutIP:
		return ipRows()
	case OSKLayoutHex:
		return hexRows()
	case OSKLayoutPhone:
		return phoneRows()
	case OSKLayoutNumericInteger:
		return numericIntegerRows()
	default:
		return numericRows()
	}
}

func numericRows() [][]OSKKey {
	return [][]OSKKey{
		charRow("789", 1.0),
		charRow("456", 1.0),
		charRow("123", 1.0),
		{
			{Label: "±", Action: OSKActionSign, Width: 1.0},
			{Label: "0", Action: OSKActionChar, Width: 1.0, Char: '0'},
			{Label: ".", Action: OSKActionDecimal, Width: 1.0},
		},
	}
}

func numericIntegerRows() [][]OSKKey {
	return [][]OSKKey{
		charRow("789", 1.0),
		charRow("456", 1.0),
		charRow("123", 1.0),
		{
			{Label: "±", Action: OSKActionSign, Width: 1.0},
			{Label: "0", Action: OSKActionChar, Width: 1.0, Char: '0'},
			{Label: "⌫", Action: OSKActionBackspace, Width: 1.0},
		},
	}
}

func pinRows() [][]OSKKey {
	return [][]OSKKey{
		charRow("123", 1.0),
		charRow("456", 1.0),
		charRow("789", 1.0),
		{
			{Label: "", Action: OSKActionChar, Width: 1.0},
			{Label: "0", Action: OSKActionChar, Width: 1.0, Char: '0'},
			{Label: "⌫", Action: OSKActionBackspace, Width: 1.0},
		},
	}
}

func ipRows() [][]OSKKey {
	return [][]OSKKey{
		charRow("789", 1.0),
		charRow("456", 1.0),
		charRow("123", 1.0),
		{
			{Label: ".", Action: OSKActionChar, Width: 1.0, Char: '.'},
			{Label: "0", Action: OSKActionChar, Width: 1.0, Char: '0'},
			{Label: ":", Action: OSKActionChar, Width: 1.0, Char: ':'},
		},
	}
}

func hexRows() [][]OSKKey {
	return [][]OSKKey{
		{
			{Label: "7", Action: OSKActionChar, Width: 1.0, Char: '7'},
			{Label: "8", Action: OSKActionChar, Width: 1.0, Char: '8'},
			{Label: "9", Action: OSKActionChar, Width: 1.0, Char: '9'},
			{Label: "A", Action: OSKActionChar, Width: 1.0, Char: 'A'},
			{Label: "B", Action: OSKActionChar, Width: 1.0, Char: 'B'},
		},
		{
			{Label: "4", Action: OSKActionChar, Width: 1.0, Char: '4'},
			{Label: "5", Action: OSKActionChar, Width: 1.0, Char: '5'},
			{Label: "6", Action: OSKActionChar, Width: 1.0, Char: '6'},
			{Label: "C", Action: OSKActionChar, Width: 1.0, Char: 'C'},
			{Label: "D", Action: OSKActionChar, Width: 1.0, Char: 'D'},
		},
		{
			{Label: "1", Action: OSKActionChar, Width: 1.0, Char: '1'},
			{Label: "2", Action: OSKActionChar, Width: 1.0, Char: '2'},
			{Label: "3", Action: OSKActionChar, Width: 1.0, Char: '3'},
			{Label: "E", Action: OSKActionChar, Width: 1.0, Char: 'E'},
			{Label: "F", Action: OSKActionChar, Width: 1.0, Char: 'F'},
		},
		{
			{Label: "0", Action: OSKActionChar, Width: 1.0, Char: '0'},
			{Label: "⌫", Action: OSKActionBackspace, Width: 1.0},
			{Label: "↵", Action: OSKActionEnter, Width: 1.0},
		},
	}
}

func phoneRows() [][]OSKKey {
	return [][]OSKKey{
		charRow("123", 1.0),
		charRow("456", 1.0),
		charRow("789", 1.0),
		{
			{Label: "*", Action: OSKActionChar, Width: 1.0, Char: '*'},
			{Label: "0", Action: OSKActionChar, Width: 1.0, Char: '0'},
			{Label: "#", Action: OSKActionChar, Width: 1.0, Char: '#'},
		},
	}
}

// ── Condensed Layout (phone-style) ──────────────────────────────

func condensedRows(shifted bool) [][]OSKKey {
	if shifted {
		return [][]OSKKey{
			charRow("QWERTYUIOP", 1.0),
			charRow("ASDFGHJKL", 1.0),
			shiftRow("ZXCVBNM", true),
			condensedBottomRow(),
		}
	}
	return [][]OSKKey{
		charRow("qwertyuiop", 1.0),
		charRow("asdfghjkl", 1.0),
		shiftRow("zxcvbnm", false),
		condensedBottomRow(),
	}
}

// ── Full Layout (alpha + numpad side by side) ───────────────────

func fullRows(shifted bool) [][]OSKKey {
	alpha := alphaRows(shifted)
	num := numericRows()

	// Pad numpad rows to match alpha row count.
	for len(num) < len(alpha) {
		num = append(num, []OSKKey{})
	}

	result := make([][]OSKKey, len(alpha))
	for i := range alpha {
		row := make([]OSKKey, len(alpha[i]))
		copy(row, alpha[i])
		// Add separator.
		row = append(row, OSKKey{Label: "", Action: OSKActionChar, Width: 0.3})
		if i < len(num) {
			row = append(row, num[i]...)
		}
		result[i] = row
	}
	return result
}

// ── Helpers ─────────────────────────────────────────────────────

func charRow(chars string, width float32) []OSKKey {
	runes := []rune(chars)
	keys := make([]OSKKey, len(runes))
	for i, r := range runes {
		keys[i] = OSKKey{
			Label:  strings.ToUpper(string(r)),
			Action: OSKActionChar,
			Width:  width,
			Char:   r,
		}
		// Keep label matching the actual character for lowercase.
		keys[i].Label = string(r)
	}
	return keys
}

func symbolRow(chars string) []OSKKey {
	runes := []rune(chars)
	keys := make([]OSKKey, len(runes))
	for i, r := range runes {
		keys[i] = OSKKey{
			Label:  string(r),
			Action: OSKActionChar,
			Width:  1.0,
			Char:   r,
		}
	}
	return keys
}

func shiftRow(chars string, shifted bool) []OSKKey {
	shiftLabel := "⇧"
	if shifted {
		shiftLabel = "⇩"
	}
	row := []OSKKey{
		{Label: shiftLabel, Action: OSKActionShift, Width: 1.5},
	}
	for _, r := range chars {
		row = append(row, OSKKey{
			Label:  string(r),
			Action: OSKActionChar,
			Width:  1.0,
			Char:   r,
		})
	}
	row = append(row, OSKKey{Label: "⌫", Action: OSKActionBackspace, Width: 1.5})
	return row
}

func bottomRow() []OSKKey {
	return []OSKKey{
		{Label: "?123", Action: OSKActionSwitch, Width: 1.5},
		{Label: ",", Action: OSKActionChar, Width: 1.0, Char: ','},
		{Label: " ", Action: OSKActionSpace, Width: 4.0},
		{Label: ".", Action: OSKActionChar, Width: 1.0, Char: '.'},
		{Label: "↵", Action: OSKActionEnter, Width: 1.5},
		{Label: "⌨", Action: OSKActionDismiss, Width: 1.0},
	}
}

func condensedBottomRow() []OSKKey {
	return []OSKKey{
		{Label: "123", Action: OSKActionSwitch, Width: 1.5},
		{Label: ",", Action: OSKActionChar, Width: 1.0, Char: ','},
		{Label: " ", Action: OSKActionSpace, Width: 5.0},
		{Label: ".", Action: OSKActionChar, Width: 1.0, Char: '.'},
		{Label: "↵", Action: OSKActionEnter, Width: 1.5},
	}
}
