"""Build an original soft-toy Chinese baby dragon pet with Blender."""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

import bpy  # noqa: E402
from pet_pipeline import (  # noqa: E402
    ellipsoid, export_glb, make_material, render_poster, reset_scene, tube,
)

reset_scene()

sky = make_material("Sky blue scales", (0.16, 0.64, 0.86), 0.08, 0.42)
sky_dark = make_material("Blue shadow scales", (0.08, 0.38, 0.64), 0.1, 0.42)
sky_light = make_material("Soft blue mane", (0.36, 0.78, 0.96), 0.04, 0.46)
lavender = make_material("Lavender accents", (0.62, 0.57, 0.86), 0.02, 0.48)
cream = make_material("Pearl belly", (0.94, 0.95, 0.88), 0.0, 0.5)
cream_blue = make_material("Blue belly edge", (0.66, 0.84, 0.93), 0.0, 0.48)
coral = make_material("Warm blush", (1.0, 0.38, 0.42), 0.0, 0.5)
ink = make_material("Glossy eyes", (0.008, 0.02, 0.06), 0.1, 0.12)
white = make_material("Eye shine", (1.0, 1.0, 0.98), 0.0, 0.18)
horn = make_material("Pearl horns", (0.91, 0.94, 0.9), 0.15, 0.28)
gold = make_material("Small gold details", (0.98, 0.72, 0.22), 0.35, 0.28)

root = bpy.data.objects.new("ChineseDragon", None)
bpy.context.collection.objects.link(root)

# Compact seated silhouette: a round torso, a curled tail, and oversized head.
ellipsoid("Torso", (0.0, 0.08, 0.78), (0.48, 0.40, 0.62), sky, root)
ellipsoid("Shoulder shadow", (0.0, 0.12, 1.13), (0.40, 0.32, 0.34), sky_dark, root)

belly = [
    (0.0, -0.34, 0.48, 0.27, 0.07, 0.18),
    (0.0, -0.38, 0.68, 0.25, 0.07, 0.17),
    (0.0, -0.40, 0.87, 0.22, 0.065, 0.16),
    (0.0, -0.40, 1.05, 0.19, 0.06, 0.14),
    (0.0, -0.36, 1.20, 0.15, 0.055, 0.12),
]
for index, (x, y, z, sx, sy, sz) in enumerate(belly):
    ellipsoid(f"Belly scale {index}", (x, y, z), (sx, sy, sz), cream, root)

# Large seated haunches and soft paws.
for side in (-1, 1):
    ellipsoid("Haunch", (side * 0.34, -0.01, 0.46), (0.30, 0.30, 0.27), sky, root)
    ellipsoid("Foot", (side * 0.32, -0.34, 0.27), (0.25, 0.22, 0.16), sky_light, root)
    for toe in (-1, 0, 1):
        ellipsoid("Toe", (side * 0.32 + toe * 0.075, -0.53, 0.27), (0.055, 0.035, 0.045), horn, root)
    ellipsoid("Arm", (side * 0.42, -0.21, 0.92), (0.15, 0.18, 0.30), sky, root)
    ellipsoid("Hand", (side * 0.42, -0.39, 0.72), (0.14, 0.12, 0.13), sky_light, root)
    for toe in (-1, 0, 1):
        ellipsoid("Finger", (side * 0.42 + toe * 0.045, -0.50, 0.70), (0.035, 0.025, 0.045), horn, root)

# A thick tail curls around the seated body and ends in a fluffy tuft.
tail_points = [
    (-0.16, 0.18, 0.52),
    (-0.58, 0.16, 0.36),
    (-0.82, 0.12, 0.18),
    (-0.65, 0.08, 0.08),
    (-0.34, 0.06, 0.12),
]
tube("Tail", tail_points, [1.0, 1.0, 0.82, 0.6, 0.28], 0.17, sky_dark, root)
for position, scale in [
    ((-0.82, 0.10, 0.18), (0.15, 0.13, 0.14)),
    ((-0.94, 0.10, 0.28), (0.11, 0.10, 0.16)),
    ((-0.91, 0.10, 0.08), (0.12, 0.10, 0.10)),
]:
    ellipsoid("Tail fluff", position, scale, sky_light, root)

