# Projects and export

![](img/models.png)

A project is a `.texxer` file anywhere. It lists models; each has its own texture, layers and variants. Painted masks sit in a folder beside the project.

- **models** tab: **+** adds GLBs, or drop them in. Double-click renames, **×** removes from the project (the files stay).
- **⌘P** switches by name, **PgUp PgDn** steps through.
- The **+** on a model adds more GLBs that share its texture, like attachments. They are laid out side by side and painted together.

## Export

**⌘E** writes the texture into the model's GLB files in place, and PNGs beside them: `<model>.png`, emissive and variants.

The dot beside each model shows what it needs: dim when exported, teal when edited, not exported yet or its file changed (when Blender wrote it again), red when the file is missing. **⇧⌘E** exports all models that need it.
