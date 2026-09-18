"""Build a detailed, original soft-toy Chinese baby dragon pet with Blender."""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

import bpy  # noqa: E402
from mathutils import Euler  # noqa: E402
from pet_pipeline import (  # noqa: E402
    cone,
    ellipsoid,
    export_glb,
    make_material,
    render_poster,
    reset_scene,
    tube,
)


reset_scene()

# The palette keeps the dragon recognizable at desktop-pet scale:
# cool blue body, pearl belly, lavender mane, and warm face accents.
body = make_material("Dragon blue", (0.10, 0.53, 0.78), 0.02, 0.26)
body_dark = make_material("Dragon blue shadow", (0.045, 0.26, 0.48), 0.04, 0.28)
body_mid = make_material("Dragon blue midtone", (0.16, 0.66, 0.87), 0.01, 0.24)
body_light = make_material("Dragon blue highlight", (0.38, 0.80, 0.94), 0.0, 0.22)
lavender = make_material("Lavender mane", (0.55, 0.54, 0.82), 0.01, 0.28)
lavender_light = make_material("Lavender mane light", (0.73, 0.72, 0.94), 0.0, 0.25)
cream = make_material("Pearl belly", (0.94, 0.95, 0.88), 0.0, 0.30)
cream_shadow = make_material("Pearl belly shadow", (0.76, 0.86, 0.88), 0.0, 0.30)
blush = make_material("Rosy blush", (1.0, 0.26, 0.36), 0.0, 0.25)
blush_soft = make_material("Soft blush", (1.0, 0.50, 0.57), 0.0, 0.30)
ink = make_material("Deep glossy eyes", (0.004, 0.012, 0.035), 0.05, 0.08)
eye_rim = make_material("Eye rim blue", (0.035, 0.28, 0.58), 0.02, 0.18)
white = make_material("Eye highlights", (1.0, 1.0, 0.98), 0.0, 0.12)
horn = make_material("Warm ivory horns", (0.90, 0.91, 0.81), 0.04, 0.27)
horn_shadow = make_material("Horn shadow", (0.67, 0.75, 0.72), 0.02, 0.31)
gold = make_material("Golden horn tips", (0.98, 0.68, 0.18), 0.18, 0.22)
mouth = make_material("Mouth", (0.27, 0.035, 0.08), 0.0, 0.24)
tongue = make_material("Tongue", (1.0, 0.24, 0.37), 0.0, 0.22)

root = bpy.data.objects.new("ChineseDragon", None)
bpy.context.collection.objects.link(root)


def rig(name, location, rotation=(0.0, 0.0, 0.0), parent=None):
    obj = bpy.data.objects.new(name, None)
    bpy.context.collection.objects.link(obj)
    obj.location = location
    obj.rotation_euler = Euler(rotation, "XYZ")
    if parent is not None:
        obj.parent = parent
    return obj


def local_ellipsoid(name, position, scale, material, parent, rotation=(0.0, 0.0, 0.0)):
    obj = ellipsoid(name, position, scale, material, parent)
    obj.rotation_euler = Euler(rotation, "XYZ")
    return obj


def local_tube(name, points, radii, base_radius, material, parent):
    return tube(name, points, radii, base_radius, material, parent)


def local_cone(name, position, radius, depth, material, parent, rotation=(0.0, 0.0, 0.0)):
    return cone(name, position, radius, depth, material, rotation, parent)


def scale_plate(name, position, scale, material, parent, rotation=(0.0, 0.0, 0.0)):
    return local_ellipsoid(name, position, scale, material, parent, rotation)


body_rig = rig("BodyRig", (0.0, 0.02, 0.82), parent=root)
head_rig = rig("HeadRig", (0.04, -0.02, 1.70), rotation=(0.0, 0.0, -0.18), parent=root)
tail_rig = rig("TailRig", (-0.08, 0.20, 0.40), rotation=(0.0, 0.0, 0.06), parent=root)
left_arm_rig = rig("LeftArmRig", (-0.43, -0.12, 1.02), rotation=(0.0, 0.0, -0.12), parent=root)
right_arm_rig = rig("RightArmRig", (0.43, -0.12, 1.02), rotation=(0.0, 0.0, 0.10), parent=root)

# Body: soft rounded torso with a visible shoulder mantle.
local_ellipsoid("Torso", (0.0, 0.04, 0.0), (0.56, 0.44, 0.68), body, body_rig)
local_ellipsoid("Shoulder mantle", (0.0, 0.02, 0.36), (0.47, 0.38, 0.34), body_dark, body_rig)
local_ellipsoid("Chest glow", (0.0, -0.42, 0.22), (0.35, 0.10, 0.37), body_mid, body_rig)

