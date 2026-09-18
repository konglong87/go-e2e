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

jade = make_material("Jade scales", (0.10, 0.58, 0.44), 0.15)
jade_dark = make_material("Deep jade", (0.05, 0.42, 0.32), 0.2)
cream = make_material("Cloud cream", (0.96, 0.93, 0.8))
crimson = make_material("Mane crimson", (0.82, 0.14, 0.12), 0.1)
crimson_light = make_material("Mane highlight", (0.96, 0.24, 0.20), 0.05, 0.4)
gold = make_material("Imperial gold", (0.98, 0.72, 0.18), 0.55, 0.25)
ink = make_material("Ink", (0.02, 0.03, 0.04), 0.2)
white = make_material("Porcelain", (0.95, 0.98, 0.98))
blush = make_material("Blush pink", (0.98, 0.45, 0.45), 0.0, 0.5)

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
tube("Body", BODY, BODY_RADII, 0.185, jade, root)
belly_points = [(x, y - 0.13, z - 0.02) for x, y, z in BODY[1:]]
tube("Belly", belly_points, [r * 0.8 for r in BODY_RADII[1:]], 0.13, cream, root)

ellipsoid("Head", (0.0, -0.10, 1.60), (0.44, 0.39, 0.39), jade, root)
ellipsoid("Snout", (0.0, -0.46, 1.50), (0.21, 0.14, 0.13), jade, root)
ellipsoid("Nose ridge", (0.0, -0.50, 1.60), (0.09, 0.07, 0.06), jade_dark, root)
ellipsoid("Chin", (0.0, -0.46, 1.40), (0.16, 0.12, 0.10), cream, root)
cone("Beard", (0.0, -0.38, 1.30), 0.05, 0.14, cream, rotation=(3.14159, 0, 0), parent=root)
ellipsoid("Head mane", (0.0, 0.20, 1.72), (0.37, 0.19, 0.37), crimson, root)
ellipsoid("Smile", (0.0, -0.585, 1.415), (0.058, 0.03, 0.045), ink, root)
ellipsoid("Tongue", (0.0, -0.60, 1.40), (0.036, 0.02, 0.026), blush, root)

for side in (-1, 1):
    ellipsoid("Eye", (side * 0.135, -0.40, 1.68), (0.12, 0.062, 0.13), ink, root)
    ellipsoid("Eye glint big", (side * 0.135 - 0.022, -0.458, 1.73), (0.034, 0.016, 0.043), white, root)
    ellipsoid("Eye glint small", (side * 0.135 + 0.030, -0.458, 1.635), (0.016, 0.009, 0.020), white, root)
    ellipsoid("Brow", (side * 0.16, -0.28, 1.88), (0.07, 0.04, 0.04), crimson, root)
    ellipsoid("Blush", (side * 0.25, -0.37, 1.53), (0.07, 0.035, 0.055), blush, root)
    ellipsoid("Cheek mane", (side * 0.30, 0.06, 1.58), (0.14, 0.12, 0.21), crimson, root)
    ellipsoid("Nostril", (side * 0.05, -0.565, 1.55), (0.016, 0.012, 0.016), ink, root)
    cone("Horn", (side * 0.14, 0.06, 1.95), 0.065, 0.30, gold, rotation=(-0.35, side * 0.15, 0), parent=root)
    ellipsoid("Horn tip", (side * 0.161, 0.111, 2.089), (0.05, 0.05, 0.05), gold, root)
    cone("Horn branch low", (side * 0.18, 0.11, 2.00), 0.04, 0.14, gold, rotation=(-0.9, side * 0.55, 0), parent=root)
    cone("Horn branch high", (side * 0.17, 0.13, 2.10), 0.032, 0.12, gold, rotation=(-1.1, side * 0.45, 0), parent=root)
    ellipsoid("Ear fin", (side * 0.36, 0.0, 1.66), (0.10, 0.045, 0.08), crimson, root)
    tube(
        "Whisker",
        [(side * 0.16, -0.48, 1.54), (side * 0.38, -0.52, 1.58), (side * 0.56, -0.44, 1.70)],
        [1.0, 0.7, 0.25], 0.018, gold, root,
    )

for name, position, scale, material in [
    ("Mane puff crown", (0.0, 0.22, 1.98), (0.16, 0.10, 0.13), crimson_light),
    ("Mane puff left", (-0.34, 0.15, 1.84), (0.13, 0.10, 0.15), crimson_light),
    ("Mane puff right", (0.34, 0.15, 1.84), (0.13, 0.10, 0.15), crimson_light),
    ("Mane puff back left", (-0.40, 0.18, 1.58), (0.12, 0.11, 0.17), crimson),
    ("Mane puff back right", (0.40, 0.18, 1.58), (0.12, 0.11, 0.17), crimson),
    ("Mane puff neck", (0.0, 0.18, 1.34), (0.18, 0.11, 0.12), crimson_light),
]:
    ellipsoid(name, position, scale, material, root)

for i, (x, y, z) in enumerate(BODY[2:7]):
    ellipsoid("Spine mane", (x, y + 0.16, z + 0.12), (0.07, 0.05, 0.12 - i * 0.008), crimson, root)

for lx, ly, lz in [(0.14, -0.20, 1.28), (-0.40, -0.18, 1.12), (0.58, -0.10, 0.48), (-0.52, -0.02, 0.30)]:
    ellipsoid("Leg", (lx, ly, lz), (0.10, 0.09, 0.11), jade, root)
    ellipsoid("Paw", (lx, ly - 0.06, lz - 0.10), (0.12, 0.12, 0.09), jade, root)
    for toe in (-1, 0, 1):
        cone("Claw", (lx + toe * 0.055, ly - 0.17, lz - 0.11), 0.018, 0.06, gold, rotation=(1.9, 0, 0), parent=root)

for cx, cy, cz, s in [(-0.70, 0.10, 0.16, 0.10), (-0.60, 0.10, 0.10, 0.08), (-0.76, 0.10, 0.24, 0.07)]:
    ellipsoid("Cloud", (cx, cy, cz), (s, s * 0.8, s), cream, root)
for position, scale in [((-0.82, 0.10, 0.22), (0.12, 0.10, 0.12)), ((-0.88, 0.10, 0.34), (0.09, 0.08, 0.13))]:
    ellipsoid("Tail fluff", position, scale, crimson_light, root)

export_glb(root, "chinese-dragon.glb")
render_poster("chinese-dragon.png", look_at=(0, 0, 1.05), camera=(1.6, -6.8, 2.4), ortho_scale=2.6)
