# Layers

![](img/layers.png)

Layers stack bottom to top. A new layer starts with an empty paint mask and shows nothing until painted; another mask takes its place. A layer without masks covers everything.

- **color**: the layer's color. Swatches reuse colors of other layers.
- **tone**, **tone scale**: lighter and darker patches over the color, at a world size.
- **grain**: single texels a bit lighter or darker.
- **shine**: baked sheen and glints, for glass and gloss.
- **glows**: the color also goes into the emissive map.
- **colors from the model's texture**: the layer takes the model's own texture or material colors. One such layer alone just crunches a finished texture; layers above it add to it. texxer keeps a copy of the original beside the project, since the export overwrites it, and takes a new one when the file changes outside texxer.
- **blend**: normal, multiply, add, overlay, relief. Relief takes gray stamps as bumps: mid gray keeps the surface.
- **opacity**

**N** new (in the picked folder, if one is), **D** duplicate, **⌫** delete, **V** show or hide, **↩** rename, **↑ ↓** select, **⌥↑ ⌥↓** move, into and out of folders on the way. Right-click a layer or folder for the same and more.

Hovering a layer tints where it shows on the model. **⌥click** the model picks the topmost layer there. Drag over the eyes to show or hide several, **⌥click** an eye to show only that layer, again for all.

When dragging, the line shows where the layer lands: indented inside a folder, at the left edge below it. **Esc** cancels.

## Folders

**⌘G** puts the layer into a new folder, **⇧⌘G** dissolves it. Drag layers onto a folder to put them in. Folders hide and show all their layers.

Click a folder to give it a **zone**: with **pick** on, clicks in the view add surfaces (**⌘click** whole parts), **Esc** ends. Its layers show only there. Without a zone, the folder covers everything.

## Dirt

**1** to **6** add one dirt layer above the selected one: dust on top, grime in cavities, dirt from below, rust on edges, streaks on walls, and a grunge brush that paints dirt with the brush (B). They are also at the top of the material menu (**⇧M**).

## Materials

A material is a folder of layers: worn steel, painted metal, rust, wood, plastic, rubber, glass. **⇧M** or the ball above the layers adds one above the selected layer, with **pick** on so the next clicks set its zone. Put the main material first without a zone, then the others with theirs.

**save** in a folder's panel keeps it as your own material, for every project. Parts, islands, paint and stamps stay with the model.
