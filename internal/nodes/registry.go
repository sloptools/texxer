package nodes

var Registry = map[string]Def{
	"noise": noiseDef,
	"cells": cellsDef,
	"color": colorDef,
	"image": imageDef,

	"dirt":      dirtDef,
	"spots":     spotsDef,
	"streaks":   streaksDef,
	"scratches": scratchesDef,
	"grain":     grainDef,
	"strokes":   strokesDef,

	"pattern": patternDef,
	"panels":  panelsDef,
	"tiles":   tilesDef,
	"shape":   shapeDef,
	"grid":    gridDef,
	"text":    textDef,
	"trim":    trimDef,
	"sdf":     sdfDef,
	"profile": profileDef,

	"floodfill": floodFillDef,
	"warp":      warpDef,
	"dirwarp":   dirwarpDef,
	"gradwarp":  gradwarpDef,
	"slopeblur": slopeBlurDef,
	"scan":      scanDef,
	"blur":      blurDef,
	"repeat":    repeatDef,
	"transform": transformDef,
	"sharpen":   sharpenDef,
	"levels":    levelsDef,
	"gradient":  gradientDef,
	"blend":     blendDef,

	"light":     lightDef,
	"slope":     slopeDef,
	"curvature": curvatureDef,
	"ao":        aoDef,
	"shadow":    shadowDef,

	"mask": maskDef,

	"downsample": downsampleDef,
	"quantize":   quantizeDef,
}
