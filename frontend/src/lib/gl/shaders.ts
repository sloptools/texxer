export const quadVertex = /* glsl */ `
void main() {
  gl_Position = vec4(position.xy, 0.0, 1.0);
}
`

export const maxMasks = 4
export const maxStamps = 96

export const kindIndex = { surfaces: 0, ao: 1, edges: 2, cavity: 3, gradient: 4, facing: 5, noise: 6, paint: 7, stamps: 8, grunge: 9, volume: 11, drips: 12 } as const
export const shapeIndex = { box: 0, cylinder: 1, sphere: 2 } as const

// Island ids a mask can select; smaller ones are too small to matter.
export const maxIslands = 4096

// Folders that can have a zone.
export const maxGroups = 64

// Rows of the parts and islands lookups: one per layer mask, then one per folder.
export const maskRows = 256
export const lookupRows = maskRows + maxGroups
export const blendIndex = { normal: 0, multiply: 1, add: 2, overlay: 3, relief: 4 } as const

// Light from direction l on normal n: 1 facing it, 0 away. Wrap lights half the
// back too; uBoth adds the light mirrored across x, for models seen from both sides.
const shade = /* glsl */ `
float shade(vec3 n, vec3 l, float power, bool wrap) {
  float d = dot(n, l);
  if (uBoth == 1) d = max(d, dot(n, vec3(-l.x, l.yz)));
  if (wrap) d = d * 0.5 + 0.5;
  return pow(clamp(d, 0.0, 1.0), power);
}

vec3 screen(vec3 a, vec3 b) {
  return 1.0 - (1.0 - a) * (1.0 - b);
}
`

const noise = /* glsl */ `
float hash(vec3 p) {
  p = fract(p * 0.3183099 + 0.1);
  p *= 17.0;
  return fract(p.x * p.y * p.z * (p.x + p.y + p.z));
}

// Wraps every period cells along the axes where it isn't 0.
float hashCell(vec3 i, vec3 period) {
  return hash(mix(i, mod(i, max(period, 1.0)), step(0.5, period)));
}

float vnoise(vec3 x, vec3 period) {
  vec3 i = floor(x);
  vec3 f = fract(x);
  f = f * f * (3.0 - 2.0 * f);
  return mix(
    mix(mix(hashCell(i, period), hashCell(i + vec3(1, 0, 0), period), f.x), mix(hashCell(i + vec3(0, 1, 0), period), hashCell(i + vec3(1, 1, 0), period), f.x), f.y),
    mix(mix(hashCell(i + vec3(0, 0, 1), period), hashCell(i + vec3(1, 0, 1), period), f.x), mix(hashCell(i + vec3(0, 1, 1), period), hashCell(i + vec3(1, 1, 1), period), f.x), f.y),
    f.z);
}

// Four octaves, spread back out to roughly 0..1. Wrapping ones double exactly.
float fbm(vec3 p, vec3 period) {
  float s = 0.0;
  float a = 0.5;
  float k = period == vec3(0.0) ? 2.03 : 2.0;
  for (int i = 0; i < 4; i++) {
    s += a * vnoise(p, period);
    p = p * k + 11.7;
    period *= 2.0;
    a *= 0.5;
  }
  return clamp((s / 0.9375 - 0.5) * 2.2 + 0.5, 0.0, 1.0);
}

float fbm(vec3 p) {
  return fbm(p, vec3(0.0));
}
`

