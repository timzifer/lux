package input

// platformModifier returns the primary command modifier for the current OS.
// macOS uses Super (Cmd).
func platformModifier() ModifierSet { return ModSuper }
