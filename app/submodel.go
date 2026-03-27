package app

// SubModel describes how to delegate messages to a child model (RFC §3.5).
// Get extracts the child from the parent, Set writes it back,
// and Update processes the message on the child.
type SubModel[Parent, Child any] struct {
	Get    func(Parent) Child
	Set    func(Parent, Child) Parent
	Update UpdateFunc[Child]
}

// Delegate runs a SubModel's Update on the child extracted from parent,
// then writes the result back. Call this from your parent update function.
func Delegate[Parent, Child any](sm SubModel[Parent, Child], parent Parent, msg Msg) Parent {
	child := sm.Get(parent)
	child = sm.Update(child, msg)
	return sm.Set(parent, child)
}

// SubModelWithCmd is like SubModel but for update functions that return commands.
type SubModelWithCmd[Parent, Child any] struct {
	Get    func(Parent) Child
	Set    func(Parent, Child) Parent
	Update UpdateWithCmd[Child]
}

// DelegateWithCmd runs a SubModelWithCmd's Update on the child extracted from parent,
// writes the result back, and returns the command from the child update.
func DelegateWithCmd[Parent, Child any](sm SubModelWithCmd[Parent, Child], parent Parent, msg Msg) (Parent, Cmd) {
	child := sm.Get(parent)
	child, cmd := sm.Update(child, msg)
	return sm.Set(parent, child), cmd
}