// One layer: mixes its color over the previous result where all its masks agree.
export const layerFragment = /* glsl */ `
precision highp float;
precision highp int;

out vec4 outColor;

uniform sampler2D uPrev;
uniform sampler2D uGeo;
uniform sampler2D uPos;
uniform sampler2D uNrm;
uniform sampler2D uParts;
uniform sampler2D uIslands;
uniform int uZone;
uniform vec3 uExtent;
// A tile's size in bake positions, 0 along axes it doesn't repeat (all for models).
uniform vec3 uPeriod;
uniform vec2 uWidths;
uniform float uFactor;
uniform vec3 uColor;
uniform float uTone;
uniform float uToneScale;
uniform float uGrain;
uniform float uShine;
// Hue shift in turns, saturation factor, lightness offset.
uniform vec3 uAdjust;
uniform float uSeed;
uniform int uStampColors;
// The mask whose paint holds colors (coverage in alpha), or -1.
uniform int uPaintColor;
uniform sampler2D uSource;
uniform int uFromSource;
uniform int uEmitPass;
uniform int uGlow;
uniform vec3 uL;
uniform vec3 uL2;
uniform int uBoth;
uniform float uOpacity;
uniform int uBlend;
uniform int uMaskOnly;
uniform int uCount;
uniform int uKind[${maxMasks}];
uniform vec4 uA[${maxMasks}];
uniform vec4 uB[${maxMasks}];
// Stamps of the layer: first index and count per mask; volumes: shape and mirror.
uniform vec4 uC[${maxMasks}];
// Volumes: from model positions into the volume's own space.
uniform mat4 uV[${maxMasks}];
uniform float uMirrorX;
uniform sampler2D uPaint0;
uniform sampler2D uPaint1;
uniform sampler2D uPaint2;
uniform sampler2D uPaint3;
uniform sampler2D uGrunge0;
uniform sampler2D uGrunge1;
uniform sampler2D uGrunge2;
uniform sampler2D uGrunge3;
uniform sampler2D uAtlas;
uniform float uAtlasGrid;
uniform float uAtlasInset;
// Per stamp: center and half width; right axis and half height; normal and atlas cell.
uniform vec4 uSC[${maxStamps}];
uniform vec4 uST[${maxStamps}];
uniform vec4 uSN[${maxStamps}];

${noise}
${shade}

// On a tile, scales snap to whole repeats per tile, so patterns run on across its seams.
vec3 snapScale(vec3 s) {
  vec3 on = step(1e-9, uPeriod);
  return mix(s, max(floor(uPeriod * s + 0.5), 1.0) / max(uPeriod, 1e-9), on);
}

float noiseAt(vec3 p, vec3 s, float offset) {
  vec3 snapped = snapScale(s);
  return fbm(p * snapped + offset, floor(uPeriod * snapped + 0.5) * step(1e-9, uPeriod));
}

// The nearest repeat of d on a tile.
vec3 wrapped(vec3 d) {
  return d - uPeriod * floor(d / max(uPeriod, 1e-9) + 0.5);
}

// Sampler arrays can't take a loop index in GLSL ES 3.0.
vec4 paintTexel(int i, ivec2 t) {
  if (i == 0) return texelFetch(uPaint0, t, 0);
  if (i == 1) return texelFetch(uPaint1, t, 0);
  if (i == 2) return texelFetch(uPaint2, t, 0);
  return texelFetch(uPaint3, t, 0);
}

// Coverage: alpha for painted colors, red otherwise.
float paintAt(int i, ivec2 t) {
  vec4 p = paintTexel(i, t);
  return i == uPaintColor ? p.a : p.r;
}

float grungeAt(int i, vec2 uv) {
  if (i == 0) return texture(uGrunge0, uv).r;
  if (i == 1) return texture(uGrunge1, uv).r;
  if (i == 2) return texture(uGrunge2, uv).r;
  return texture(uGrunge3, uv).r;
}

// Projected from the three axes and blended by the normal, so there are no UV seams.
// Image up is world up on the sides, so streaks run down.
float grunge(int i, vec3 p, vec3 n) {
  vec3 w = pow(abs(n), vec3(4.0));
  w /= w.x + w.y + w.z;
  vec2 o = vec2(uB[i].y * 0.137, uB[i].y * 0.291);
  vec3 q = p * snapScale(vec3(uA[i].w));
  return grungeAt(i, vec2(q.z, -q.y) + o) * w.x
    + grungeAt(i, q.xz + o) * w.y
    + grungeAt(i, vec2(q.x, -q.y) + o) * w.z;
}

// The color under the strongest stamp, for layers that take their color from stamps.
vec3 gStampRGB = vec3(0.0);
float gStampA = 0.0;
// Light and shade of stamps with depth, added to the layer's color.
float gDepth = 0.0;

float cellAlpha(vec2 origin, float u, float w) {
  if (abs(u) > 1.0 || abs(w) > 1.0) return 0.0;
  vec2 local = vec2(u * 0.5 + 0.5, 0.5 - w * 0.5) * (1.0 - 2.0 * uAtlasInset) + uAtlasInset;
  return texture(uAtlas, (origin + local) / uAtlasGrid).a;
}

// Walls of a stamp sunk into the surface, or standing out of it, lit by the baked
// key light (l, in the stamp's u and w): the wall facing it is lit, the other in
// shade, in two steps. A sunk floor is darker, and the near wall's shadow falls
// further onto it.
float depthShade(vec2 origin, float u, float w, float mode, vec2 l) {
  float lit = 0.0;
  float shade = 0.0;
  for (int k = 1; k <= 2; k++) {
    vec2 d = l * float(k) * 0.5;
    vec2 far = mode > 0.0 ? d * 2.0 : d;
    shade += 1.0 - cellAlpha(origin, u + far.x, w + far.y);
    lit += 1.0 - cellAlpha(origin, u - d.x, w - d.y);
  }
  lit *= 0.5;
  shade *= 0.5;
  if (mode > 0.0) return 0.32 * lit - 0.3 * shade - 0.16;
  return 0.3 * shade - 0.32 * lit + 0.04;
}

float stamps(int i, vec3 p, vec3 n, float island) {
  int first = int(uC[i].x);
  int count = int(uC[i].y);
  float v = 0.0;
  for (int k = 0; k < ${maxStamps}; k++) {
    if (k >= count) break;
    int j = first + k;
    vec3 sn = uSN[j].xyz;
    vec3 st = uST[j].xyz;
    vec3 d = wrapped(p - uSC[j].xyz);
    float u = dot(d, st) / uSC[j].w;
    float w = dot(d, cross(sn, st)) / uST[j].w;
    float h = dot(d, sn) / uSC[j].w;
    if (abs(u) > 1.0 || abs(w) > 1.0 || abs(h) > 0.5 || dot(n, sn) < 0.2) continue;

    float cells = uAtlasGrid * uAtlasGrid;
    float only = floor(uSN[j].w / cells) - 1.0;
    if (only >= 0.0 && abs(island - only) > 0.5) continue;
    float cell = mod(uSN[j].w, cells);
    vec2 origin = vec2(mod(cell, uAtlasGrid), floor(cell / uAtlasGrid));
    vec2 local = vec2(u * 0.5 + 0.5, 0.5 - w * 0.5) * (1.0 - 2.0 * uAtlasInset) + uAtlasInset;
    vec4 s = texture(uAtlas, (origin + local) / uAtlasGrid);
    v = max(v, s.a);
    if (s.a > gStampA) {
      gStampA = s.a;
      gStampRGB = s.rgb;
      if (uC[i].z != 0.0) {
        vec3 sw = cross(sn, st);
        vec2 l = vec2(dot(uL, st), dot(uL, sw));
        l = length(l) < 0.15 ? vec2(-0.7, 0.7) : normalize(l);
        // The bevel as a part of the width, the same across both axes.
        float b = uC[i].w * 2.0;
        gDepth = depthShade(origin, u, w, uC[i].z, l * vec2(b, b * uSC[j].w / uST[j].w));
      }
    }
  }
  return v;
}

// 1 in the middle of the volume, 0 on its walls, below outside.
float inside(int i, vec3 p) {
  vec3 l = (uV[i] * vec4(p, 1.0)).xyz;
  int shape = int(uC[i].z);
  if (shape == 1) return 1.0 - max(length(l.xz), abs(l.y));
  if (shape == 2) return 1.0 - length(l);
  vec3 a = abs(l);
  return 1.0 - max(a.x, max(a.y, a.z));
}

float volumeAt(int i, vec3 p) {
  float v = inside(i, p);
  if (uC[i].w > 0.5) v = max(v, inside(i, vec3(2.0 * uMirrorX - p.x, p.yz)));
  return v;
}

// On a tile, the neighbors' volumes reach across the seams.
float volume(int i, vec3 p) {
  if (uPeriod == vec3(0.0)) return volumeAt(i, p);
  float v = -1e9;
  for (int y = -1; y <= 1; y++) {
    for (int x = -1; x <= 1; x++) v = max(v, volumeAt(i, p + uPeriod * vec3(x, y, 0)));
  }
  return v;
}

// World size of a texel of the finished texture, from its nearest neighbor.
float texelSize(ivec2 t, vec3 p) {
  ivec2 last = textureSize(uPos, 0) - 1;
  float s = 1e9;
  for (int k = 0; k < 4; k++) {
    ivec2 o = k < 2 ? ivec2(k * 2 - 1, 0) : ivec2(0, k * 2 - 5);
    float d = distance(texelFetch(uPos, clamp(t + o, ivec2(0), last), 0).xyz, p);
    if (d > 1e-7) s = min(s, d);
  }
  return s * uFactor;
}

// Edges and cavities, 1 on the edge and 0 at their width, which crisp ones give in texels.
float edgeField(int i, ivec2 t, float f, float width, vec3 p) {
  float crisp = uC[i].z;
  if (crisp <= 0.0) return f;
  // Well below any cut beyond the line, so rough noise can't lift it there.
  if (f <= 0.0) return -1.0;
  return 1.0 - (1.0 - f) * width / (texelSize(t, p) * crisp);
}

float field(int i, ivec2 t, vec4 geo, vec4 pos, vec3 n) {
  int k = uKind[i];
  // Axes 3 to 5 are 0 to 2 the other way round.
  int axis = int(uB[i].x) % 3;
  bool flip = int(uB[i].x) >= 3;
  if (k == 7) return paintAt(i, t);
  if (k == 8) return stamps(i, pos.xyz, n, texelFetch(uNrm, t, 0).w);
  if (k == 9) return grunge(i, pos.xyz, n);
  if (k == 11) return volume(i, pos.xyz);
  // Baked onto the coverage, which is 0 or 1.
  if (k == 12) return max(geo.a - 1.0, 0.0);
  if (k == 1) return 1.0 - geo.r;
  if (k == 2) return edgeField(i, t, geo.g, uWidths.x, pos.xyz);
  if (k == 3) return edgeField(i, t, geo.b, uWidths.y, pos.xyz);
  if (k == 4) {
    float g = (pos.xyz / uExtent)[axis];
    return flip ? g : 1.0 - g;
  }
  if (k == 5) return (flip ? -n[axis] : n[axis]) * 0.5 + 0.5;
  return noiseAt(pos.xyz, vec3(uA[i].w), uB[i].y * 1.37);
}

// Picked parts and islands, by their row in the lookups: masks first, then folder zones.
float surfaces(int row, ivec2 t, vec4 pos) {
  float p = texelFetch(uParts, ivec2(int(pos.w), row), 0).r;
  int island = int(texelFetch(uNrm, t, 0).w);
  if (island < 0 || island >= ${maxIslands}) return p;
  return max(p, texelFetch(uIslands, ivec2(island, row), 0).r);
}

float mask(int i, ivec2 t, vec4 geo, vec4 pos, vec3 n) {
  float m;
  if (uKind[i] == 0) {
    m = surfaces(int(uB[i].w), t, pos);
  } else if (uKind[i] == 7) {
    // Painted values are coverage as they are, so soft brushes stay soft.
    m = clamp(paintAt(i, t) * uA[i].x * 2.0, 0.0, 1.0);
  } else {
    float f = field(i, t, geo, pos, n);
    float rough = uA[i].z;
    if (rough > 0.0 && uKind[i] != 6 && uKind[i] != 9) {
      // Drips get noise stretched downwards, so each runs its own length.
      vec3 s = uKind[i] == 12 ? vec3(1.0, 0.06, 1.0) : vec3(1.0);
      f += (noiseAt(pos.xyz, s * uA[i].w, uB[i].y * 2.91 + 5.3) - 0.5) * rough;
    }
    float cut = 1.0 - uA[i].x * 1.0001;
    float soft = uA[i].y * 0.5 + 0.0001;
    m = smoothstep(cut - soft, cut + soft, f);
  }
  return uB[i].z > 0.5 ? 1.0 - m : m;
}

vec3 rgb2hsl(vec3 c) {
  float hi = max(max(c.r, c.g), c.b);
  float lo = min(min(c.r, c.g), c.b);
  float l = (hi + lo) * 0.5;
  float d = hi - lo;
  if (d < 1e-5) return vec3(0.0, 0.0, l);
  float h = hi == c.r ? mod((c.g - c.b) / d, 6.0) : hi == c.g ? (c.b - c.r) / d + 2.0 : (c.r - c.g) / d + 4.0;
  return vec3(h / 6.0, d / (1.0 - abs(2.0 * l - 1.0)), l);
}

vec3 hsl2rgb(vec3 hsl) {
  vec3 k = clamp(abs(mod(hsl.x * 6.0 + vec3(0.0, 4.0, 2.0), 6.0) - 3.0) - 1.0, 0.0, 1.0);
  return hsl.z + (k - 0.5) * (1.0 - abs(2.0 * hsl.z - 1.0)) * hsl.y;
}

vec3 adjust(vec3 c) {
  if (uAdjust == vec3(0.0, 1.0, 0.0)) return c;
  vec3 hsl = rgb2hsl(c);
  hsl = vec3(fract(hsl.x + uAdjust.x), clamp(hsl.y * uAdjust.y, 0.0, 1.0), clamp(hsl.z + uAdjust.z, 0.0, 1.0));
  return hsl2rgb(hsl);
}

vec3 blend(vec3 a, vec3 b) {
  if (uBlend == 1) return a * b;
  if (uBlend == 2) return a + b;
  if (uBlend == 3) return mix(2.0 * a * b, 1.0 - 2.0 * (1.0 - a) * (1.0 - b), step(0.5, a));
  // Mid gray keeps the surface, darker and lighter shift it by the same amount on any color.
  if (uBlend == 4) return a + (b - 0.5) * 0.7;
  return b;
}

// The folder's zone, or everywhere without one.
float zone(ivec2 t, vec4 pos) {
  return uZone < 0 ? 1.0 : surfaces(uZone, t, pos);
}

void main() {
  ivec2 t = ivec2(gl_FragCoord.xy);
  vec4 geo = texelFetch(uGeo, t, 0);
  vec4 pos = texelFetch(uPos, t, 0);
  vec3 n = texelFetch(uNrm, t, 0).xyz;

  float m = zone(t, pos);
  for (int i = 0; i < ${maxMasks}; i++) {
    if (i >= uCount) break;
    m *= mask(i, t, geo, pos, n);
  }

  if (uMaskOnly == 1) {
    outColor = vec4(vec3(m), 1.0);
    return;
  }

  vec3 c = uColor;
  if (uPaintColor >= 0) {
    // Dabs blend color over what is there, weighted by coverage; dividing it back out
    // gives the color itself, also at soft edges.
    vec4 p = paintTexel(uPaintColor, t);
    if (p.a > 0.0) c = clamp(p.rgb / p.a, 0.0, 1.0);
  }
  if (uStampColors == 1 && gStampA > 0.0) c = gStampRGB;
  c = clamp(c + gDepth, 0.0, 1.0);
  if (uFromSource == 1) c = texelFetch(uSource, t, 0).rgb;

  // The layer's colors before adjusting, weighted by its mask, for matching them.
  if (uMaskOnly == 2) {
    outColor = vec4(c * m, m);
    return;
  }
  c = adjust(c);

  float v = (noiseAt(pos.xyz, vec3(uToneScale), uSeed) - 0.5) * 2.0 * uTone;
  v += (hash(vec3(vec2(t), uSeed)) - 0.5) * 2.0 * uGrain;
  c = clamp(c * (1.0 + v), 0.0, 1.0);

  if (uShine > 0.0) {
    float sheen = shade(n, uL, 10.0, false) * uShine;
    c = screen(c, clamp(uColor * 3.0 + 0.08, 0.0, 1.0) * sheen);
    float glint = max(shade(n, uL, 90.0, false), shade(n, uL2, 200.0, false)) * uShine * 0.8;
    c = screen(c, vec3(glint));
  }

  vec4 under = texelFetch(uPrev, t, 0);
  vec3 prev = under.rgb;

  // The emissive pass: glowing layers add their color, others cover it.
  if (uEmitPass == 1) {
    outColor = vec4(mix(prev, uGlow == 1 ? c : vec3(0.0), m * uOpacity), 1.0);
    return;
  }

  // Coverage builds up in alpha, so flattening onto nothing gives premultiplied colors.
  float k = m * uOpacity;
  outColor = vec4(clamp(mix(prev, blend(prev, c), k), 0.0, 1.0), under.a + k * (1.0 - under.a));
}
`

