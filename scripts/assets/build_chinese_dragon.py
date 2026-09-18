"""Rebuild the Chinese dragon pet using Blender in background mode.

Original stylized loong: serpentine coiled body, antlers, mane, whiskers,
four legs, and a cloud tail. No external models or textures are used.
"""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

import bpy  # noqa: E402
from pet_pipeline import (  # noqa: E402
    cone, ellipsoid, export_glb, make_material, render_poster, reset_scene, tube,
)

reset_scene()

jade = make_material("Jade scales", (0.06, 0.52, 0.38), 0.15)
jade_dark = make_material("Deep jade", (0.03, 0.36, 0.27), 0.2)
cream = make_material("Cloud cream", (0.96, 0.93, 0.8))
crimson = make_material("Mane crimson", (0.82, 0.14, 0.12), 0.1)
gold = make_material("Imperial gold", (0.98, 0.72, 0.18), 0.55, 0.25)
ink = make_material("Ink", (0.02, 0.03, 0.04), 0.2)
white = make_material("Porcelain", (0.95, 0.98, 0.98))

root = bpy.data.objects.new("ChineseDragon", None)
bpy.context.collection.objects.link(root)

BODY = [
    (-0.62, 0.10, 0.20),
    (0.12, 0.12, 0.28),
    (0.62, 0.10, 0.55),
    (0.48, 0.05, 0.95),
    (-0.08, 0.00, 1.00),
    (-0.36, 0.00, 1.26),
    (-0.10, -0.02, 1.44),
    (0.00, -0.05, 1.52),
]
BODY_RADII = [0.22, 0.62, 0.9, 1.0, 1.05, 1.0, 0.95, 0.9]
tube("Body", BODY, BODY_RADII, 0.17, jade, root)
belly_points = [(x, y - 0.13, z - 0.02) for x, y, z in BODY[1:]]
tube("Belly", belly_points, [r * 0.8 for r in BODY_RADII[1:]], 0.13, cream, root)

ellipsoid("Head", (0.0, -0.10, 1.62), (0.34, 0.30, 0.30), jade, root)
ellipsoid("Snout", (0.0, -0.40, 1.52), (0.20, 0.17, 0.14), jade, root)
ellipsoid("Nose ridge", (0.0, -0.45, 1.62), (0.08, 0.08, 0.06), jade_dark, root)
ellipsoid("Chin", (0.0, -0.40, 1.42), (0.15, 0.13, 0.09), cream, root)
cone("Beard", (0.0, -0.38, 1.30), 0.05, 0.14, cream, rotation=(3.14159, 0, 0), parent=root)
ellipsoid("Head mane", (0.0, 0.18, 1.70), (0.30, 0.16, 0.30), crimson, root)

for side in (-1, 1):
    ellipsoid("Eye", (side * 0.15, -0.355, 1.70), (0.07, 0.05, 0.085), ink, root)
    ellipsoid("Eye glint", (side * 0.15 - 0.015, -0.40, 1.73), (0.018, 0.013, 0.024), white, root)
    ellipsoid("Brow", (side * 0.18, -0.28, 1.86), (0.11, 0.05, 0.05), crimson, root)
    ellipsoid("Cheek mane", (side * 0.30, 0.06, 1.58), (0.10, 0.10, 0.17), crimson, root)
    ellipsoid("Nostril", (side * 0.055, -0.545, 1.56), (0.02, 0.015, 0.02), ink, root)
    cone("Horn", (side * 0.13, 0.06, 2.02), 0.05, 0.42, gold, rotation=(-0.35, side * 0.15, 0), parent=root)
    cone("Horn branch low", (side * 0.17, 0.11, 2.06), 0.032, 0.2, gold, rotation=(-0.9, side * 0.55, 0), parent=root)
    cone("Horn branch high", (side * 0.16, 0.13, 2.18), 0.026, 0.16, gold, rotation=(-1.1, side * 0.45, 0), parent=root)
    cone("Ear fin", (side * 0.31, -0.02, 1.68), 0.06, 0.16, crimson, rotation=(0, side * 1.35, 0), parent=root)
    tube(
        "Whisker",
        [(side * 0.16, -0.48, 1.54), (side * 0.38, -0.52, 1.58), (side * 0.56, -0.44, 1.70)],
        [1.0, 0.7, 0.25], 0.016, gold, root,
    )

for i, (x, y, z) in enumerate(BODY[2:7]):
    ellipsoid("Spine mane", (x, y + 0.16, z + 0.12), (0.07, 0.05, 0.12 - i * 0.008), crimson, root)

for lx, ly, lz in [(0.12, -0.20, 1.30), (-0.40, -0.18, 1.12), (0.58, -0.10, 0.48), (-0.52, -0.02, 0.30)]:
    ellipsoid("Leg", (lx, ly, lz), (0.09, 0.08, 0.14), jade, root)
    ellipsoid("Paw", (lx, ly - 0.06, lz - 0.13), (0.10, 0.11, 0.07), jade, root)
    for toe in (-1, 0, 1):
        cone("Claw", (lx + toe * 0.05, ly - 0.16, lz - 0.15), 0.018, 0.06, gold, rotation=(1.9, 0, 0), parent=root)

for cx, cy, cz, s in [(-0.70, 0.10, 0.16, 0.10), (-0.60, 0.10, 0.10, 0.08), (-0.76, 0.10, 0.24, 0.07)]:
    ellipsoid("Cloud", (cx, cy, cz), (s, s * 0.8, s), cream, root)

export_glb(root, "chinese-dragon.glb")
render_poster("chinese-dragon.png", look_at=(0, 0, 1.05), camera=(1.6, -6.8, 2.4), ortho_scale=2.6)
