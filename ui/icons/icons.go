// Package icons provides named constants for Phosphor icon codepoints.
//
// Each constant is a single-rune string that can be passed to ui.Icon()
// or ui.IconSize(). The glyphs live in the embedded Phosphor font and
// are rendered via the "Phosphor" font family.
package icons

const (
	// Navigation
	CaretRight = "\uE13A"
	CaretDown  = "\uE136"
	CaretUp    = "\uE13C"
	CaretLeft  = "\uE138"
	ArrowRight = "\uE06C"
	ArrowLeft  = "\uE058"
	House      = "\uE2C2"

	// Actions
	Check         = "\uE182"
	X             = "\uE4F6"
	Plus          = "\uE3D4"
	Minus         = "\uE32A"
	Pencil        = "\uE3AE"
	Trash         = "\uE4A6"
	MagnifyingGlass = "\uE30C"
	Gear          = "\uE270"

	// Files & Folders
	Folder     = "\uE24A"
	FolderOpen = "\uE256"
	File       = "\uE230"
	FileText   = "\uE23A"

	// Media
	Play  = "\uE3CA"
	Pause = "\uE3A0"

	// Communication / Sharing
	Share        = "\uE43C"
	Copy         = "\uE1A6"
	Download     = "\uE1E2"
	Upload       = "\uE4BA"
	Link         = "\uE2DE"
	EnvelopeSimple = "\uE1F6"

	// Misc
	DotsThreeVertical = "\uE1D4"
	FunnelSimple      = "\uE268"
	SortAscending     = "\uE458"
	Warning           = "\uE4DA"
	Info              = "\uE2CC"

	// Date & Time
	Calendar = "\uE118"
	Clock    = "\uE196"
	Palette  = "\uE398"

	// UI / Symbols
	Star        = "\uE46A"
	Heart       = "\uE2A8"
	Eye         = "\uE220"
	EyeSlash    = "\uE222"
	User        = "\uE4C2"
	Sun         = "\uE472"
	Moon        = "\uE330"
	ToggleLeft  = "\uE674"
	ToggleRight = "\uE676"

	// Text Formatting
	TextBolder    = "\uE5BE"
	TextItalic    = "\uE5C0"
	TextUnderline = "\uE5C4"
	ImageSquare   = "\uE2CA"
)
