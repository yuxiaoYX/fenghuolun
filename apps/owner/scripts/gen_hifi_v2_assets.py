# Generate hifi-v2 tab/nav icons and a top-down vehicle PNG for uni-app x.
from pathlib import Path

from PIL import Image, ImageDraw, ImageFilter

ROOT = Path(__file__).resolve().parents[1]
TAB = ROOT / "static" / "tab"
NAV = ROOT / "static" / "nav"
VEH = ROOT / "static" / "vehicle"
ELE = (47, 224, 141, 255)
FAINT = (88, 112, 127, 255)
INK2 = (183, 201, 216, 255)
INK_ON_LIGHT = (61, 80, 100, 255)
BRAND = (255, 106, 61, 255)


def scale_pts(pts, s, ox, oy):
    return [(ox + x * s, oy + y * s) for x, y in pts]


ICON_MUL = 4
ICON_SIZE = 81


def icon_canvas():
    return Image.new("RGBA", (ICON_SIZE * ICON_MUL, ICON_SIZE * ICON_MUL), (0, 0, 0, 0))


def down(img: Image.Image) -> Image.Image:
    return img.resize((ICON_SIZE, ICON_SIZE), Image.Resampling.LANCZOS)


def draw_stroke(draw: ImageDraw.ImageDraw, color, width):
    def line(pts, closed=False):
        if closed and pts[0] != pts[-1]:
            pts = pts + [pts[0]]
        draw.line(pts, fill=color, width=width, joint="curve")

    return line


def save_pair(name, painter):
    for suffix, color in (("", FAINT), ("-active", ELE)):
        img = icon_canvas()
        d = ImageDraw.Draw(img)
        painter(d, color)
        down(img).save(TAB / f"{name}{suffix}.png")


def paint_now(d, color):
    s, ox, oy = 2.15 * ICON_MUL, 15 * ICON_MUL, 16 * ICON_MUL
    line = draw_stroke(d, color, 4 * ICON_MUL)
    line(scale_pts([(4, 11.5), (12, 5), (20, 11.5), (20, 20), (4, 20), (4, 11.5)], s, ox, oy))
    line(scale_pts([(9, 20), (9, 14), (15, 14), (15, 20)], s, ox, oy))


def paint_ctrl(d, color):
    s, ox, oy = 2.15 * ICON_MUL, 15 * ICON_MUL, 16 * ICON_MUL
    line = draw_stroke(d, color, 4 * ICON_MUL)
    for x, y in ((3, 3), (13, 3), (3, 13), (13, 13)):
        line(scale_pts([(x, y), (x + 8, y), (x + 8, y + 8), (x, y + 8)], s, ox, oy), closed=True)


def paint_energy(d, color):
    s, ox, oy = 2.15 * ICON_MUL, 15 * ICON_MUL, 16 * ICON_MUL
    line = draw_stroke(d, color, 4 * ICON_MUL)
    line(scale_pts([(13, 2), (5, 14), (11, 14), (10, 22), (18, 10), (12, 10), (13, 2)], s, ox, oy))


def paint_battery(d, color):
    s, ox, oy = 2.15 * ICON_MUL, 15 * ICON_MUL, 16 * ICON_MUL
    line = draw_stroke(d, color, 4 * ICON_MUL)
    line(scale_pts([(6, 7), (18, 7), (18, 21), (6, 21)], s, ox, oy), closed=True)
    line(scale_pts([(10, 7), (10, 4), (14, 4), (14, 7)], s, ox, oy))


def paint_more(d, color):
    s, ox, oy = 2.15 * ICON_MUL, 15 * ICON_MUL, 16 * ICON_MUL
    line = draw_stroke(d, color, 4 * ICON_MUL)
    cx, cy, r = ox + 12 * s, oy + 8 * s, 4 * s
    d.ellipse((cx - r, cy - r, cx + r, cy + r), outline=color, width=4 * ICON_MUL)
    line(scale_pts([(4, 20), (6, 16.5), (12, 15), (18, 16.5), (20, 20)], s, ox, oy))


def paint_sync(d, color):
    # hifi-v2 sync: two clockwise arcs + L arrowheads (24x24 viewBox).
    s, ox, oy = 2.15 * ICON_MUL, 15 * ICON_MUL, 16 * ICON_MUL
    w = 4 * ICON_MUL
    line = draw_stroke(d, color, w)
    bbox = (ox + 4 * s, oy + 4 * s, ox + 20 * s, oy + 20 * s)
    d.arc(bbox, start=180, end=321, fill=color, width=w)
    d.arc(bbox, start=0, end=141, fill=color, width=w)
    line(scale_pts([(18, 4), (18, 9), (13, 9)], s, ox, oy))
    line(scale_pts([(6, 20), (6, 15), (11, 15)], s, ox, oy))


