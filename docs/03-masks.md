# Masks

![](img/masks.png)

A mask decides where a layer shows. With several masks, the layer shows where all of them do. **M** shows the selected layer's mask.

**+ mask** adds one; it replaces an empty paint or stamps mask. Most masks have **amount** (how much shows), **soft** (edge falloff), **rough** (a noisy edge) and **invert**. Soft and rough show once set, or under **more**.

| mask | shows where |
|---|---|
| surfaces | the picked parts of the model, and islands: surfaces bounded by hard edges. **⌘click** adds a part, **⇧⌘click** an island; once the layer has the mask, plain clicks do (**⇧** for islands). A model of one part has only islands |
| paint | you paint with the brush |
| stamps | stamps sit |
| volume | inside a box, cylinder or sphere you move in the view, all around the model. **soft** fades it towards its walls |
| grunge | a grunge map, projected from three sides |
| ao | the surface is occluded |
| edges | convex edges, for wear. **1–3 px** gives crisp lines that many texels wide |
| cavity | creases, gaps and corners where surfaces meet, for grime. **1–3 px** as for edges |
| drips | below edges, running down walls. **rough** varies their length |
| gradient | rises from one side: below, above, left, right, back, front (as in a front view) |
| facing | the surface faces a direction |

Use surfaces for zones, ao, edges and cavity for wear. Volume with grunge puts dirt exactly where you want it.

## Grunge maps

texxer brings a few maps made from noise: clouds, blotches, grime, specks, streaks, scratches, lines. Your own go into the grunge folder, the last tile of the map grid (on a Mac `~/Library/Application Support/texxer/grunge`): grays, white = dirt, best tileable, PNG or JPG, the file name is the map's name. They are evened out on loading, so **amount** is about the share they cover. A map a project uses but the folder lacks shows red and masks nothing.

**Q** edits the layer's volume, adding one if needed: click the model to put it there, drag it over the surface, or use the gizmo. **W E R** (or the buttons at the top of the panel) move, turn and size, **[ ]** size it all at once, **X** mirrors it across x. Prefer hard cuts: soft masks read as smears once crunched.
