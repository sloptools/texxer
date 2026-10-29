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
	"tiles":   tilesDef,
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
	"ao":        aoDef,

	"mask": maskDef,

	"downsample": downsampleDef,
	"quantize":   quantizeDef,
}
