# texxer bridge: sends a .blend's objects to texxer and shows the texture texxer
# paints on them. Install: Preferences > Add-ons > Install from Disk, this file.

bl_info = {
    "name": "texxer bridge",
    "author": "texxer",
    "version": (0, 1, 0),
    "blender": (4, 2, 0),
    "location": "View3D > Sidebar > texxer",
    "description": "Paint the objects in texxer, live",
    "category": "Paint",
}

import json
import os
import sys
import urllib.parse
import urllib.request

import bpy

# What the last poll saw, so only changes act.
state = {"connected": False, "texture": -1, "sync": None, "blend": ""}


def config_dir():
    if sys.platform == "darwin":
        return os.path.expanduser("~/Library/Application Support")
    if sys.platform == "win32":
        return os.environ.get("APPDATA", "")
    return os.environ.get("XDG_CONFIG_HOME") or os.path.expanduser("~/.config")


def call(path, body=None, timeout=0.3):
    """A request to texxer, None when it isn't running."""
    try:
        with open(os.path.join(config_dir(), "texxer", "bridge.json")) as f:
            info = json.load(f)
    except (OSError, ValueError):
        return None
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(f"http://127.0.0.1:{info['port']}{path}", data=data, headers={"X-Texxer-Token": info["token"]})
    try:
        with urllib.request.urlopen(req, timeout=timeout) as res:
            text = res.read()
            return json.loads(text) if text else {}
    except (OSError, ValueError):
        return None


def blend_path():
    return bpy.path.abspath(bpy.data.filepath) if bpy.data.filepath else ""


def blend_name():
    return os.path.splitext(os.path.basename(bpy.data.filepath))[0]


def synced_objects(scene):
    names = [n for n in scene.texxer_objects.split("\n") if n]
    return [o for o in (bpy.data.objects.get(n) for n in names) if o and o.type == "MESH"]


# Materials: the objects show one texxer material while the texture is on; their
# own stay remembered on them and come back when it is off, and for the export to
# texxer, which reads parts and colors from them.

def material_name():
    return "texxer " + blend_name()


def show_material(objects, mat):
    for o in objects:
        if "texxer_own" not in o:
            o["texxer_own"] = [s.material.name if s.material else "" for s in o.material_slots]
        if not o.material_slots:
            o.data.materials.append(mat)
        for s in o.material_slots:
            s.material = mat


def restore_materials(objects):
    for o in objects:
        own = o.get("texxer_own")
        if own is None:
            continue
        if not own:
            o.data.materials.clear()
        for s, name in zip(o.material_slots, own):
            s.material = bpy.data.materials.get(name) if name else None
        del o["texxer_own"]


def image(path, existing):
    rel = bpy.path.relpath(path)
    if existing and bpy.path.abspath(existing.filepath) == path:
        existing.reload()
        return existing
    img = bpy.data.images.load(path, check_existing=True)
    img.filepath = rel
    img.reload()
    return img


def socket(node, *names):
    for n in names:
        if n in node.inputs:
            return node.inputs[n]


def update_material(s):
    """The texxer material with the latest texture: nearest pixels, no shine, glow from the emissive map."""
    name = material_name()
    mat = bpy.data.materials.get(name) or bpy.data.materials.new(name)
    mat.use_nodes = True
    nodes = mat.node_tree.nodes
    links = mat.node_tree.links

    if not mat.get("texxer"):
        nodes.clear()
        out = nodes.new("ShaderNodeOutputMaterial")
        out.location = (300, 0)
        bsdf = nodes.new("ShaderNodeBsdfPrincipled")
        bsdf.name = "bsdf"
        links.new(bsdf.outputs[0], out.inputs[0])
        socket(bsdf, "Roughness").default_value = 1.0
        spec = socket(bsdf, "Specular IOR Level", "Specular")
        if spec:
            spec.default_value = 0.0
        for key, y in (("color", 150), ("emissive", -200)):
            tex = nodes.new("ShaderNodeTexImage")
            tex.name = key
            tex.interpolation = "Closest"
            tex.location = (-400, y)
        links.new(nodes["color"].outputs["Color"], socket(bsdf, "Base Color"))
        mat["texxer"] = 1
    if "uv" not in nodes:
        uvmap = nodes.new("ShaderNodeUVMap")
        uvmap.name = "uv"
        uvmap.location = (-650, 0)
        for key in ("color", "emissive"):
            links.new(uvmap.outputs["UV"], nodes[key].inputs["Vector"])
    nodes["uv"].uv_map = UV if s.get("uv") in ("texxer", "locked") else ""

    bsdf = nodes["bsdf"]
    nodes["color"].image = image(s["color"], nodes["color"].image)
    glow = socket(bsdf, "Emission Color", "Emission")
    if s.get("emissive"):
        nodes["emissive"].image = image(s["emissive"], nodes["emissive"].image)
        links.new(nodes["emissive"].outputs["Color"], glow)
        socket(bsdf, "Emission Strength").default_value = s.get("glow") or 1.0
    else:
        for l in list(glow.links):
            links.remove(l)
        socket(bsdf, "Emission Strength").default_value = 0.0
    return mat


