package nodes

import (
	"math"
	"testing"

	"texxer/internal/graph"
	"texxer/internal/img"
)

// The distance field of a rasterized box should match the box's exact sdf
// within a pixel, also across the tiled edge.
func TestDistanceMatchesSDF(t *testing.T) {
	ctx := graph.Ctx{W: 64, H: 64, Tiling: img.TileBoth, Scale: 1}
	g := &graph.Graph{Nodes: map[string]graph.Node{
		"box":  {Type: "shape", Params: graph.Params{"width": 20.0, "height": 12.0, "x": 28.0}},
		"mask": {Type: "scan", Params: graph.Params{"position": 0.5, "contrast": 1.0}, Inputs: map[string]string{"in": "box.sdf"}},
		"dist": {Type: "distance", Params: graph.Params{"range": 40.0}, Inputs: map[string]string{"in": "mask"}},
	}}
	g.Nodes["mask"].Params["invert"] = true

	ev := graph.NewEvaluator(g, Evals(), nil)
	want, err := ev.Eval("box.sdf", ctx)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ev.Eval("dist", ctx)
	if err != nil {
		t.Fatal(err)
	}

	for i := range got.Pix {
		if d := math.Abs(float64(got.Pix[i] - want.Pix[i])); d > 1 {
			t.Fatalf("pixel %d,%d: got %v, want %v", i%64, i/64, got.Pix[i], want.Pix[i])
		}
	}
}

func TestHSLRoundTrip(t *testing.T) {
	for _, c := range [][3]float64{{0.8, 0.2, 0.1}, {0.1, 0.6, 0.3}, {0.2, 0.3, 0.9}, {0.5, 0.5, 0.5}} {
		h, s, l := toHSL(c[0], c[1], c[2])
		r, g, b := fromHSL(h, s, l)
		if math.Abs(r-c[0])+math.Abs(g-c[1])+math.Abs(b-c[2]) > 1e-9 {
			t.Errorf("%v came back as %v %v %v", c, r, g, b)
		}
	}
}