# Belly scales are individually modeled so they read as overlapping plates,
# rather than a single white stripe when the pet is rendered small.
belly_rows = [
    (-0.52, 0.30, 0.27, 0.16),
    (-0.50, 0.48, 0.25, 0.15),
    (-0.47, 0.65, 0.23, 0.145),
    (-0.43, 0.81, 0.21, 0.13),
    (-0.38, 0.96, 0.19, 0.12),
    (-0.31, 1.08, 0.16, 0.105),
]
for index, (y, z, width, height) in enumerate(belly_rows):
    scale_plate(
        f"BellyScale_{index}",
        (0.0, y, z - 0.82),
        (width, 0.085, height),
        cream if index % 2 == 0 else cream_shadow,
        body_rig,
        rotation=(0.10, 0.0, 0.0),
    )
    if index < 4:
        for side in (-1, 1):
            scale_plate(
                f"BellyScale_{index}_{side}",
                (side * width * 0.72, y + 0.012, z - 0.82),
                (width * 0.42, 0.068, height * 0.72),
                cream_shadow,
                body_rig,
                rotation=(0.12, side * 0.12, side * 0.10),
            )

# Rounded haunches and feet anchor the seated silhouette.
for side in (-1, 1):
    local_ellipsoid(f"Haunch_{side}", (side * 0.38, 0.06, -0.28), (0.32, 0.34, 0.30), body, body_rig)
    local_ellipsoid(f"Foot_{side}", (side * 0.36, -0.30, -0.52), (0.29, 0.28, 0.17), body_mid, body_rig)
    local_ellipsoid(f"FootPad_{side}", (side * 0.36, -0.52, -0.52), (0.19, 0.075, 0.095), cream_shadow, body_rig)
    for toe in (-1, 0, 1):
        local_ellipsoid(
            f"FootClaw_{side}_{toe}",
            (side * 0.36 + toe * 0.085, -0.59, -0.51),
            (0.048, 0.032, 0.060),
            horn,
            body_rig,
        )

# Shoulder scale fans give the arms a dragon-like surface treatment.
for side in (-1, 1):
    for index, (z, offset, size) in enumerate(
        ((0.35, 0.02, 0.14), (0.20, 0.04, 0.12), (0.06, 0.06, 0.10))
    ):
        scale_plate(
            f"ShoulderScale_{side}_{index}",
            (side * (0.44 + offset), -0.22, z),
            (size, 0.045, size * 1.15),
            body_light if index == 0 else body_mid,
            body_rig,
            rotation=(0.0, side * 0.25, side * 0.15),
        )

# Tail curls around the base, with a larger ornamental tuft at the tip.
tail_points = [
    (0.08, 0.08, 0.12),
    (-0.30, 0.16, -0.02),
    (-0.70, 0.18, -0.25),
    (-0.86, 0.10, -0.48),
    (-0.65, -0.03, -0.58),
    (-0.34, -0.10, -0.52),
]
local_tube("Tail", tail_points, [1.0, 1.12, 1.05, 0.82, 0.55, 0.15], 0.19, body_dark, tail_rig)
for index, (position, scale, material) in enumerate(
    (
        ((-0.82, 0.10, -0.40), (0.18, 0.15, 0.16), body_mid),
        ((-0.92, 0.08, -0.53), (0.14, 0.12, 0.18), body_light),
        ((-0.78, 0.02, -0.64), (0.15, 0.12, 0.12), lavender),
        ((-0.57, -0.08, -0.65), (0.13, 0.10, 0.10), body_light),
    )
):
    local_ellipsoid(f"TailTuft_{index}", position, scale, material, tail_rig)

# Arms: one paw waves slightly higher to create personality and asymmetry.
for side, arm_parent, lift in ((-1, left_arm_rig, 0.12), (1, right_arm_rig, 0.0)):
    local_ellipsoid("UpperArm", (0.0, 0.02, -0.07 + lift), (0.18, 0.20, 0.34), body, arm_parent)
    local_ellipsoid("Forearm", (side * 0.015, -0.20, -0.28 + lift), (0.16, 0.17, 0.24), body_mid, arm_parent)
    local_ellipsoid("Palm", (side * 0.02, -0.34, -0.44 + lift), (0.15, 0.13, 0.14), body_light, arm_parent)
    for index, z in enumerate((0.10, -0.02, -0.14)):
        scale_plate(
            f"ArmScale_{side}_{index}",
            (side * 0.02, -0.205, z + lift),
            (0.13 - index * 0.015, 0.035, 0.075),
            body_light if index == 0 else body_mid,
            arm_parent,
            rotation=(0.05, 0.0, side * 0.10),
        )
    for finger in (-1, 0, 1):
        local_ellipsoid(
            f"FingerClaw_{side}_{finger}",
            (side * 0.02 + finger * 0.055, -0.46, -0.45 + lift),
            (0.035, 0.025, 0.06),
            horn,
            arm_parent,
        )

