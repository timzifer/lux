package a11y

// Rect represents a bounding rectangle in screen coordinates.
type Rect struct{ X, Y, Width, Height float64 }

// AccessTreeNode is a node in the flattened, depth-first ordered access tree.
// Navigation indices enable O(1) parent/child/sibling traversal.
type AccessTreeNode struct {
	ID          AccessNodeID
	ParentIndex int32 // -1 for root
	FirstChild  int32 // -1 if no children
	NextSibling int32 // -1 if last child
	PrevSibling int32 // -1 if first child
	ChildCount  int
	Node        AccessNode
	Bounds      Rect // Screen coordinates
}

// AccessTree holds the complete access tree for a window.
// Nodes are stored in depth-first order.
type AccessTree struct {
	Nodes     []AccessTreeNode
	FocusedID AccessNodeID
	idIndex   map[AccessNodeID]int
}

// buildIndex (re)builds the ID→index lookup map.
func (t *AccessTree) buildIndex() {
	t.idIndex = make(map[AccessNodeID]int, len(t.Nodes))
	for i := range t.Nodes {
		t.idIndex[t.Nodes[i].ID] = i
	}
}

// EnsureIndex builds the index if it hasn't been built yet.
func (t *AccessTree) EnsureIndex() {
	if t.idIndex == nil {
		t.buildIndex()
	}
}

// FindByID returns the node with the given ID, or nil if not found.
func (t *AccessTree) FindByID(id AccessNodeID) *AccessTreeNode {
	t.EnsureIndex()
	if idx, ok := t.idIndex[id]; ok {
		return &t.Nodes[idx]
	}
	return nil
}

// IndexByID returns the index of the node with the given ID, or -1.
func (t *AccessTree) IndexByID(id AccessNodeID) int {
	t.EnsureIndex()
	if idx, ok := t.idIndex[id]; ok {
		return idx
	}
	return -1
}

// NodeByIndex returns the node at index i, or nil if out of range.
func (t *AccessTree) NodeByIndex(i int) *AccessTreeNode {
	if i < 0 || i >= len(t.Nodes) {
		return nil
	}
	return &t.Nodes[i]
}

// Parent returns the parent node, or nil for root nodes.
func (t *AccessTree) Parent(n *AccessTreeNode) *AccessTreeNode {
	return t.NodeByIndex(int(n.ParentIndex))
}

// Children returns all direct children of the given node.
func (t *AccessTree) Children(n *AccessTreeNode) []*AccessTreeNode {
	if n.FirstChild < 0 {
		return nil
	}
	children := make([]*AccessTreeNode, 0, n.ChildCount)
	for idx := n.FirstChild; idx >= 0; {
		child := &t.Nodes[idx]
		children = append(children, child)
		idx = child.NextSibling
	}
	return children
}

// FindByRole returns all nodes with the given role.
func (t *AccessTree) FindByRole(role AccessRole) []*AccessTreeNode {
	var result []*AccessTreeNode
	for i := range t.Nodes {
		if t.Nodes[i].Node.Role == role {
			result = append(result, &t.Nodes[i])
		}
	}
	return result
}

// FindByLabel returns all nodes whose label matches exactly.
func (t *AccessTree) FindByLabel(label string) []*AccessTreeNode {
	var result []*AccessTreeNode
	for i := range t.Nodes {
		if t.Nodes[i].Node.Label == label {
			result = append(result, &t.Nodes[i])
		}
	}
	return result
}

// Root returns the first (root) node, or nil if empty.
func (t *AccessTree) Root() *AccessTreeNode {
	if len(t.Nodes) == 0 {
		return nil
	}
	return &t.Nodes[0]
}

// FocusedNode returns the currently focused node, or nil.
func (t *AccessTree) FocusedNode() *AccessTreeNode {
	if t.FocusedID == 0 {
		return nil
	}
	return t.FindByID(t.FocusedID)
}
