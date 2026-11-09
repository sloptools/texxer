# Look

Under the model in the models tab, its own:

- **texture**: size of the model's texture. **crunch** renders it larger, then shrinks and sharpens it.
- **edge width**, **cavity width**: how far the edges and cavity masks reach from an edge. Small models with fine detail want them narrow.
- **detail maps**: for models with a normal or occlusion map (baked from a high-poly model), how much its grooves, bevels and shading count. The export carries both maps into texxer's layout.

In settings, shared by all models of the project, as are the lights:

- **palette**: none, auto (the best colors for the texture, kept steady while you edit; painted colors always get theirs), Quake, or a palette file. **dither** mixes neighbor colors.
- **baked light**: light and AO painted into the texture, so it reads in any engine light.

## View lights

The sun button under the gizmo sets the lights of the view; they don't change the texture. In the in-game view (**I**, flat ambient, AO, a light from above) it shows the game light: **tone** picks Godot's tone mapping: standard (linear, the default), reinhard, filmic, aces, agx.

**P** renders at game resolution (480, 360, 240 lines), with mipmaps like in the game. **T** holds a torch at the cursor. **H** headlight, **⇧B** bloom on and off; its strength is under the sun button.
