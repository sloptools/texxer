# Look

Settings in **crunch**, **baked light** and the lights are the project's, shared by all its models.

- **texture**: size of the model's texture. **crunch** renders it larger, then shrinks and sharpens it.
- **edge width**, **cavity width**: how far the edges and cavity masks reach from an edge, per model. Small models with fine detail want them narrow.
- **detail maps**: for models with a normal or occlusion map (baked from a high-poly model), how much its grooves, bevels and shading count. The export carries both maps into texxer's layout.
- **palette**: none, auto (the best colors for the texture, kept steady while you edit), Quake, or a palette file. **dither** mixes neighbor colors.
- **baked light**: light and AO painted into the texture, so it reads in any engine light.
- **game light**: the in-game view (**I**): flat ambient, AO, a light from above, tone mapping like Godot.

**P** renders at game resolution (480, 360, 240 lines), with mipmaps like in the game. **T** holds a torch at the cursor. **H** headlight, **⇧B** bloom on and off; its strength is under view light.
