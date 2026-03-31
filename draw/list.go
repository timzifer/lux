package draw

// ListType identifies the kind of list a paragraph belongs to (CSS list-style-type category).
type ListType uint8

const (
	ListTypeNone      ListType = iota // no list
	ListTypeUnordered                 // ul — bullets
	ListTypeOrdered                   // ol — numbers
)

// ListMarker controls the bullet or number style (CSS list-style-type).
type ListMarker uint8

const (
	ListMarkerDefault    ListMarker = iota // auto based on type and nesting level
	ListMarkerDisc                         // • (filled circle, U+2022)
	ListMarkerCircle                       // ⁃ (hyphen bullet, U+2043)
	ListMarkerSquare                       // ‣ (triangular bullet, U+2023)
	ListMarkerDecimal                      // 1. 2. 3.
	ListMarkerLowerAlpha                   // a. b. c.
	ListMarkerUpperAlpha                   // A. B. C.
	ListMarkerLowerRoman                   // i. ii. iii.
	ListMarkerUpperRoman                   // I. II. III.
	ListMarkerNone                         // no marker (content only)
)