# Head: oversized and slightly turned, with a soft cheek/muzzle silhouette.
local_ellipsoid("Head", (0.0, 0.0, 0.0), (0.73, 0.59, 0.64), body, head_rig)
local_ellipsoid("HeadSideShadow", (0.0, 0.12, -0.05), (0.68, 0.51, 0.58), body_dark, head_rig)
local_ellipsoid("FacePlane", (0.0, -0.42, -0.02), (0.57, 0.16, 0.49), body_mid, head_rig)
local_ellipsoid("MuzzleBase", (0.0, -0.57, -0.18), (0.37, 0.18, 0.20), cream_shadow, head_rig)
local_ellipsoid("MuzzleLeft", (-0.17, -0.64, -0.16), (0.22, 0.15, 0.15), cream, head_rig)
local_ellipsoid("MuzzleRight", (0.17, -0.64, -0.16), (0.22, 0.15, 0.15), cream, head_rig)
local_ellipsoid("NoseBridge", (0.0, -0.70, -0.04), (0.12, 0.075, 0.08), body_light, head_rig)
local_ellipsoid("Nose", (0.0, -0.76, -0.04), (0.075, 0.04, 0.052), blush, head_rig)
for side in (-1, 1):
    local_ellipsoid(f"Nostril_{side}", (side * 0.037, -0.795, -0.025), (0.017, 0.010, 0.012), mouth, head_rig)

# The smile is a dark crescent, with tongue and two tiny fangs peeking out.
local_ellipsoid("OpenMouth", (0.0, -0.675, -0.36), (0.20, 0.035, 0.11), mouth, head_rig)
local_ellipsoid("Tongue", (0.025, -0.718, -0.39), (0.085, 0.025, 0.048), tongue, head_rig)
for side in (-1, 1):
    local_ellipsoid(
        f"Fang_{side}",
        (side * 0.095, -0.718, -0.31),
        (0.025, 0.018, 0.065),
        cream,
        head_rig,
        rotation=(0.0, side * 0.08, side * 0.08),
    )

# Large eyes use a colored rim, cream sclera, deep glassy pupils, and
# two different highlights to keep the expression alive at 112 x 132 px.
for side in (-1, 1):
    eye_x = side * 0.275
    local_ellipsoid(f"EyeRim_{side}", (eye_x, -0.45, 0.16), (0.255, 0.075, 0.285), eye_rim, head_rig)
    local_ellipsoid(f"EyeWhite_{side}", (eye_x, -0.505, 0.16), (0.222, 0.060, 0.252), cream, head_rig)
    local_ellipsoid(f"Pupil_{side}", (eye_x, -0.565, 0.16), (0.155, 0.045, 0.190), ink, head_rig)
    local_ellipsoid(f"EyeGlow_{side}", (eye_x + side * 0.035, -0.602, 0.235), (0.054, 0.018, 0.070), white, head_rig)
    local_ellipsoid(f"EyeDot_{side}", (eye_x - side * 0.055, -0.605, 0.105), (0.022, 0.012, 0.030), white, head_rig)
    local_ellipsoid(f"Blush_{side}", (side * 0.44, -0.52, -0.10), (0.145, 0.032, 0.085), blush_soft, head_rig)
    local_ellipsoid(
        f"Brow_{side}",
        (side * 0.27, -0.50, 0.49),
        (0.20, 0.035, 0.055),
        lavender,
        head_rig,
        rotation=(0.0, side * 0.10, side * 0.16),
    )

# Forehead cloud marks and small scales add a Chinese ornamental cue.
for side in (-1, 1):
    local_tube(
        f"CloudMark_{side}",
        [
            (side * 0.07, -0.615, 0.47),
            (side * 0.16, -0.635, 0.57),
            (side * 0.27, -0.62, 0.50),
            (side * 0.23, -0.63, 0.42),
        ],
        [0.8, 1.0, 0.7, 0.12],
        0.025,
        lavender_light,
        head_rig,
    )
for index, x in enumerate((-0.10, 0.0, 0.10)):
    local_ellipsoid(
        f"ForeheadScale_{index}",
        (x, -0.615, 0.36 + abs(x) * 0.10),
        (0.055, 0.022, 0.075),
        body_light,
        head_rig,
        rotation=(0.10, 0.0, 0.0),
    )

