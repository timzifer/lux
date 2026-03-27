package data

import (
	"testing"

	"github.com/timzifer/lux/ui"
)

func TestTreeResolvedRootIDsLegacy(t *testing.T) {
	tr := Tree{RootIDs: []string{"a", "b", "c"}}
	ids := tr.resolvedRootIDs()
	if len(ids) != 3 || ids[0] != "a" || ids[1] != "b" || ids[2] != "c" {
		t.Fatalf("resolvedRootIDs() = %v, want [a b c]", ids)
	}
}

func TestTreeResolvedRootIDsDataset(t *testing.T) {
	ds := NewSliceDataset([]string{"x", "y"})
	tr := Tree{DatasetRoots: ds, RootIDs: []string{"ignored"}}
	ids := tr.resolvedRootIDs()
	if len(ids) != 2 || ids[0] != "x" || ids[1] != "y" {
		t.Fatalf("resolvedRootIDs() = %v, want [x y]", ids)
	}
}

func TestTreeResolvedRootIDsDatasetPriority(t *testing.T) {
	ds := NewSliceDataset([]string{"ds"})
	tr := Tree{DatasetRoots: ds, RootIDs: []string{"legacy"}}
	ids := tr.resolvedRootIDs()
	if len(ids) != 1 || ids[0] != "ds" {
		t.Fatalf("DatasetRoots should take priority, got %v", ids)
	}
}

func TestTreeResolvedRootIDsEmpty(t *testing.T) {
	tr := Tree{}
	ids := tr.resolvedRootIDs()
	if len(ids) != 0 {
		t.Fatalf("resolvedRootIDs() = %v, want empty", ids)
	}
}

func TestTreeResolvedRootIDsPagedDataset(t *testing.T) {
	ds := NewPagedDataset[string](10)
	ds.SetPage(0, []string{"p1", "p2", "p3"}, 3)
	tr := Tree{DatasetRoots: ds}
	ids := tr.resolvedRootIDs()
	if len(ids) != 3 || ids[0] != "p1" || ids[2] != "p3" {
		t.Fatalf("resolvedRootIDs() with PagedDataset = %v", ids)
	}
}

func TestTreeResolvedRootIDsStreamDataset(t *testing.T) {
	ds := NewStreamDataset[string](StreamAppend)
	ds.Append("s1", "s2")
	tr := Tree{DatasetRoots: ds}
	ids := tr.resolvedRootIDs()
	if len(ids) != 2 || ids[0] != "s1" || ids[1] != "s2" {
		t.Fatalf("resolvedRootIDs() with StreamDataset = %v", ids)
	}
}

func TestTreeResolvedChildrenLegacy(t *testing.T) {
	tr := Tree{
		Children: func(id string) []string {
			if id == "root" {
				return []string{"child1", "child2"}
			}
			return nil
		},
	}
	kids := tr.resolvedChildren("root")
	if len(kids) != 2 || kids[0] != "child1" {
		t.Fatalf("resolvedChildren(root) = %v", kids)
	}
	if len(tr.resolvedChildren("leaf")) != 0 {
		t.Fatal("resolvedChildren(leaf) should be empty")
	}
}

func TestTreeResolvedChildrenDataset(t *testing.T) {
	tr := Tree{
		DatasetChildren: func(id string) Dataset[string] {
			if id == "root" {
				return NewSliceDataset([]string{"dc1", "dc2"})
			}
			return nil
		},
		Children: func(id string) []string {
			return []string{"ignored"}
		},
	}
	kids := tr.resolvedChildren("root")
	if len(kids) != 2 || kids[0] != "dc1" {
		t.Fatalf("DatasetChildren should take priority, got %v", kids)
	}
	if len(tr.resolvedChildren("leaf")) != 0 {
		t.Fatal("resolvedChildren(leaf) should be empty")
	}
}

func TestTreeResolvedChildrenNilFunctions(t *testing.T) {
	tr := Tree{}
	if kids := tr.resolvedChildren("any"); len(kids) != 0 {
		t.Fatalf("resolvedChildren with nil functions should return nil, got %v", kids)
	}
}

func TestTreeResolvedChildrenPagedDataset(t *testing.T) {
	tr := Tree{
		DatasetChildren: func(id string) Dataset[string] {
			ds := NewPagedDataset[string](5)
			if id == "parent" {
				ds.SetPage(0, []string{"c1", "c2", "c3"}, 3)
			}
			return ds
		},
	}
	kids := tr.resolvedChildren("parent")
	if len(kids) != 3 {
		t.Fatalf("resolvedChildren(parent) = %v, want 3 items", kids)
	}
}

func TestTreeResolvedChildrenPartiallyLoaded(t *testing.T) {
	tr := Tree{
		DatasetChildren: func(id string) Dataset[string] {
			ds := NewPagedDataset[string](3)
			// Only load first page of 3, but total is 6
			ds.SetPage(0, []string{"a", "b", "c"}, 6)
			return ds
		},
	}
	kids := tr.resolvedChildren("node")
	// Should only return loaded items (first 3), not the unloaded page 1
	if len(kids) != 3 {
		t.Fatalf("resolvedChildren should only return loaded items, got %d", len(kids))
	}
}

func TestTreeBackwardCompat(t *testing.T) {
	tr := Tree{
		RootIDs:  []string{"r1", "r2"},
		Children: func(id string) []string { return nil },
	}
	ids := tr.resolvedRootIDs()
	if len(ids) != 2 || ids[0] != "r1" || ids[1] != "r2" {
		t.Fatalf("legacy RootIDs not resolved correctly: %v", ids)
	}
}

func TestTreeResolveChildren(t *testing.T) {
	tr := Tree{RootIDs: []string{"a"}}
	result := tr.ResolveChildren(func(el ui.Element, i int) ui.Element { return el })
	if _, ok := result.(Tree); !ok {
		t.Fatal("ResolveChildren should return the same Tree")
	}
}

func TestTreeTreeEqualAlwaysFalse(t *testing.T) {
	a := Tree{RootIDs: []string{"a"}}
	b := Tree{RootIDs: []string{"a"}}
	if a.TreeEqual(b) {
		t.Fatal("Tree.TreeEqual should always return false")
	}
}