def paint_info(d, color):
    s, ox, oy = 2.15 * ICON_MUL, 15 * ICON_MUL, 16 * ICON_MUL
    cx, cy, r = ox + 12 * s, oy + 12 * s, 9 * s
    d.ellipse((cx - r, cy - r, cx + r, cy + r), outline=color, width=4 * ICON_MUL)
    d.ellipse((cx - 2 * ICON_MUL, oy + 7 * s, cx + 2 * ICON_MUL, oy + 7 * s + 4 * ICON_MUL), fill=color)
    d.rectangle((cx - 2 * ICON_MUL, oy + 11 * s, cx + 2 * ICON_MUL, oy + 17 * s), fill=color)


def paint_lock(d, color):
    s, ox, oy = 2.15 * ICON_MUL, 15 * ICON_MUL, 16 * ICON_MUL
    line = draw_stroke(d, color, 4 * ICON_MUL)
    line(scale_pts([(5, 11), (19, 11), (19, 20), (5, 20)], s, ox, oy), closed=True)
    d.arc((ox + 8 * s, oy + 4 * s, ox + 16 * s, oy + 12 * s), start=180, end=0, fill=color, width=4 * ICON_MUL)


def paint_sun(d, color):
    s, ox, oy = 2.15 * ICON_MUL, 15 * ICON_MUL, 16 * ICON_MUL
    w = 4 * ICON_MUL
    cx, cy, r = ox + 12 * s, oy + 12 * s, 4.2 * s
    d.ellipse((cx - r, cy - r, cx + r, cy + r), outline=color, width=w)
    line = draw_stroke(d, color, w)
    for a, b in (
        ((12, 2), (12, 5)),
        ((12, 19), (12, 22)),
        ((2, 12), (5, 12)),
        ((19, 12), (22, 12)),
        ((5.2, 5.2), (7.2, 7.2)),
        ((16.8, 16.8), (18.8, 18.8)),
        ((18.8, 5.2), (16.8, 7.2)),
        ((7.2, 16.8), (5.2, 18.8)),
    ):
        line(scale_pts([a, b], s, ox, oy))


def paint_moon(d, color):
    s, ox, oy = 2.15 * ICON_MUL, 15 * ICON_MUL, 16 * ICON_MUL
    w = 4 * ICON_MUL
    d.arc((ox + 5 * s, oy + 4 * s, ox + 19 * s, oy + 20 * s), start=40, end=320, fill=color, width=w)


def paint_grid(d, color):
    s, ox, oy = 2.15, 15, 16
    line = draw_stroke(d, color, 4)
    line(scale_pts([(4, 4), (20, 4), (20, 20), (4, 20)], s, ox, oy), closed=True)
    line(scale_pts([(4, 12), (20, 12)], s, ox, oy))


def rounded_poly(draw, pts, fill, outline, width=3):
    draw.polygon(pts, fill=fill)
    if outline is not None:
        draw.line(pts + [pts[0]], fill=outline, width=width, joint="curve")


