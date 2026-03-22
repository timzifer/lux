// Package a11y defines the core accessibility types for the Lux UI framework.
//
// These types are used both by the internal AccessTree (constructed from the VTree)
// and by external surfaces that provide semantic content via [SemanticProvider].
//
// This package depends only on the standard library and must NOT import
// ui, app, or draw packages.
package a11y

import "golang.org/x/text/language"

// AccessNodeID is a unique identifier for a node in the global AccessTree.
type AccessNodeID uint64

// AccessRole identifies the semantic role of an accessibility node.
type AccessRole uint32

const (
	RoleButton AccessRole = iota
	RoleCheckbox
	RoleCombobox
	RoleDialog
	RoleGrid
	RoleGroup
	RoleHeading
	RoleImage
	RoleLink
	RoleListbox
	RoleMenu
	RoleProgressBar
	RoleScrollBar
	RoleSlider
	RoleSpinButton
	RoleTab
	RoleTable
	RoleTextInput
	RoleToggle
	RoleTree

	// RoleCustomBase is the starting point for application-defined roles.
	// Custom roles should use RoleCustomBase + n.
	RoleCustomBase AccessRole = 1 << 16
)

// AccessLiveRegion controls how dynamic content updates are announced.
type AccessLiveRegion uint8

const (
	LiveOff       AccessLiveRegion = iota // No live update.
	LivePolite                            // Wait for idle before announcing.
	LiveAssertive                         // Interrupt immediately.
)

// AccessStates holds the boolean states of an accessibility node.
type AccessStates struct {
	Focused  bool
	Checked  bool
	Selected bool
	Expanded bool
	Disabled bool
	ReadOnly bool
	Required bool
	Invalid  bool
	Busy     bool
	Live     AccessLiveRegion
}

// AccessAction describes an action that can be performed on an accessibility node.
// Used by internal widgets where the trigger function runs in the app loop.
type AccessAction struct {
	Name    string
	Trigger func() // Executed in the app loop via Send.
}

// AccessActionDesc describes an action by name only, without a trigger function.
// Used by external surfaces where actions are routed via [SemanticProvider.PerformSemanticAction].
type AccessActionDesc struct {
	Name string
}

// AccessRelationKind identifies the type of relationship between accessibility nodes.
type AccessRelationKind uint8

const (
	RelationLabelledBy  AccessRelationKind = iota // This node is labelled by the target.
	RelationDescribedBy                           // This node is described by the target.
	RelationControls                              // This node controls the target.
	RelationFlowsTo                               // Reading order flows to the target.
)

// AccessRelation describes a relationship to another node in the global AccessTree.
type AccessRelation struct {
	Kind     AccessRelationKind
	TargetID AccessNodeID
}

// AccessRelationDesc describes a relationship using a generic uint64 target ID.
// Used by external surfaces where node IDs are surface-local.
type AccessRelationDesc struct {
	Kind     AccessRelationKind
	TargetID uint64
}

// AccessNumericValue describes a numeric value with bounds and step size.
// Used by platform bridges (UIA IRangeValueProvider, AT-SPI2 Value interface,
// NSAccessibility accessibilityValue) to expose sliders, progress bars, etc.
type AccessNumericValue struct {
	Current float64
	Min     float64
	Max     float64
	Step    float64 // 0 = continuous
}

// AccessTextState describes caret and selection state for editable text nodes.
// Used by platform bridges (UIA ITextProvider, AT-SPI2 Text interface,
// NSAccessibility text attributes) to expose cursor and selection.
type AccessTextState struct {
	Length         int // Total number of characters (rune count).
	CaretOffset    int // Caret position as rune offset; -1 if not applicable.
	SelectionStart int // Start of selection range (rune offset); -1 if no selection.
	SelectionEnd   int // End of selection range (rune offset); -1 if no selection.
}

// AccessNode represents a single node in the accessibility tree.
type AccessNode struct {
	Role         AccessRole
	Label        string       // Primary name (aria-label equivalent).
	Description  string       // Longer description (aria-describedby equivalent).
	Value        string       // Current value (for sliders, inputs, etc.).
	Lang         language.Tag // BCP 47 language tag; empty inherits from parent.
	States       AccessStates
	Actions      []AccessAction
	Relations    []AccessRelation
	NumericValue *AccessNumericValue // Non-nil for nodes with numeric range (slider, progress bar, spin button).
	TextState    *AccessTextState    // Non-nil for editable or selectable text nodes.
}