// WebKit on macOS links at most 16 textures per shader; past that the layers just don't draw.
if (import.meta.env.DEV && (layerFragment.match(/uniform sampler2D/g) ?? []).length > 16) {
  throw new Error('the layer shader samples more than 16 textures')
}

// Light baked into the texture: AO, form light, a colored key, a dark under light and lit rims.
export const lightFragment = /* glsl */ `
precision highp float;
precision highp int;

out vec4 outColor;

uniform sampler2D uSrc;
uniform sampler2D uGeo;
uniform sampler2D uNrm;
uniform vec3 uL;
uniform vec3 uUnderL;
uniform int uBoth;
uniform float uAoFloor;
uniform float uFormFloor;
uniform float uKey;
uniform vec3 uKeyColor;
uniform float uUnder;
uniform vec3 uUnderColor;
uniform float uRim;

${shade}

void main() {
  ivec2 t = ivec2(gl_FragCoord.xy);
  vec3 c = texelFetch(uSrc, t, 0).rgb;
  vec4 geo = texelFetch(uGeo, t, 0);
  vec3 n = texelFetch(uNrm, t, 0).xyz;
  float ao = geo.r;

  c *= mix(uAoFloor, 1.0, pow(ao, 1.0 / 1.2));
  c *= mix(uFormFloor, 1.1, shade(n, uL, 1.0, true));

  float key = shade(n, uL, 2.0, false);
  c = mix(c, screen(c, uKeyColor), key * ao * uKey);
  c = mix(c, uUnderColor, shade(n, uUnderL, 1.5, false) * uUnder);

  float rim = smoothstep(0.6, 0.8, geo.g) * key;
  c = mix(c, vec3(1.0, 0.91, 0.78), rim * uRim);

  outColor = vec4(clamp(c, 0.0, 1.0), 1.0);
}
`

