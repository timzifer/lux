package a11y

// A11yBridge is the platform-specific accessibility bridge interface.
// Implementations live in platform packages (e.g. platform/windows).
// This interface lives in a11y/ to avoid circular imports.
type A11yBridge interface {
	// UpdateTree replaces the current access tree snapshot.
	// Called after each reconcile+layout pass.
	UpdateTree(tree AccessTree)

	// NotifyFocus informs the bridge that keyboard focus moved to the given node.
	NotifyFocus(nodeID AccessNodeID)

	// NotifyLiveRegion announces a live-region content change.
	NotifyLiveRegion(nodeID AccessNodeID, text string)
}