def set_material_uv(name):
    """The texxer material reads this UV map; empty for the one Blender renders with."""
    mat = bpy.data.materials.get(material_name())
    if mat and mat.node_tree and "uv" in mat.node_tree.nodes:
        mat.node_tree.nodes["uv"].uv_map = name


def on_show(self, context):
    objects = synced_objects(context.scene)
    mat = bpy.data.materials.get(material_name())
    if self.texxer_show and mat:
        show_material(objects, mat)
    else:
        restore_materials(objects)


def export(context, objects, path, apply=True):
    view_layer = context.view_layer
    selected = [o for o in view_layer.objects if o.select_get()]
    active = view_layer.objects.active
    shown = context.scene.texxer_show
    restore_materials(objects)
    try:
        for o in view_layer.objects:
            o.select_set(o in objects)
        bpy.ops.export_scene.gltf(
            filepath=path,
            export_format="GLB",
            use_selection=True,
            export_apply=apply,
            export_materials="EXPORT",
            export_image_format="NONE",
            export_animations=False,
            export_cameras=False,
            export_lights=False,
        )
    finally:
        for o in view_layer.objects:
            o.select_set(o in selected)
        view_layer.objects.active = active
        mat = bpy.data.materials.get(material_name())
        if shown and mat:
            show_material(objects, mat)


CORNERS = "texxer corners"
UV = "texxer"


def meshes(objects):
    """Each object's mesh once, as objects can share one."""
    seen = {}
    for o in objects:
        seen.setdefault(o.data.name, o.data)
    return seen.values()


def unwrap(context, objects, path, size):
    """texxer lays out the objects without their modifiers, which then carry the
    UVs through as they do Blender's; they land in the UV map "texxer". The
    corners go there numbered, so the UVs come back to the right loops."""
    sets = {}
    for me in meshes(objects):
        layer = me.uv_layers.new(name=CORNERS)
        if layer is None:
            return f"{me.name} has no room for another UV map"
        ids = [0.0] * (2 * len(me.loops))
        ids[0::2] = [float(i) for i in range(len(me.loops))]
        layer.data.foreach_set("uv", ids)
    for o in objects:
        sets[o.name] = o.data.uv_layers.find(CORNERS)
    try:
        export(context, objects, path, apply=False)
    finally:
        for me in meshes(objects):
            me.uv_layers.remove(me.uv_layers[CORNERS])

    uvs = call("/unwrap", {"glb": path, "sets": sets, "size": size}, timeout=120)
    if uvs is None:
        return "texxer could not lay out the UVs"
    for o in objects:
        me = o.data
        data = uvs.get(o.name)
        if not data:
            continue
        layer = me.uv_layers.get(UV) or me.uv_layers.new(name=UV)
        if layer is None:
            return f"{me.name} has no room for another UV map"
        n = 2 * len(me.loops)
        layer.data.foreach_set("uv", (data + [0.0] * n)[:n])
    return None


def uv_sets(objects, uv):
    """The UV set texxer paints on, per object: texxer's map, or Blender's for render."""
    out = {}
    for o in objects:
        layers = o.data.uv_layers
        at = layers.find(UV) if uv in ("texxer", "locked") else -1
        if at < 0:
            at = next((i for i, l in enumerate(layers) if l.active_render), 0)
        out[o.name] = at
    return out


def sync(context):
    """Writes the objects as a GLB beside the .blend and tells texxer. An error text, or None."""
    blend = blend_path()
    if not blend:
        return "save the .blend first"
    mode = context.mode
    if mode != "OBJECT":
        bpy.ops.object.mode_set(mode="OBJECT")
    try:
        return send(context, blend)
    finally:
        if mode == "EDIT_MESH" and context.view_layer.objects.active:
            bpy.ops.object.mode_set(mode="EDIT")