def vehicle_png():
    W, H = 560, 620
    scale = 2.0
    img = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    d = ImageDraw.Draw(img)

    def X(x):
        return (x - 90) * scale

    def Y(y):
        return (y - 16) * scale

    # shadow
    shadow = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    sd = ImageDraw.Draw(shadow)
    body = [
        (X(170), Y(28)),
        (X(145), Y(29)),
        (X(126), Y(34)),
        (X(116), Y(48)),
        (X(108), Y(68)),
        (X(106), Y(90)),
        (X(104), Y(200)),
        (X(105), Y(236)),
        (X(118), Y(258)),
        (X(140), Y(270)),
        (X(170), Y(274)),
        (X(200), Y(270)),
        (X(222), Y(258)),
        (X(235), Y(236)),
        (X(236), Y(200)),
        (X(234), Y(90)),
        (X(232), Y(68)),
        (X(224), Y(48)),
        (X(214), Y(34)),
        (X(195), Y(29)),
    ]
    body_s = [(x, y + 8) for x, y in body]
    sd.polygon(body_s, fill=(0, 0, 0, 36))
    shadow = shadow.filter(ImageFilter.GaussianBlur(6))
    img = Image.alpha_composite(img, shadow)
    d = ImageDraw.Draw(img)

    # wheels
    wheel = (75, 88, 101, 255)
    wheel_line = (120, 133, 146, 255)
    for x, y in ((98, 84), (226, 84), (98, 214), (226, 214)):
        d.rounded_rectangle((X(x), Y(y), X(x + 16), Y(y + 36)), radius=8, fill=wheel, outline=wheel_line, width=2)

    # body
    paint = (240, 244, 248, 255)
    paint_line = (158, 172, 185, 255)
    d.polygon(body, fill=paint)
    d.line(body + [body[0]], fill=paint_line, width=3, joint="curve")

    # front light bar
    d.arc((X(118), Y(36), X(222), Y(78)), start=200, end=340, fill=(255, 255, 255, 255), width=6)
    d.line([(X(120), Y(52)), (X(116), Y(74))], fill=(184, 198, 209, 255), width=4)
    d.line([(X(220), Y(52)), (X(224), Y(74))], fill=(184, 198, 209, 255), width=4)
    d.line([(X(112), Y(62)), (X(112), Y(76))], fill=(221, 230, 236, 255), width=5)
    d.line([(X(228), Y(62)), (X(228), Y(76))], fill=(221, 230, 236, 255), width=5)

    # flame mark
    fx, fy = X(170), Y(62)
    flame = [
        (fx, fy - 10),
        (fx + 8, fy + 2),
        (fx + 4, fy + 4),
        (fx + 10, fy + 16),
        (fx, fy + 8),
        (fx - 10, fy + 16),
        (fx - 4, fy + 4),
        (fx - 8, fy + 2),
    ]
    d.polygon(flame, fill=(101, 119, 133, 255))

    # windshield
    glass = (186, 201, 214, 255)
    glass_line = (150, 166, 180, 255)
    windshield = [
        (X(118), Y(100)),
        (X(170), Y(88)),
        (X(222), Y(100)),
        (X(210), Y(128)),
        (X(170), Y(120)),
        (X(130), Y(128)),
    ]
    d.polygon(windshield, fill=glass)
    d.line(windshield + [windshield[0]], fill=glass_line, width=2, joint="curve")

    # roof
    roof = [
        (X(128), Y(136)),
        (X(212), Y(136)),
        (X(212), Y(204)),
        (X(128), Y(204)),
    ]
    d.rounded_rectangle((X(126), Y(136), X(214), Y(206)), radius=16, fill=(216, 226, 234, 255), outline=(181, 195, 206, 255), width=2)
    d.line([(X(138), Y(148)), (X(138), Y(196))], fill=(249, 251, 252, 180), width=2)
    d.line([(X(202), Y(148)), (X(202), Y(196))], fill=(249, 251, 252, 180), width=2)

    # rear window
    rear = [
        (X(124), Y(214)),
        (X(170), Y(218)),
        (X(216), Y(214)),
        (X(224), Y(242)),
        (X(170), Y(254)),
        (X(116), Y(242)),
    ]
    d.polygon(rear, fill=glass)
    d.line(rear + [rear[0]], fill=glass_line, width=2, joint="curve")

    # tail lamp
    d.arc((X(130), Y(248), X(210), Y(268)), start=200, end=340, fill=(212, 136, 130, 255), width=4)

    # door seams
    seam = (170, 185, 199, 255)
    d.line([(X(112), Y(118)), (X(118), Y(168))], fill=seam, width=2)
    d.line([(X(228), Y(118)), (X(222), Y(168))], fill=seam, width=2)
    d.line([(X(112), Y(176)), (X(118), Y(220))], fill=seam, width=2)
    d.line([(X(228), Y(176)), (X(222), Y(220))], fill=seam, width=2)

    # mirrors
    d.polygon([(X(108), Y(94)), (X(96), Y(102)), (X(97), Y(110)), (X(110), Y(104))], fill=(117, 133, 145, 255), outline=(219, 228, 235, 255))
    d.polygon([(X(232), Y(94)), (X(244), Y(102)), (X(243), Y(110)), (X(230), Y(104))], fill=(117, 133, 145, 255), outline=(219, 228, 235, 255))

    bbox = img.getbbox()
    if bbox != None:
        pad = 12
        img = img.crop((max(0, bbox[0] - pad), max(0, bbox[1] - pad), min(W, bbox[2] + pad), min(H, bbox[3] + pad)))
    VEH.mkdir(parents=True, exist_ok=True)
    img.save(VEH / "neta-l.png")


def main():
    TAB.mkdir(parents=True, exist_ok=True)
    NAV.mkdir(parents=True, exist_ok=True)
    save_pair("now", paint_now)
    save_pair("ctrl", paint_ctrl)
    save_pair("energy", paint_energy)
    save_pair("battery", paint_battery)
    save_pair("more", paint_more)
    for name, painter, color in (
        ("sync", paint_sync, INK2),
        ("info", paint_info, INK2),
        ("lock", paint_lock, INK2),
        ("sun", paint_sun, INK2),
        ("moon", paint_moon, INK2),
        ("sync-on-light", paint_sync, INK_ON_LIGHT),
        ("info-on-light", paint_info, INK_ON_LIGHT),
        ("lock-on-light", paint_lock, INK_ON_LIGHT),
        ("sun-on-light", paint_sun, INK_ON_LIGHT),
        ("moon-on-light", paint_moon, INK_ON_LIGHT),
    ):
        img = icon_canvas()
        painter(ImageDraw.Draw(img), color)
        down(img).save(NAV / f"{name}.png")
    from render_vehicle import main as render_vehicle_main
    render_vehicle_main()
    print("wrote icons and vehicle art")


if __name__ == "__main__":
    main()
