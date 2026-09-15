# Rasterize the hifi-v2 vehicle body (official Neta flame logo) for uni-app x.
# Layout in the app keeps the 248px map; this PNG is the car only, correct aspect.
from __future__ import annotations

import math
import re
from pathlib import Path

from PIL import Image, ImageDraw, ImageFilter

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "static" / "vehicle" / "neta-l.png"

# vehicleBody local coords from hifi-v2.html (before translate/scale).
BODY = (
    "M170 24 C145 24 126 28 116 42 Q106 57 106 84 L104 244 "
    "Q104 266 118 286 Q130 299 170 304 Q210 299 222 286 "
    "Q236 266 236 244 L234 84 Q234 57 224 42 C214 28 195 24 170 24Z"
)
WINDSHIELD = "M113 103 Q170 84 227 103 L216 138 Q170 128 124 138Z"
REAR_GLASS = "M122 230 Q170 235 218 230 L228 261 Q170 278 112 261Z"
HATCH = "M126 279 Q170 289 214 279 L210 287 Q170 301 130 287Z"
MIRROR_L = "M108 94 Q99 92 94 102 L95 112 Q104 110 111 104Z"
MIRROR_R = "M232 94 Q241 92 246 102 L245 112 Q236 110 229 104Z"
FLAME = [
    "M24.86,13.93a26,26,0,0,0-2,3.55,1,1,0,0,1-1.9,0,27.17,27.17,0,0,0-2-3.55A26.81,26.81,0,0,0,1.72,2.29a.88.88,0,0,0-.92.42L.12,3.9a.88.88,0,0,0,.6,1.3,23.77,23.77,0,0,1,16.8,12.11,23,23,0,0,1,2,4.87c.57,2,1.72,13,1.72,13a.76.76,0,0,0,1.49,0s1.15-11,1.72-13a23.51,23.51,0,0,1,2-4.87A23.77,23.77,0,0,1,43.17,5.2a.87.87,0,0,0,.6-1.3l-.68-1.19a.87.87,0,0,0-.92-.42A26.76,26.76,0,0,0,24.86,13.93Z",
    "M20.32,7.37A32,32,0,0,0,10.78.09.91.91,0,0,0,9.62.43L8.85,1.78A.87.87,0,0,0,9.22,3,28.69,28.69,0,0,1,20,12.24c.43.61.84,1.24,1.22,1.89a.87.87,0,0,0,1.5,0c.38-.65.78-1.28,1.21-1.89A28.64,28.64,0,0,1,34.68,3a.88.88,0,0,0,.37-1.22L34.27.43A.9.9,0,0,0,33.12.09a32,32,0,0,0-9.55,7.28,2.18,2.18,0,0,1-3.25,0Z",
    "M16.31,28.55h1.43a.87.87,0,0,0,.87-.91,21.86,21.86,0,0,0-1.87-8,22.13,22.13,0,0,0-10.9-11A.91.91,0,0,0,4.71,9L4,10.18a.87.87,0,0,0,.38,1.23,18.84,18.84,0,0,1,11.06,16.3.89.89,0,0,0,.87.84Z",
    "M27.15,19.64a22.06,22.06,0,0,0-1.87,8,.87.87,0,0,0,.88.91h1.42a.89.89,0,0,0,.87-.84,18.85,18.85,0,0,1,11.06-16.3.87.87,0,0,0,.39-1.23L39.18,9a.91.91,0,0,0-1.13-.36,22.21,22.21,0,0,0-10.9,11.05Z",
]

TOKEN_RE = re.compile(r"[MmZzLlHhVvCcSsQqTtAa]|[-+]?(?:\d*\.\d+|\d+)(?:[eE][-+]?\d+)?")


def tokens(d: str):
    return TOKEN_RE.findall(d.replace(",", " "))


def is_cmd(t: str) -> bool:
    return len(t) == 1 and t.isalpha()


def nums(ts, i, n):
    out = []
    k = 0
    while k < n:
        out.append(float(ts[i + k]))
        k += 1
    return out, i + n


def lerp(a, b, t):
    return a + (b - a) * t


def cubic_pts(p0, p1, p2, p3, n=20):
    pts = []
    i = 0
    while i <= n:
        t = i / n
        u = 1 - t
        x = u * u * u * p0[0] + 3 * u * u * t * p1[0] + 3 * u * t * t * p2[0] + t * t * t * p3[0]
        y = u * u * u * p0[1] + 3 * u * u * t * p1[1] + 3 * u * t * t * p2[1] + t * t * t * p3[1]
        pts.append((x, y))
        i += 1
    return pts