// Box-filters uFactor² texels into one, then sharpens against the neighbors.
export const reduceFragment = /* glsl */ `
precision highp float;
precision highp int;

out vec4 outColor;

uniform sampler2D uSrc;
uniform int uFactor;
uniform float uSharpen;
// A tile's texels: it repeats past them, so the rest of the texture holds copies.
uniform ivec2 uWrap;

vec3 block(ivec2 o) {
  ivec2 size = textureSize(uSrc, 0) / uFactor;
  o = uWrap.x > 0 ? (o % uWrap + uWrap) % uWrap : clamp(o, ivec2(0), size - 1);
  vec3 s = vec3(0.0);
  for (int y = 0; y < 4; y++) {
    for (int x = 0; x < 4; x++) {
      if (x >= uFactor || y >= uFactor) continue;
      s += texelFetch(uSrc, o * uFactor + ivec2(x, y), 0).rgb;
    }
  }
  return s / float(uFactor * uFactor);
}

void main() {
  ivec2 t = ivec2(gl_FragCoord.xy);
  vec3 c = block(t);
  vec3 soft = vec3(0.0);
  for (int y = -1; y <= 1; y++) {
    for (int x = -1; x <= 1; x++) soft += block(t + ivec2(x, y));
  }
  soft /= 9.0;
  outColor = vec4(clamp(c + (c - soft) * uSharpen, 0.0, 1.0), 1.0);
}
`

