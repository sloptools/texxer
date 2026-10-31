# Layers

![](img/layers.png)

Layers stack bottom to top. A layer without masks covers everything.

- **color**: the layer's color. Swatches reuse colors of other layers.
- **tone**, **tone scale**: lighter and darker patches over the color, at a world size.
- **grain**: single texels a bit lighter or darker.
- **shine**: baked sheen and glints, for glass and gloss.
- **glows**: the color also goes into the emissive map.
- **blend**: normal, multiply, add, overlay, relief. Relief takes gray stamps as bumps: mid gray keeps the surface.
- **opacity**

**N** new, **D** duplicate, **⌫** delete, **V** show or hide, **↑ ↓** select, **⌥↑ ⌥↓** move.

## Folders

**⌘G** puts the layer into a new folder, **⇧⌘G** dissolves it. Drag layers onto a folder to put them in. Folders hide and show all their layers.
