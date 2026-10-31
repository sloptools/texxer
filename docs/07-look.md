# Look

Settings in **crunch**, **baked light** and the lights are the project's, shared by all its models.

- **texture**: size of the model's texture. **crunch** renders it larger, then shrinks and sharpens it.
- **palette**: none, auto (the best colors for the texture), Quake, or a palette file. **dither** mixes neighbor colors.
- **baked light**: light and AO painted into the texture, so it reads in any engine light.
- **game light**: the in-game view (**I**): flat ambient, AO, a light from above, tone mapping like Godot.

**P** renders at game resolution (480, 360, 240 lines), with mipmaps like in the game. **T** holds a torch at the cursor. **H** headlight, **⇧B** bloom.
