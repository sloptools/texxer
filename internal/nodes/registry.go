package nodes

import "texxer/internal/graph"

var Registry = map[string]graph.EvalFunc{
	"noise":      noiseNode,
	"grunge":     grungeNode,
	"cells":      cellsNode,
	"warp":       warpNode,
	"floodfill":  floodFillNode,
	"pattern":    patternNode,
	"trim":       trimNode,
	"panels":     panelsNode,
	"sdf":        sdfNode,
	"profile":    profileNode,
	"light":      lightNode,
	"slope":      slopeNode,
	"curvature":  curvatureNode,
	"color":      colorNode,
	"downsample": downsampleNode,
	"sharpen":    sharpenNode,
	"blur":       blurNode,
	"levels":     levelsNode,
	"gradient":   gradientNode,
	"blend":      blendNode,
	"quantize":   quantizeNode,
}
