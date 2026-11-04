# Photos and snippets

Paint from real photos: cut snippets from them, calm them to the texture's look, and stamp them onto the model.

## Cutting

Switch to **photos** at the top. Drop photos onto the window or use **add photos**; they are copied into the project (`<project>.snippets/`). Pick one, then on the photo:

- **drag** a rectangle for a straight cut,
- **click four corners** for a skewed one, like a wall seen at an angle; it gets deskewed,
- drag corners to fit, drag inside to move, **⌫** deletes, **F** fits the photo. Two fingers pan, pinch zooms.

## Look

The panel on the right shows the snippet as it will be stamped:

- **pixels**: its long side. Keep it near the texel density, so it isn't sharper than the rest.
- **width**: widens what the photo foreshortened, as a wall seen at an angle.
- **calm**, **saturation**, **brightness**: flatter and quieter than the photo, closer to the texture.
- **monochrome**: only brightness around mid gray; it goes on a relief layer and tints what is below.

The finish crunches snippets like everything else, auto palette included.

## Categories

**project** holds the project's photos. **+** makes a global category, for snippets every project can use; a snippet taken from one is copied into the project.

## Stamping

With the stamp tool (**S**) the snippets show at the bottom. Click one: it becomes the stamp shape, on a **photo** layer that takes its colors from the stamps (**photo relief** for monochrome ones). Place, size and turn it as any stamp. **⌘click** fills the surface under the cursor, bounded by hard edges, with it; **tiles** repeats it across, rows following so tiles keep their shape. Axis views from the gizmo are ortho, which keeps projected snippets true.