def quad_pts(p0, p1, p2, n=16):
    pts = []
    i = 0
    while i <= n:
        t = i / n
        u = 1 - t
        x = u * u * p0[0] + 2 * u * t * p1[0] + t * t * p2[0]
        y = u * u * p0[1] + 2 * u * t * p1[1] + t * t * p2[1]
        pts.append((x, y))
        i += 1
    return pts


def arc_pts(x1, y1, rx, ry, phi_deg, large, sweep, x2, y2, n=24):
    rx = abs(rx)
    ry = abs(ry)
    if rx < 1e-6 or ry < 1e-6:
        return [(x1, y1), (x2, y2)]
    phi = math.radians(phi_deg)
    cos_p = math.cos(phi)
    sin_p = math.sin(phi)
    dx = (x1 - x2) / 2
    dy = (y1 - y2) / 2
    x1p = cos_p * dx + sin_p * dy
    y1p = -sin_p * dx + cos_p * dy
    lam = (x1p * x1p) / (rx * rx) + (y1p * y1p) / (ry * ry)
    if lam > 1:
        s = math.sqrt(lam)
        rx *= s
        ry *= s
    num = rx * rx * ry * ry - rx * rx * y1p * y1p - ry * ry * x1p * x1p
    den = rx * rx * y1p * y1p + ry * ry * x1p * x1p
    if den <= 0:
        return [(x1, y1), (x2, y2)]
    coef = math.sqrt(max(0, num / den))
    if large == sweep:
        coef = -coef
    cxp = coef * rx * y1p / ry
    cyp = coef * -ry * x1p / rx
    cx = cos_p * cxp - sin_p * cyp + (x1 + x2) / 2
    cy = sin_p * cxp + cos_p * cyp + (y1 + y2) / 2

    def angle(ux, uy, vx, vy):
        dot = ux * vx + uy * vy
        nrm = math.hypot(ux, uy) * math.hypot(vx, vy)
        if nrm == 0:
            return 0
        c = max(-1, min(1, dot / nrm))
        ang = math.acos(c)
        if ux * vy - uy * vx < 0:
            ang = -ang
        return ang

    ux = (x1p - cxp) / rx
    uy = (y1p - cyp) / ry
    vx = (-x1p - cxp) / rx
    vy = (-y1p - cyp) / ry
    t1 = angle(1, 0, ux, uy)
    dt = angle(ux, uy, vx, vy)
    if sweep == 0 and dt > 0:
        dt -= 2 * math.pi
    if sweep == 1 and dt < 0:
        dt += 2 * math.pi
    pts = []
    i = 0
    while i <= n:
        t = t1 + dt * i / n
        x = cos_p * rx * math.cos(t) - sin_p * ry * math.sin(t) + cx
        y = sin_p * rx * math.cos(t) + cos_p * ry * math.sin(t) + cy
        pts.append((x, y))
        i += 1
    return pts