# Layered mane: rounded locks plus pointed tufts produce a fluffy silhouette.
mane_locks = [
    ("CrownCenter", (0.0, 0.18, 0.56), (0.25, 0.22, 0.25), body_light, 0.0),
    ("CrownLeft", (-0.25, 0.15, 0.52), (0.22, 0.19, 0.30), lavender_light, -0.22),
    ("CrownRight", (0.25, 0.15, 0.52), (0.22, 0.19, 0.30), lavender_light, 0.22),
    ("CheekLeft", (-0.59, 0.10, 0.13), (0.19, 0.17, 0.30), body_light, -0.32),
    ("CheekRight", (0.59, 0.10, 0.13), (0.19, 0.17, 0.30), body_light, 0.32),
    ("JawLeft", (-0.56, 0.08, -0.21), (0.18, 0.16, 0.24), lavender, -0.40),
    ("JawRight", (0.56, 0.08, -0.21), (0.18, 0.16, 0.24), lavender, 0.40),
    ("NeckLeft", (-0.47, 0.10, -0.49), (0.16, 0.14, 0.24), body_light, -0.50),
    ("NeckRight", (0.47, 0.10, -0.49), (0.16, 0.14, 0.24), body_light, 0.50),
]
for name, position, scale, material, angle in mane_locks:
    local_ellipsoid(name, position, scale, material, head_rig, rotation=(0.0, angle * 0.25, angle))

for side in (-1, 1):
    for index, (x, z, length) in enumerate(
        ((0.36, 0.64, 0.32), (0.50, 0.46, 0.25), (0.58, 0.22, 0.20))
    ):
        base_x = side * x
        local_tube(
            f"ManeTuft_{side}_{index}",
            [
                (base_x, 0.18, z),
                (base_x + side * 0.09, 0.20, z + length * 0.55),
                (base_x + side * 0.16, 0.15, z + length),
            ],
            [1.0, 0.78, 0.10],
            0.075 if index == 0 else 0.060,
            lavender if index == 1 else body_light,
            head_rig,
        )

# Front-facing locks keep the mane readable instead of letting it disappear
# behind the head from the small desktop camera angle.
for side in (-1, 1):
    local_tube(
        f"FrontMane_{side}",
        [
            (side * 0.48, -0.28, 0.46),
            (side * 0.67, -0.30, 0.56),
            (side * 0.78, -0.24, 0.74),
        ],
        [1.0, 0.82, 0.10],
        0.080,
        lavender_light,
        head_rig,
    )
    local_tube(
        f"JawMane_{side}",
        [
            (side * 0.48, -0.26, -0.24),
            (side * 0.68, -0.28, -0.34),
            (side * 0.73, -0.20, -0.48),
        ],
        [1.0, 0.78, 0.12],
        0.070,
        lavender,
        head_rig,
    )

# Ears, ivory horns, and gold tips frame the face.
for side in (-1, 1):
    local_ellipsoid(f"Ear_{side}", (side * 0.64, 0.07, 0.40), (0.18, 0.12, 0.23), body_mid, head_rig)
    local_ellipsoid(f"EarInner_{side}", (side * 0.67, -0.02, 0.40), (0.095, 0.045, 0.14), lavender, head_rig)
    local_cone(
        f"EarFin_{side}",
        (side * 0.72, 0.08, 0.58),
        0.13,
        0.34,
        body_light,
        head_rig,
        rotation=(0.0, side * 0.52, side * 0.08),
    )
    local_tube(
        f"Horn_{side}",
        [
            (side * 0.28, 0.15, 0.50),
            (side * 0.38, 0.17, 0.80),
            (side * 0.30, 0.18, 1.04),
            (side * 0.17, 0.18, 1.15),
        ],
        [1.0, 0.94, 0.64, 0.16],
        0.070,
        horn,
        head_rig,
    )
    local_ellipsoid(f"HornTip_{side}", (side * 0.17, 0.18, 1.16), (0.055, 0.055, 0.055), gold, head_rig)

# Long whiskers are kept thin but curved enough to survive the desktop render.
for side in (-1, 1):
    local_tube(
        f"Whisker_{side}",
        [
            (side * 0.20, -0.72, -0.15),
            (side * 0.45, -0.76, -0.19),
            (side * 0.68, -0.62, -0.08),
            (side * 0.80, -0.46, 0.08),
        ],
        [1.0, 0.82, 0.45, 0.08],
        0.020,
        horn,
        head_rig,
    )

# A pair of small cloud curls behind the cheeks reinforces the Chinese motif.
for side in (-1, 1):
    local_tube(
        f"CheekCloud_{side}",
        [
            (side * 0.53, 0.09, 0.00),
            (side * 0.72, 0.10, 0.12),
            (side * 0.72, 0.10, 0.28),
            (side * 0.60, 0.10, 0.35),
        ],
        [0.80, 1.0, 0.60, 0.10],
        0.035,
        lavender_light,
        head_rig,
    )

export_glb(root, "chinese-dragon.glb")
render_poster(
    "chinese-dragon.png",
    look_at=(0.0, 0.0, 1.30),
    camera=(1.95, -7.6, 2.86),
    ortho_scale=3.25,
)
