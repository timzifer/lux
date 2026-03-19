package ui

import (
	"testing"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
)

// fakeSemantic is a mock SurfaceProvider + SemanticProvider for testing.
type fakeSemantic struct {
	nodes   []SurfaceAccessNode
	version uint64
	focused SurfaceNodeID
}

func (f *fakeSemantic) AcquireFrame(bounds draw.Rect) (draw.TextureID, FrameToken) {
	return 0, 0
}
func (f *fakeSemantic) ReleaseFrame(token FrameToken) {}
func (f *fakeSemantic) HandleMsg(msg any) bool         { return false }

func (f *fakeSemantic) SnapshotSemantics(bounds draw.Rect) SurfaceSemantics {
	return SurfaceSemantics{Roots: f.nodes, Version: f.version}
}

func (f *fakeSemantic) HitTestSemantics(p draw.Point) (SurfaceNodeID, bool) {
	for _, n := range f.nodes {
		if p.X >= n.Bounds.X && p.X < n.Bounds.X+n.Bounds.W &&
			p.Y >= n.Bounds.Y && p.Y < n.Bounds.Y+n.Bounds.H {
			return n.ID, true
		}
	}
	return 0, false
}

func (f *fakeSemantic) FocusSemanticNode(id SurfaceNodeID) bool {
	for _, n := range f.nodes {
		if n.ID == id {
			f.focused = id
			return true
		}
	}
	return false
}

func (f *fakeSemantic) PerformSemanticAction(id SurfaceNodeID, action string) bool {
	for _, n := range f.nodes {
		if n.ID == id {
			for _, a := range n.Actions {
				if a.Name == action {
					return true
				}
			}
		}
	}
	return false
}

// Compile-time checks that fakeSemantic implements both interfaces.
var _ SurfaceProvider = (*fakeSemantic)(nil)
var _ SemanticProvider = (*fakeSemantic)(nil)

func TestSemanticProviderOptional(t *testing.T) {
	// A SurfaceProvider that does NOT implement SemanticProvider.
	var sp SurfaceProvider = &fakeSurface{}
	if _, ok := sp.(SemanticProvider); ok {
		t.Error("fakeSurface should not implement SemanticProvider")
	}
}

// fakeSurface is a SurfaceProvider without semantics.
type fakeSurface struct{}

func (f *fakeSurface) AcquireFrame(bounds draw.Rect) (draw.TextureID, FrameToken) {
	return 0, 0
}
func (f *fakeSurface) ReleaseFrame(token FrameToken) {}
func (f *fakeSurface) HandleMsg(msg any) bool         { return false }

func TestSnapshotSemantics(t *testing.T) {
	pdf := &fakeSemantic{
		nodes: []SurfaceAccessNode{
			{ID: 1, Role: a11y.RoleHeading, Label: "Chapter 1", Bounds: draw.Rect{X: 0, Y: 0, W: 200, H: 30}},
			{ID: 2, Role: a11y.RoleLink, Label: "More info", Bounds: draw.Rect{X: 0, Y: 30, W: 100, H: 20}},
			{ID: 3, Role: a11y.RoleTextInput, Label: "Name", Bounds: draw.Rect{X: 0, Y: 50, W: 150, H: 25}},
		},
		version: 1,
	}

	sem := pdf.SnapshotSemantics(draw.Rect{W: 300, H: 400})
	if len(sem.Roots) != 3 {
		t.Fatalf("expected 3 roots, got %d", len(sem.Roots))
	}
	if sem.Version != 1 {
		t.Errorf("expected version 1, got %d", sem.Version)
	}
	if sem.Roots[0].Role != a11y.RoleHeading {
		t.Errorf("expected RoleHeading, got %d", sem.Roots[0].Role)
	}
	if sem.Roots[1].Label != "More info" {
		t.Errorf("expected 'More info', got %q", sem.Roots[1].Label)
	}
}

func TestHitTestSemantics(t *testing.T) {
	pdf := &fakeSemantic{
		nodes: []SurfaceAccessNode{
			{ID: 1, Role: a11y.RoleHeading, Label: "Title", Bounds: draw.Rect{X: 0, Y: 0, W: 200, H: 30}},
			{ID: 2, Role: a11y.RoleLink, Label: "Link", Bounds: draw.Rect{X: 0, Y: 30, W: 100, H: 20}},
		},
	}

	// Hit the heading.
	id, ok := pdf.HitTestSemantics(draw.Pt(10, 10))
	if !ok || id != 1 {
		t.Errorf("expected hit on node 1, got %d (ok=%v)", id, ok)
	}

	// Hit the link.
	id, ok = pdf.HitTestSemantics(draw.Pt(50, 35))
	if !ok || id != 2 {
		t.Errorf("expected hit on node 2, got %d (ok=%v)", id, ok)
	}

	// Miss.
	_, ok = pdf.HitTestSemantics(draw.Pt(250, 250))
	if ok {
		t.Error("expected miss")
	}
}

func TestFocusSemanticNode(t *testing.T) {
	pdf := &fakeSemantic{
		nodes: []SurfaceAccessNode{
			{ID: 1, Role: a11y.RoleTextInput, Label: "Name"},
		},
	}

	if !pdf.FocusSemanticNode(1) {
		t.Error("expected focus to succeed for node 1")
	}
	if pdf.focused != 1 {
		t.Errorf("expected focused=1, got %d", pdf.focused)
	}
	if pdf.FocusSemanticNode(99) {
		t.Error("expected focus to fail for non-existent node 99")
	}
}

func TestPerformSemanticAction(t *testing.T) {
	pdf := &fakeSemantic{
		nodes: []SurfaceAccessNode{
			{
				ID:      1,
				Role:    a11y.RoleButton,
				Label:   "Submit",
				Actions: []a11y.AccessActionDesc{{Name: "activate"}},
			},
		},
	}

	if !pdf.PerformSemanticAction(1, "activate") {
		t.Error("expected action 'activate' to succeed")
	}
	if pdf.PerformSemanticAction(1, "unknown") {
		t.Error("expected unknown action to fail")
	}
	if pdf.PerformSemanticAction(99, "activate") {
		t.Error("expected action on non-existent node to fail")
	}
}

func TestSurfaceAccessNodeStates(t *testing.T) {
	node := SurfaceAccessNode{
		ID:    42,
		Role:  a11y.RoleCustomBase + 1,
		Label: "Caption",
		Value: "Please leave the building.",
		States: a11y.AccessStates{
			Live: a11y.LivePolite,
		},
	}

	if node.States.Live != a11y.LivePolite {
		t.Errorf("expected LivePolite, got %d", node.States.Live)
	}
	if node.Role != a11y.RoleCustomBase+1 {
		t.Error("expected custom role")
	}
}

func TestSurfaceAccessNodeRelations(t *testing.T) {
	node := SurfaceAccessNode{
		ID:   1,
		Role: a11y.RoleTextInput,
		Relations: []a11y.AccessRelationDesc{
			{Kind: a11y.RelationLabelledBy, TargetID: 2},
		},
	}
	if len(node.Relations) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(node.Relations))
	}
	if node.Relations[0].Kind != a11y.RelationLabelledBy {
		t.Errorf("expected RelationLabelledBy, got %d", node.Relations[0].Kind)
	}
}
