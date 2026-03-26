package uitest

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/render"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
)

// BuildScene constructs a Scene from an Element tree using the default theme
// and a SceneCanvas. This is the standard entry point for golden-file tests.
func BuildScene(root ui.Element, width, height int) draw.Scene {
	canvas := render.NewSceneCanvas(width, height)
	return ui.BuildScene(root, canvas, theme.Default, width, height, nil)
}