// Ordered dither, then the nearest palette color.
export const finishFragment = /* glsl */ `
precision highp float;
precision highp int;

out vec4 outColor;

uniform sampler2D uSrc;
uniform sampler2D uPalette;
uniform int uColors;
uniform float uDither;

const float bayer[16] = float[16](0., 8., 2., 10., 12., 4., 14., 6., 3., 11., 1., 9., 15., 7., 13., 5.);

void main() {
  ivec2 t = ivec2(gl_FragCoord.xy);
  vec3 c = texelFetch(uSrc, t, 0).rgb;

  if (uDither > 0.0) {
    float b = (bayer[(t.y % 4) * 4 + t.x % 4] + 0.5) / 16.0 - 0.5;
    c += b * uDither * 0.15;
  }

  if (uColors > 0) {
    vec3 best = c;
    float bestDist = 1e9;
    for (int i = 0; i < 256; i++) {
      if (i >= uColors) break;
      vec3 p = texelFetch(uPalette, ivec2(i, 0), 0).rgb;
      vec3 d = p - c;
      float dist = dot(d * d, vec3(0.3, 0.59, 0.11));
      if (dist < bestDist) {
        best = p;
        bestDist = dist;
      }
    }
    c = best;
  }

  outColor = vec4(clamp(c, 0.0, 1.0), 1.0);
}
`
