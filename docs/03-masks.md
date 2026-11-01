# Masks

![](img/masks.png)

A mask decides where a layer shows. With several masks, the layer shows where all of them do. **M** shows the selected layer's mask.

Most masks have **amount** (how much shows), **soft** (edge falloff), **rough** (a noisy edge) and **invert**.

| mask | shows where |
|---|---|
| parts | the picked parts of the model. **⌘click** a part to add it |
| islands | the picked surfaces, bounded by hard edges. **⇧⌘click** to add one |
| paint | you paint with the brush |
| stamps | stamps sit |
| grunge | a grunge map, projected from three sides |
| ao | the surface is occluded |
| edges | convex edges, for wear |
| cavity | creases and gaps, for grime |
| gradient | rises from one side: below, above, left, right, back, front (as in a front view) |
| facing | the surface faces a direction |
| noise | value noise |

Use parts or islands for zones, ao, edges and cavity for wear. Prefer hard cuts: soft masks read as smears once crunched.
