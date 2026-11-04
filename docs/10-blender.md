# Blender

Model in Blender, paint in texxer, see the texture in Blender as you paint.

1. Install `tools/blender/texxer_bridge.py`: Preferences › Add-ons › Install from Disk.
2. Save the `.blend`, keep texxer running.
3. In the 3D view's sidebar (**N**), tab **texxer**: **sync**.

texxer opens the objects as a model, in the open project or a new one beside the `.blend`. Sync again after modelling, in Blender or with **sync** on the model in texxer. Layers, stamps and painted masks stay.

- **Objects**: the selected ones on the first sync, all visible meshes if none are. **use selection** sets them anew. They share one texture; each object is a part.
- **Material**: the objects show one material, `texxer <name>`, with the texture (nearest pixels, no shine) and the emissive map at the export's strength. Their own materials stay remembered on them and come back with **show texture** off. texxer still gets their own, for parts and model colors.
- **Textures** go to `textures/` beside the `.blend` (the folder on the model changes it), the same PNGs the export writes, so Blender links them and its own export takes them along.
- **UVs**: Blender's, for now. Unwrap in Blender when they overlap.

## Export

From Blender: its glTF export carries the texxer material and its PNGs.

From texxer: **⌘E** asks once for a GLB of its own, as the model's file is the add-on's copy (in `.texxer/` beside the `.blend`).