def path_to_rings(d: str):
    ts = tokens(d)
    i = 0
    rings = []
    ring = []
    x = 0.0
    y = 0.0
    sx = 0.0
    sy = 0.0
    last_c = ""
    c2 = (0.0, 0.0)
    q1 = (0.0, 0.0)
    while i < len(ts):
        if is_cmd(ts[i]):
            cmd = ts[i]
            i += 1
        else:
            if last_c == "":
                break
            cmd = last_c
            if cmd == "M":
                cmd = "L"
            elif cmd == "m":
                cmd = "l"
        rel = cmd.islower()
        c = cmd.upper()
        executed = cmd
        if c == "Z":
            if len(ring) > 2:
                rings.append(ring)
            ring = []
            x = sx
            y = sy
            last_c = executed
            continue
        if c == "M":
            if len(ring) > 2:
                rings.append(ring)
            ring = []
            vals, i = nums(ts, i, 2)
            if rel:
                x += vals[0]
                y += vals[1]
            else:
                x = vals[0]
                y = vals[1]
            sx, sy = x, y
            ring.append((x, y))
            last_c = executed
            continue
        if c == "L":
            vals, i = nums(ts, i, 2)
            if rel:
                x += vals[0]
                y += vals[1]
            else:
                x = vals[0]
                y = vals[1]
            ring.append((x, y))
        elif c == "H":
            vals, i = nums(ts, i, 1)
            if rel:
                x += vals[0]
            else:
                x = vals[0]
            ring.append((x, y))
        elif c == "V":
            vals, i = nums(ts, i, 1)
            if rel:
                y += vals[0]
            else:
                y = vals[0]
            ring.append((x, y))
        elif c == "C":
            vals, i = nums(ts, i, 6)
            p0 = (x, y)
            p1 = (x + vals[0], y + vals[1]) if rel else (vals[0], vals[1])
            p2 = (x + vals[2], y + vals[3]) if rel else (vals[2], vals[3])
            p3 = (x + vals[4], y + vals[5]) if rel else (vals[4], vals[5])
            ring.extend(cubic_pts(p0, p1, p2, p3)[1:])
            c2 = p2
            x, y = p3
        elif c == "S":
            vals, i = nums(ts, i, 4)
            p0 = (x, y)
            if last_c.upper() in ("C", "S"):
                p1 = (2 * x - c2[0], 2 * y - c2[1])
            else:
                p1 = p0
            p2 = (x + vals[0], y + vals[1]) if rel else (vals[0], vals[1])
            p3 = (x + vals[2], y + vals[3]) if rel else (vals[2], vals[3])
            ring.extend(cubic_pts(p0, p1, p2, p3)[1:])
            c2 = p2
            x, y = p3
        elif c == "Q":
            vals, i = nums(ts, i, 4)
            p0 = (x, y)
            p1 = (x + vals[0], y + vals[1]) if rel else (vals[0], vals[1])
            p2 = (x + vals[2], y + vals[3]) if rel else (vals[2], vals[3])
            ring.extend(quad_pts(p0, p1, p2)[1:])
            q1 = p1
            x, y = p2
        elif c == "A":
            vals, i = nums(ts, i, 7)
            rx, ry, phi, large, sweep, x2, y2 = vals
            if rel:
                x2 += x
                y2 += y
            ring.extend(arc_pts(x, y, rx, ry, phi, int(large), int(sweep), x2, y2)[1:])
            x, y = x2, y2
        last_c = executed
    if len(ring) > 2:
        rings.append(ring)
    return rings


def xf(pts, scale, ox, oy):
    return [(ox + p[0] * scale, oy + p[1] * scale) for p in pts]


def fill_path(draw, d, scale, ox, oy, fill, outline=None, width=2):
    for ring in path_to_rings(d):
        pts = xf(ring, scale, ox, oy)
        draw.polygon(pts, fill=fill)
        if outline is not None:
            draw.line(pts + [pts[0]], fill=outline, width=width, joint="curve")


