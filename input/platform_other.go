//go:build !darwin

package input

// platformModifier returns the primary command modifier for the current OS.
// Windows and Linux use Ctrl.
func platformModifier() ModifierSet { return ModCtrl }
