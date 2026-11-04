# Brush and stamps

![](img/stamps.png)

**B** brush, **S** stamp, **Esc** back to the view tool. Both paint into the selected layer and add the mask they need.

## Brush

Paints into the layer's paint mask, or with **colors** the layer's own colors, many in one layer (see layers). **[ ]** size, **⌥** or **E** erase, **⇧click** a straight line from the last stroke, along the nearest axis (**⇧⌘click** free; holding **⇧** shows it in the axis color), **⌘click** fills the surface under the cursor, bounded by hard edges (with **⌥** empties it), **X** mirror across x.

- **hardness**: how far in from the rim the brush fades.
- **strength**: the most a stroke paints; painting over it doesn't go further.
- **soft** or **pixel**: pixel paints whole texels of the texture, hard, **pixels** across (**[ ]** one more or less), for details that follow a projected photo.
- **grit**: breaks the brush up with noise that sits on the model, for rust, dirt and chips.
- **grunge**: the brush reveals a grunge map, as grunge masks project it; more strength shows more of it.

The paint mask's **amount** scales what is painted; 0.5 keeps it as it is.

## Stamps

Click to place, drag to move, **⌥click** places a new one over others. A shape you pick is for the next stamp; **⌥click** a shape to swap the picked stamp's. Shapes, relief stamps (for the relief blend), decals, your PNGs (**+ png**) and text: **pixel** is a 5×7 font that stays readable at low sizes.

The picked stamp has grips: drag the one above it to turn it (**⇧** in 15° steps), a corner to pull it over the surface while the opposite corner stays (**⇧** keeps its aspect). **R** rotate 15° (**⇧R** back), **G** snap to a grid on the surface, arrows nudge, **D** duplicate, **⇧click** a row from the picked stamp, **⌘click** fill the surface under the cursor, **⌫** delete.

Layers with **colors from image stamps** take the colors of decals and images instead of their own.