def main():
    # Display target ~118x224 inside 248px map. Render 4x for sharpness.
    scale = 4.0
    pad = 16
    # body+mirrors roughly x 94..246, y 24..304
    x0, y0, x1, y1 = 90, 18, 250, 312
    w = int((x1 - x0) * scale + pad * 2)
    h = int((y1 - y0) * scale + pad * 2)
    ox = pad - x0 * scale
    oy = pad - y0 * scale

    img = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    shadow = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    sd = ImageDraw.Draw(shadow)
    fill_path(sd, BODY, scale, ox, oy + 6, (0, 0, 0, 40))
    img = Image.alpha_composite(img, shadow.filter(ImageFilter.GaussianBlur(5)))
    d = ImageDraw.Draw(img)

    wheel = (75, 88, 101, 255)
    wheel_line = (120, 133, 146, 255)
    for x, y in ((98, 81), (226, 81), (98, 230), (226, 230)):
        d.rounded_rectangle(
            (ox + x * scale, oy + y * scale, ox + (x + 16) * scale, oy + (y + 37) * scale),
            radius=int(5 * scale),
            fill=wheel,
            outline=wheel_line,
            width=max(2, int(scale)),
        )

    paint = (240, 243, 246, 255)
    paint_line = (158, 172, 185, 255)
    fill_path(d, BODY, scale, ox, oy, paint, paint_line, max(3, int(1.5 * scale)))

    # front light bar
    d.arc(
        (ox + 118 * scale, oy + 30 * scale, ox + 222 * scale, oy + 70 * scale),
        start=200,
        end=340,
        fill=(255, 255, 255, 255),
        width=max(4, int(3.5 * scale)),
    )
    d.line([(ox + 120 * scale, oy + 48 * scale), (ox + 116 * scale, oy + 76 * scale)], fill=(184, 198, 209, 255), width=max(3, int(2.2 * scale)))
    d.line([(ox + 220 * scale, oy + 48 * scale), (ox + 224 * scale, oy + 76 * scale)], fill=(184, 198, 209, 255), width=max(3, int(2.2 * scale)))
    d.line([(ox + 112 * scale, oy + 62 * scale), (ox + 112 * scale, oy + 78 * scale)], fill=(221, 230, 236, 255), width=max(3, int(2.5 * scale)))
    d.line([(ox + 228 * scale, oy + 62 * scale), (ox + 228 * scale, oy + 78 * scale)], fill=(221, 230, 236, 255), width=max(3, int(2.5 * scale)))

    # official Neta flame, placed like hifi-v2: svg x=161 y=53 w=18 h=15 viewBox 0 0 43.89 35.8
    flame_scale = 18 / 43.89 * scale
    fx = ox + 161 * scale
    fy = oy + 53 * scale
    for dpath in FLAME:
        fill_path(d, dpath, flame_scale, fx, fy, (101, 119, 133, 255))

    glass = (186, 201, 214, 255)
    glass_line = (150, 166, 180, 255)
    fill_path(d, WINDSHIELD, scale, ox, oy, glass, glass_line, max(2, int(scale)))
    d.line([(ox + 116 * scale, oy + 109 * scale), (ox + 123 * scale, oy + 132 * scale)], fill=(230, 237, 243, 255), width=max(2, int(scale)))
    d.line([(ox + 224 * scale, oy + 109 * scale), (ox + 217 * scale, oy + 132 * scale)], fill=(230, 237, 243, 255), width=max(2, int(scale)))

    roof = (216, 226, 234, 255)
    roof_line = (181, 195, 206, 255)
    d.rounded_rectangle(
        (ox + 126 * scale, oy + 143 * scale, ox + 214 * scale, oy + 221 * scale),
        radius=int(10 * scale),
        fill=roof,
        outline=roof_line,
        width=max(2, int(scale)),
    )
    d.line([(ox + 135 * scale, oy + 153 * scale), (ox + 135 * scale, oy + 208 * scale)], fill=(249, 251, 252, 180), width=max(2, int(scale)))
    d.line([(ox + 205 * scale, oy + 153 * scale), (ox + 205 * scale, oy + 208 * scale)], fill=(249, 251, 252, 180), width=max(2, int(scale)))

    fill_path(d, REAR_GLASS, scale, ox, oy, glass, (172, 186, 198, 255), max(2, int(scale)))
    fill_path(d, MIRROR_L, scale, ox, oy, (117, 133, 145, 255), (219, 228, 235, 255), max(2, int(0.6 * scale)))
    fill_path(d, MIRROR_R, scale, ox, oy, (117, 133, 145, 255), (219, 228, 235, 255), max(2, int(0.6 * scale)))

    # closed doors / hatch as in the design
    door = (232, 236, 240, 255)
    door_line = (154, 166, 177, 255)
    for pts in (
        [(110, 124), (119, 133), (120, 178), (109, 178)],
        [(230, 124), (221, 133), (220, 178), (231, 178)],
        [(109, 182), (120, 182), (120, 218), (109, 228)],
        [(231, 182), (220, 182), (220, 218), (231, 228)],
    ):
        d.polygon(xf(pts, scale, ox, oy), fill=door, outline=door_line)
    fill_path(d, HATCH, scale, ox, oy, door, door_line, max(2, int(scale)))
    d.arc(
        (ox + 130 * scale, oy + 248 * scale, ox + 210 * scale, oy + 268 * scale),
        start=200,
        end=340,
        fill=(212, 136, 130, 255),
        width=max(3, int(1.7 * scale)),
    )

    bbox = img.getbbox()
    if bbox is not None:
        img = img.crop((max(0, bbox[0] - 8), max(0, bbox[1] - 8), min(w, bbox[2] + 8), min(h, bbox[3] + 8)))
    OUT.parent.mkdir(parents=True, exist_ok=True)
    img.save(OUT)
    print("wrote", OUT, img.size)


if __name__ == "__main__":
    main()