def send(context, blend):
    scene = context.scene
    objects = synced_objects(scene)
    if not objects:
        objects = [o for o in context.selected_objects if o.type == "MESH"]
        if not objects:
            objects = [o for o in scene.objects if o.type == "MESH" and o.visible_get()]
        scene.texxer_objects = "\n".join(o.name for o in objects)
    if not objects:
        return "no mesh objects"

    s = call("/state?blend=" + urllib.parse.quote(blend))
    if s is None:
        return "texxer is not running"
    uv = s.get("uv") or "blender"
    base = os.path.join(os.path.dirname(blend), ".texxer", blend_name())
    os.makedirs(os.path.dirname(base), exist_ok=True)
    if uv == "texxer":
        err = unwrap(context, objects, base + ".corners.glb", s.get("size") or 512)
        if err:
            return err
    set_material_uv(UV if uv in ("texxer", "locked") else "")

    written = base + ".sync.glb"
    export(context, objects, written)
    body = {"blend": blend, "glb": base + ".glb", "name": blend_name(), "written": written, "uvSets": uv_sets(objects, uv)}
    if call("/sync", body, timeout=10) is None:
        return "texxer is not running"
    return None


def in_view(fn):
    """Runs an operator-calling function from a timer, in a window's context."""
    wm = bpy.context.window_manager
    if not wm.windows:
        return
    win = wm.windows[0]
    area = next((a for a in win.screen.areas if a.type == "VIEW_3D"), None)
    with bpy.context.temp_override(window=win, area=area):
        fn(bpy.context)


def poll():
    blend = blend_path()
    if blend != state["blend"]:
        state.update(blend=blend, texture=-1, sync=None)
    s = call("/state?blend=" + urllib.parse.quote(blend)) if blend else None
    connected = s is not None
    if connected != state["connected"]:
        state["connected"] = connected
        for win in bpy.context.window_manager.windows:
            for area in win.screen.areas:
                area.tag_redraw()
    if not s:
        return 0.5

    if s["texture"] != state["texture"] and s.get("color") and os.path.exists(s["color"]):
        state["texture"] = s["texture"]
        mat = update_material(s)
        scene = bpy.context.scene
        if scene.texxer_show:
            show_material(synced_objects(scene), mat)

    # texxer's sync button; a count seen for the first time is only taken note of.
    if state["sync"] is not None and s["sync"] != state["sync"]:
        in_view(sync)
    state["sync"] = s["sync"]
    return 0.5


class TEXXER_OT_sync(bpy.types.Operator):
    bl_idname = "texxer.sync"
    bl_label = "sync"
    bl_description = "Send the objects to texxer"

    def execute(self, context):
        err = sync(context)
        if err:
            self.report({"WARNING"}, err)
            return {"CANCELLED"}
        self.report({"INFO"}, "sent to texxer")
        return {"FINISHED"}


class TEXXER_OT_use_selection(bpy.types.Operator):
    bl_idname = "texxer.use_selection"
    bl_label = "use selection"
    bl_description = "Paint the selected objects in texxer, sharing one texture"

    def execute(self, context):
        scene = context.scene
        old = synced_objects(scene)
        restore_materials(old)
        objects = [o for o in context.selected_objects if o.type == "MESH"]
        scene.texxer_objects = "\n".join(o.name for o in objects)
        mat = bpy.data.materials.get(material_name())
        if scene.texxer_show and mat:
            show_material(objects, mat)
        return {"FINISHED"}


class TEXXER_PT_panel(bpy.types.Panel):
    bl_label = "texxer"
    bl_space_type = "VIEW_3D"
    bl_region_type = "UI"
    bl_category = "texxer"

    def draw(self, context):
        col = self.layout.column()
        if not bpy.data.filepath:
            col.label(text="save the .blend first")
        elif state["connected"]:
            col.label(text="texxer: live", icon="LINKED")
        else:
            col.label(text="texxer is not running", icon="UNLINKED")

        objects = synced_objects(context.scene)
        row = col.row()
        row.label(text=f"{len(objects)} objects" if objects else "selection, on first sync")
        row.operator("texxer.use_selection", text="", icon="RESTRICT_SELECT_OFF")
        col.operator("texxer.sync", icon="FILE_REFRESH")
        col.prop(context.scene, "texxer_show")


classes = (TEXXER_OT_sync, TEXXER_OT_use_selection, TEXXER_PT_panel)


def register():
    for c in classes:
        bpy.utils.register_class(c)
    bpy.types.Scene.texxer_objects = bpy.props.StringProperty(name="objects")
    bpy.types.Scene.texxer_show = bpy.props.BoolProperty(name="show texture", default=True, update=on_show)
    bpy.app.timers.register(poll, first_interval=0.5, persistent=True)


def unregister():
    if bpy.app.timers.is_registered(poll):
        bpy.app.timers.unregister(poll)
    del bpy.types.Scene.texxer_show
    del bpy.types.Scene.texxer_objects
    for c in reversed(classes):
        bpy.utils.unregister_class(c)
