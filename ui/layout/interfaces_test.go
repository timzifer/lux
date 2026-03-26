package layout_test

import (
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/layout"
)

// Compile-time interface checks: every type must implement all 4 interfaces.
var (
	_ ui.Layouter      = layout.Box{}
	_ ui.TreeEqualizer = layout.Box{}
	_ ui.ChildResolver = layout.Box{}
	_ ui.AccessWalker  = layout.Box{}

	_ ui.Layouter      = layout.Stack{}
	_ ui.TreeEqualizer = layout.Stack{}
	_ ui.ChildResolver = layout.Stack{}
	_ ui.AccessWalker  = layout.Stack{}

	_ ui.Layouter      = layout.Padding{}
	_ ui.TreeEqualizer = layout.Padding{}
	_ ui.ChildResolver = layout.Padding{}
	_ ui.AccessWalker  = layout.Padding{}

	_ ui.Layouter      = layout.SizedBox{}
	_ ui.TreeEqualizer = layout.SizedBox{}
	_ ui.ChildResolver = layout.SizedBox{}
	_ ui.AccessWalker  = layout.SizedBox{}

	_ ui.Layouter      = layout.Expanded{}
	_ ui.TreeEqualizer = layout.Expanded{}
	_ ui.ChildResolver = layout.Expanded{}
	_ ui.AccessWalker  = layout.Expanded{}

	_ ui.Layouter      = layout.Flex{}
	_ ui.TreeEqualizer = layout.Flex{}
	_ ui.ChildResolver = layout.Flex{}
	_ ui.AccessWalker  = layout.Flex{}

	_ ui.Layouter      = layout.Grid{}
	_ ui.TreeEqualizer = layout.Grid{}
	_ ui.ChildResolver = layout.Grid{}
	_ ui.AccessWalker  = layout.Grid{}

	_ ui.Layouter      = layout.CustomLayout{}
	_ ui.TreeEqualizer = layout.CustomLayout{}
	_ ui.ChildResolver = layout.CustomLayout{}
	_ ui.AccessWalker  = layout.CustomLayout{}

	_ ui.Layouter      = layout.GridItem{}
	_ ui.TreeEqualizer = layout.GridItem{}
	_ ui.ChildResolver = layout.GridItem{}
	_ ui.AccessWalker  = layout.GridItem{}
)
