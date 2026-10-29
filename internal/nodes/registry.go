package nodes

var Registry = map[string]Def{
	"noise": noiseDef,
	"cells": cellsDef,
	"color": colorDef,

	"dirt":      dirtDef,
	"spots":     spotsDef,
	"streaks":   streaksDef,
	"scratches": scratchesDef,
	"grain":     grainDef,

	"pattern": patternDef,
	"panels":  panelsDef,
	"trim":    trimDef,
	"sdf":     sdfDef,
	"profile": profileDef,

	"floodfill": floodFillDef,
	"warp":      warpDef,
	"blur":      blurDef,
	"sharpen":   sharpenDef,
	"levels":    levelsDef,
	"gradient":  gradientDef,
	"blend":     blendDef,

	"light":     lightDef,
	"slope":     slopeDef,
	"curvature": curvatureDef,

	"downsample": downsampleDef,
	"quantize":   quantizeDef,
}