# The face is intentionally oversized and rounded for the small desktop canvas.
ellipsoid("Head", (0.0, -0.06, 1.58), (0.57, 0.48, 0.50), sky, root)
ellipsoid("Muzzle", (0.0, -0.49, 1.43), (0.29, 0.19, 0.19), cream_blue, root)
ellipsoid("Muzzle left", (-0.13, -0.54, 1.47), (0.15, 0.11, 0.10), cream, root)
ellipsoid("Muzzle right", (0.13, -0.54, 1.47), (0.15, 0.11, 0.10), cream, root)
ellipsoid("Nose", (0.0, -0.67, 1.53), (0.075, 0.045, 0.05), coral, root)
ellipsoid("Smile", (0.0, -0.64, 1.32), (0.12, 0.028, 0.06), ink, root)
ellipsoid("Tongue", (0.0, -0.67, 1.29), (0.065, 0.025, 0.035), coral, root)
ellipsoid("Tooth left", (-0.055, -0.67, 1.36), (0.028, 0.022, 0.05), cream, root)
ellipsoid("Tooth right", (0.055, -0.67, 1.36), (0.028, 0.022, 0.05), cream, root)

# Large toy-like eyes: pearl eye whites, deep pupils, and two soft highlights.
for side in (-1, 1):
    ellipsoid("Eye white", (side * 0.205, -0.42, 1.70), (0.18, 0.065, 0.22), cream, root)
    ellipsoid("Eye pupil", (side * 0.205, -0.49, 1.70), (0.13, 0.045, 0.17), ink, root)
    ellipsoid("Eye shine big", (side * 0.245, -0.54, 1.78), (0.045, 0.018, 0.06), white, root)
    ellipsoid("Eye shine small", (side * 0.16, -0.545, 1.63), (0.022, 0.012, 0.028), white, root)
    ellipsoid("Blush", (side * 0.36, -0.46, 1.47), (0.13, 0.035, 0.075), coral, root)
    ellipsoid("Brow", (side * 0.22, -0.40, 1.98), (0.12, 0.04, 0.045), lavender, root)

# Soft layered mane, ears, and short antler-like horns create a fluffy silhouette.
for name, position, scale, material in [
    ("Mane crown", (0.0, 0.18, 2.08), (0.25, 0.16, 0.22), sky_light),
    ("Mane crown left", (-0.25, 0.16, 2.00), (0.18, 0.14, 0.24), lavender),
    ("Mane crown right", (0.25, 0.16, 2.00), (0.18, 0.14, 0.24), lavender),
    ("Mane cheek left", (-0.50, 0.10, 1.75), (0.18, 0.16, 0.25), sky_light),
    ("Mane cheek right", (0.50, 0.10, 1.75), (0.18, 0.16, 0.25), sky_light),
    ("Mane cheek low left", (-0.48, 0.10, 1.48), (0.15, 0.14, 0.20), lavender),
    ("Mane cheek low right", (0.48, 0.10, 1.48), (0.15, 0.14, 0.20), lavender),
    ("Mane neck left", (-0.38, 0.12, 1.20), (0.16, 0.13, 0.18), sky_light),
    ("Mane neck right", (0.38, 0.12, 1.20), (0.16, 0.13, 0.18), sky_light),
]:
    ellipsoid(name, position, scale, material, root)

for side in (-1, 1):
    ellipsoid("Ear", (side * 0.49, 0.03, 1.86), (0.17, 0.09, 0.22), sky_light, root)
    ellipsoid("Ear inner", (side * 0.50, -0.03, 1.86), (0.09, 0.035, 0.12), lavender, root)
    tube(
        "Curved horn",
        [
            (side * 0.25, 0.12, 2.00),
            (side * 0.37, 0.12, 2.25),
            (side * 0.28, 0.12, 2.43),
        ],
        [1.0, 0.8, 0.22],
        0.065,
        horn,
        root,
    )
    ellipsoid("Horn tip", (side * 0.28, 0.12, 2.43), (0.055, 0.055, 0.055), gold, root)
    tube(
        "Whisker",
        [
            (side * 0.18, -0.58, 1.43),
            (side * 0.43, -0.62, 1.37),
            (side * 0.59, -0.52, 1.48),
        ],
        [1.0, 0.65, 0.18],
        0.018,
        horn,
        root,
    )

# Small cloud curls add Chinese ornament without making the face busy.
for side in (-1, 1):
    tube(
        "Cloud curl",
        [
            (side * 0.26, -0.535, 1.91),
            (side * 0.35, -0.55, 1.98),
            (side * 0.42, -0.53, 1.92),
        ],
        [0.9, 0.7, 0.15],
        0.022,
        lavender,
        root,
    )

export_glb(root, "chinese-dragon.glb")
render_poster(
    "chinese-dragon.png",
    look_at=(0, 0, 1.16),
    camera=(1.75, -7.4, 2.75),
    ortho_scale=2.85,
)
