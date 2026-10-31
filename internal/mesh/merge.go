package mesh

import "math"

// Merge lays models side by side along x, so several files can share one texture
// without touching in the bakes. Part names get the file's name in front. It returns
// how far each model moved.
func Merge(models []*Model, names []string) (*Model, []Vec) {
	if len(models) == 1 {
		return models[0], []Vec{{}}
	}

	out := &Model{Min: Vec{math.Inf(1), math.Inf(1), math.Inf(1)}, Max: Vec{math.Inf(-1), math.Inf(-1), math.Inf(-1)}}
	offsets := make([]Vec, len(models))
	gap := 0.0
	for _, m := range models {
		gap = max(gap, m.Max.Sub(m.Min).Len()*0.15)
	}

	x := 0.0
	for i, m := range models {
		off := Vec{x - m.Min[0], 0, 0}
		offsets[i] = off
		x += m.Max[0] - m.Min[0] + gap

		parts := len(out.Parts)
		details := len(out.Details)
		out.Details = append(out.Details, m.Details...)
		for _, p := range m.Parts {
			out.Parts = append(out.Parts, names[i]+"/"+p)
		}
		for _, t := range m.Tris {
			for c := range 3 {
				t.P[c] = t.P[c].Add(off)
			}
			t.Part += parts
			if t.Detail >= 0 {
				t.Detail += details
			}
			out.Tris = append(out.Tris, t)
		}
		out.Min, out.Max = out.Min.Min(m.Min.Add(off)), out.Max.Max(m.Max.Add(off))
	}
	return out, offsets
}
