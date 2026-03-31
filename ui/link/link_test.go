package link

import (
	"testing"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/display"
)

func TestText(t *testing.T) {
	clicked := false
	el := Text("Click me", func() { clicked = true })
	l, ok := el.(Link)
	if !ok {
		t.Fatalf("expected Link, got %T", el)
	}
	txt, ok := l.Content.(display.TextElement)
	if !ok {
		t.Fatalf("expected TextElement content, got %T", l.Content)
	}
	if txt.Content != "Click me" {
		t.Errorf("expected label %q, got %q", "Click me", txt.Content)
	}
	if l.OnClick == nil {
		t.Fatal("expected non-nil OnClick")
	}
	l.OnClick()
	if !clicked {
		t.Error("OnClick did not fire")
	}
	if l.Disabled {
		t.Error("expected Disabled=false")
	}
}

func TestWithURL(t *testing.T) {
	el := WithURL("Docs", "https://example.com", func() {})
	l := el.(Link)
	if l.URL != "https://example.com" {
		t.Errorf("expected URL %q, got %q", "https://example.com", l.URL)
	}
	txt := l.Content.(display.TextElement)
	if txt.Content != "Docs" {
		t.Errorf("expected label %q, got %q", "Docs", txt.Content)
	}
}

func TestTextDisabled(t *testing.T) {
	el := TextDisabled("Disabled")
	l := el.(Link)
	if !l.Disabled {
		t.Error("expected Disabled=true")
	}
	if l.OnClick != nil {
		t.Error("expected nil OnClick for disabled link")
	}
}

func TestNew(t *testing.T) {
	content := display.TextElement{Content: "child"}
	el := New(content, func() {})
	l := el.(Link)
	if l.Content != content {
		t.Error("expected content to match")
	}
}

func TestTreeEqual(t *testing.T) {
	a := Link{URL: "https://a.com", Disabled: false}
	b := Link{URL: "https://a.com", Disabled: false}
	c := Link{URL: "https://b.com", Disabled: false}
	d := Link{URL: "https://a.com", Disabled: true}

	if !a.TreeEqual(b) {
		t.Error("identical links should be TreeEqual")
	}
	if a.TreeEqual(c) {
		t.Error("different URLs should not be TreeEqual")
	}
	if a.TreeEqual(d) {
		t.Error("different Disabled should not be TreeEqual")
	}
	if a.TreeEqual(display.TextElement{Content: "x"}) {
		t.Error("Link should not be TreeEqual to TextElement")
	}
}

func TestResolveChildren_TextContent(t *testing.T) {
	l := Link{Content: display.TextElement{Content: "leaf"}}
	resolved := l.ResolveChildren(func(el ui.Element, idx int) ui.Element {
		t.Error("resolve should not be called for TextElement content")
		return el
	})
	if _, ok := resolved.(Link); !ok {
		t.Fatalf("expected Link, got %T", resolved)
	}
}

func TestResolveChildren_GenericContent(t *testing.T) {
	called := false
	original := display.TextElement{Content: "original"}
	replacement := display.TextElement{Content: "replaced"}
	l := Link{Content: wrapElement{child: original}}
	resolved := l.ResolveChildren(func(el ui.Element, idx int) ui.Element {
		called = true
		if idx != 0 {
			t.Errorf("expected index 0, got %d", idx)
		}
		return replacement
	})
	if !called {
		t.Error("resolve should be called for non-TextElement content")
	}
	rl := resolved.(Link)
	if rl.Content != replacement {
		t.Error("content should be replaced")
	}
}

func TestWalkAccess_Link(t *testing.T) {
	clicked := false
	l := Link{
		Content: display.TextElement{Content: "Go to docs"},
		OnClick: func() { clicked = true },
		URL:     "https://docs.example.com",
	}

	tree := ui.RenderToAccessTree(l)
	links := tree.FindByRole(a11y.RoleLink)
	if len(links) != 1 {
		t.Fatalf("expected 1 link node, got %d", len(links))
	}
	node := links[0].Node
	if node.Label != "Go to docs" {
		t.Errorf("expected label %q, got %q", "Go to docs", node.Label)
	}
	if node.Value != "https://docs.example.com" {
		t.Errorf("expected value %q, got %q", "https://docs.example.com", node.Value)
	}
	if len(node.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(node.Actions))
	}
	node.Actions[0].Trigger()
	if !clicked {
		t.Error("action trigger should set clicked=true")
	}
}

func TestWalkAccess_DisabledLink(t *testing.T) {
	l := Link{
		Content:  display.TextElement{Content: "Disabled link"},
		Disabled: true,
	}

	tree := ui.RenderToAccessTree(l)
	links := tree.FindByRole(a11y.RoleLink)
	if len(links) != 1 {
		t.Fatalf("expected 1 link node, got %d", len(links))
	}
	node := links[0].Node
	if !node.States.Disabled {
		t.Error("expected Disabled state")
	}
	if len(node.Actions) != 0 {
		t.Error("disabled link should have no actions")
	}
}

// wrapElement is a non-TextElement wrapper for testing ResolveChildren.
type wrapElement struct {
	ui.BaseElement
	child ui.Element
}

func (w wrapElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	return ui.Bounds{}
}

// Ensure Link implements key interfaces.
var (
	_ ui.Layouter      = Link{}
	_ ui.TreeEqualizer = Link{}
	_ ui.ChildResolver = Link{}
	_ ui.AccessWalker  = Link{}
)

// Verify unused import suppression.
var _ = theme.WidgetKindLink
