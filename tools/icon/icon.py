# Draws texxer's app icon: a crunchy pixel cube with a painted band on a dark tile.
# usage: tools/icon/build.sh
import math

N = 32
CELL = 21
SIZE = 1024
TILE = 824
OFF = (SIZE - N * CELL) // 2

bg, line = '#181816', '#2a2926'
faces = {'top': (0xcf, 0xc9, 0xbb), 'left': (0x8f, 0x8a, 0x7f), 'right': (0x5c, 0x58, 0x51)}
accent = {'left': (0x7f, 0xc4, 0xaf), 'right': (0x4e, 0x86, 0x76)}
wear = (0xec, 0xe7, 0xdb)

T, R, B, L = (16, 3.5), (29, 10), (16, 16.5), (3, 10)
D = 12.5


def inside(p, poly):
    x, y = p
    sign = 0
    for i in range(len(poly)):
        a, b = poly[i], poly[(i + 1) % len(poly)]
        c = (b[0] - a[0]) * (y - a[1]) - (b[1] - a[1]) * (x - a[0])
        if c != 0:
            if sign and (c > 0) != (sign > 0):
                return False
            sign = c
    return True


def hash01(x, y, seed):
    v = math.sin(x * 127.1 + y * 311.7 + seed * 74.7) * 43758.5453
    return v - math.floor(v)


polys = {
    'top': [T, R, B, L],
    'left': [L, B, (B[0], B[1] + D), (L[0], L[1] + D)],
    'right': [B, R, (R[0], R[1] + D), (B[0], B[1] + D)],
}


def color(face, x, y):
    cx, cy = x + 0.5, y + 0.5
    c = faces[face]
    # Depth below the face's top edge, for the band and the wear.
    if face == 'left':
        v = cy - (L[1] + (cx - L[0]) * 0.5)
    elif face == 'right':
        v = cy - (B[1] - (cx - B[0]) * 0.5)
    else:
        v = None

    if v is not None and 6.2 < v < 9.2:
        c = accent[face]
    n = hash01(x, y, 1)
    k = 1 + (0.09 if n > 0.8 else -0.08 if n < 0.18 else 0)
    c = tuple(min(255, int(ch * k)) for ch in c)

    # Worn rims along the top face's front edges, chipped here and there at the front corner.
    if face == 'top':
        rim = cy - (L[1] + (cx - L[0]) * 0.5) if cx < B[0] else cy - (B[1] - (cx - B[0]) * 0.5)
        if rim > -1.2 and hash01(x, y, 2) > 0.3:
            c = wear
    elif abs(cx - B[0]) < 1.0 and cy < B[1] + 4 and hash01(x, y, 3) > 0.4:
        c = wear
    return c


rects = []
for y in range(N):
    for x in range(N):
        p = (x + 0.5, y + 0.5)
        for face in ('top', 'left', 'right'):
            if inside(p, polys[face]):
                r, g, b = color(face, x, y)
                rects.append(f'<rect x="{OFF + x * CELL}" y="{OFF + y * CELL + 4}" width="{CELL}" height="{CELL}" fill="#{r:02x}{g:02x}{b:02x}"/>')
                break

pad = (SIZE - TILE) // 2
print(f'''<svg xmlns="http://www.w3.org/2000/svg" width="{SIZE}" height="{SIZE}" viewBox="0 0 {SIZE} {SIZE}" shape-rendering="crispEdges">
<rect x="{pad}" y="{pad}" width="{TILE}" height="{TILE}" rx="185" fill="{bg}" shape-rendering="auto"/>
<rect x="{pad + 3}" y="{pad + 3}" width="{TILE - 6}" height="{TILE - 6}" rx="182" fill="none" stroke="{line}" stroke-width="6" shape-rendering="auto"/>
{chr(10).join(rects)}
</svg>''')
