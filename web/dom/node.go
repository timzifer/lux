// Package dom provides a minimal DOM tree model inspired by the W3C DOM
// specification. It is designed to be reusable as the foundation for a
// browser engine (see RFC-998) while serving the immediate need of
// HTML ↔ AttributedString conversion.
package dom

// NodeType identifies the kind of a DOM node.
type NodeType uint8

const (
	DocumentNode NodeType = iota
	ElementNode
	TextNode
	CommentNode
)

// Node is the base type in the DOM tree.
// It uses a linked-list representation (FirstChild/NextSib) for efficient
// tree mutation without slice reallocation.
type Node struct {
	Type   NodeType
	Parent *Node

	FirstChild *Node
	LastChild  *Node
	PrevSib    *Node
	NextSib    *Node

	// Element-specific fields (valid when Type == ElementNode).
	Tag   string            // lowercase tag name, e.g. "p", "span"
	Attrs map[string]string // HTML attributes, e.g. "class" → "highlight"

	// Text/Comment-specific field (valid when Type == TextNode or CommentNode).
	Data string
}

// NewDocument creates an empty document node.
func NewDocument() *Node {
	return &Node{Type: DocumentNode}
}

// NewElement creates an element node with the given tag name.
func NewElement(tag string) *Node {
	return &Node{Type: ElementNode, Tag: tag, Attrs: make(map[string]string)}
}

// NewText creates a text node with the given content.
func NewText(data string) *Node {
	return &Node{Type: TextNode, Data: data}
}

// AppendChild adds child as the last child of n.
// If child already has a parent it is first removed.
func (n *Node) AppendChild(child *Node) {
	if child.Parent != nil {
		child.Parent.RemoveChild(child)
	}
	child.Parent = n
	child.PrevSib = n.LastChild
	child.NextSib = nil
	if n.LastChild != nil {
		n.LastChild.NextSib = child
	} else {
		n.FirstChild = child
	}
	n.LastChild = child
}

// RemoveChild removes child from n's children.
func (n *Node) RemoveChild(child *Node) {
	if child.Parent != n {
		return
	}
	if child.PrevSib != nil {
		child.PrevSib.NextSib = child.NextSib
	} else {
		n.FirstChild = child.NextSib
	}
	if child.NextSib != nil {
		child.NextSib.PrevSib = child.PrevSib
	} else {
		n.LastChild = child.PrevSib
	}
	child.Parent = nil
	child.PrevSib = nil
	child.NextSib = nil
}

// InsertBefore inserts child before ref among n's children.
// If ref is nil the child is appended.
func (n *Node) InsertBefore(child, ref *Node) {
	if ref == nil {
		n.AppendChild(child)
		return
	}
	if child.Parent != nil {
		child.Parent.RemoveChild(child)
	}
	child.Parent = n
	child.PrevSib = ref.PrevSib
	child.NextSib = ref
	if ref.PrevSib != nil {
		ref.PrevSib.NextSib = child
	} else {
		n.FirstChild = child
	}
	ref.PrevSib = child
}

// Children returns a slice of element children of n.
func (n *Node) Children() []*Node {
	var out []*Node
	for c := n.FirstChild; c != nil; c = c.NextSib {
		if c.Type == ElementNode {
			out = append(out, c)
		}
	}
	return out
}

// ChildNodes returns all direct child nodes (elements, text, comments).
func (n *Node) ChildNodes() []*Node {
	var out []*Node
	for c := n.FirstChild; c != nil; c = c.NextSib {
		out = append(out, c)
	}
	return out
}

// TextContent returns the recursive text content of the subtree.
func (n *Node) TextContent() string {
	if n.Type == TextNode {
		return n.Data
	}
	var b []byte
	for c := n.FirstChild; c != nil; c = c.NextSib {
		b = append(b, c.TextContent()...)
	}
	return string(b)
}

// Attr returns the value of an attribute (case-insensitive lookup).
func (n *Node) Attr(name string) string {
	if n.Attrs == nil {
		return ""
	}
	return n.Attrs[name]
}

// SetAttr sets an attribute value.
func (n *Node) SetAttr(name, value string) {
	if n.Attrs == nil {
		n.Attrs = make(map[string]string)
	}
	n.Attrs[name] = value
}
