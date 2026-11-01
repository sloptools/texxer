export const quadVertex = /* glsl */ `
void main() {
  gl_Position = vec4(position.xy, 0.0, 1.0);
}
`

export const maxMasks = 4
export const maxStamps = 96

export const kindIndex = { parts: 0, ao: 1, edges: 2, cavity: 3, gradient: 4, facing: 5, noise: 6, paint: 7, stamps: 8, grunge: 9, islands: 10 } as const

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

float vnoise(vec3 x) {
  vec3 i = floor(x);
  vec3 f = fract(x);
  f = f * f * (3.0 - 2.0 * f);
  return mix(
    mix(mix(hash(i), hash(i + vec3(1, 0, 0)), f.x), mix(hash(i + vec3(0, 1, 0)), hash(i + vec3(1, 1, 0)), f.x), f.y),
    mix(mix(hash(i + vec3(0, 0, 1)), hash(i + vec3(1, 0, 1)), f.x), mix(hash(i + vec3(0, 1, 1)), hash(i + vec3(1, 1, 1)), f.x), f.y),
    f.z);
}

// Four octaves, spread back out to roughly 0..1.
float fbm(vec3 p) {
  float s = 0.0;
  float a = 0.5;
  for (int i = 0; i < 4; i++) {
    s += a * vnoise(p);
    p = p * 2.03 + 11.7;
    a *= 0.5;
  }
  return clamp((s / 0.9375 - 0.5) * 2.2 + 0.5, 0.0, 1.0);
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
uniform vec3 uColor;
uniform float uTone;
uniform float uToneScale;
uniform float uGrain;
uniform float uShine;
uniform float uSeed;
uniform int uStampColors;
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
// Stamps of the layer: first index and count per mask.
uniform vec4 uC[${maxMasks}];
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

// Sampler arrays can't take a loop index in GLSL ES 3.0.
float paintAt(int i, ivec2 t) {
  if (i == 0) return texelFetch(uPaint0, t, 0).r;
  if (i == 1) return texelFetch(uPaint1, t, 0).r;
  if (i == 2) return texelFetch(uPaint2, t, 0).r;
  return texelFetch(uPaint3, t, 0).r;
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
  float s = uA[i].w;
  return grungeAt(i, vec2(p.z, -p.y) * s + o) * w.x
    + grungeAt(i, p.xz * s + o) * w.y
    + grungeAt(i, vec2(p.x, -p.y) * s + o) * w.z;
}

// The color under the strongest stamp, for layers that take their color from stamps.
vec3 gStampRGB = vec3(0.0);
float gStampA = 0.0;

float stamps(int i, vec3 p, vec3 n) {
  int first = int(uC[i].x);
  int count = int(uC[i].y);
  float v = 0.0;
  for (int k = 0; k < ${maxStamps}; k++) {
    if (k >= count) break;
    int j = first + k;
    vec3 sn = uSN[j].xyz;
    vec3 st = uST[j].xyz;
    vec3 d = p - uSC[j].xyz;
    float u = dot(d, st) / uSC[j].w;
    float w = dot(d, cross(sn, st)) / uST[j].w;
    float h = dot(d, sn) / uSC[j].w;
    if (abs(u) > 1.0 || abs(w) > 1.0 || abs(h) > 0.5 || dot(n, sn) < 0.2) continue;

    float cell = uSN[j].w;
    vec2 origin = vec2(mod(cell, uAtlasGrid), floor(cell / uAtlasGrid));
    vec2 local = vec2(u * 0.5 + 0.5, 0.5 - w * 0.5) * (1.0 - 2.0 * uAtlasInset) + uAtlasInset;
    vec4 s = texture(uAtlas, (origin + local) / uAtlasGrid);
    v = max(v, s.a);
    if (s.a > gStampA) {
      gStampA = s.a;
      gStampRGB = s.rgb;
    }
  }
  return v;
}

float field(int i, ivec2 t, vec4 geo, vec4 pos, vec3 n) {
  int k = uKind[i];
  int axis = int(uB[i].x);
  if (k == 7) return paintAt(i, t);
  if (k == 8) return stamps(i, pos.xyz, n);
  if (k == 9) return grunge(i, pos.xyz, n);
  if (k == 1) return 1.0 - geo.r;
  if (k == 2) return geo.g;
  if (k == 3) return geo.b;
  if (k == 4) return 1.0 - (pos.xyz / uExtent)[axis];
  if (k == 5) return n[axis] * 0.5 + 0.5;
  return fbm(pos.xyz * uA[i].w + uB[i].y * 1.37);
}

float mask(int i, ivec2 t, vec4 geo, vec4 pos, vec3 n) {
  float m;
  if (uKind[i] == 0) {
    m = texelFetch(uParts, ivec2(int(pos.w), int(uB[i].w)), 0).r;
  } else if (uKind[i] == 10) {
    m = texelFetch(uIslands, ivec2(int(texelFetch(uNrm, t, 0).w), int(uB[i].w)), 0).r;
  } else {
    float f = field(i, t, geo, pos, n);
    float rough = uA[i].z;
    if (rough > 0.0 && uKind[i] != 6 && uKind[i] != 9) {
      f += (fbm(pos.xyz * uA[i].w + uB[i].y * 2.91 + 5.3) - 0.5) * rough;
    }
    float cut = 1.0 - uA[i].x * 1.0001;
    float soft = uA[i].y * 0.5 + 0.0001;
    m = smoothstep(cut - soft, cut + soft, f);
  }
  return uB[i].z > 0.5 ? 1.0 - m : m;
}

vec3 blend(vec3 a, vec3 b) {
  if (uBlend == 1) return a * b;
  if (uBlend == 2) return a + b;
  if (uBlend == 3) return mix(2.0 * a * b, 1.0 - 2.0 * (1.0 - a) * (1.0 - b), step(0.5, a));
  // Mid gray keeps the surface, darker and lighter shift it by the same amount on any color.
  if (uBlend == 4) return a + (b - 0.5) * 0.7;
  return b;
}

// The folder's zone: its parts and islands, or everywhere without one. Zones have
// rows of their own below the masks' in the parts and islands lookups.
float zone(ivec2 t, vec4 pos) {
  if (uZone < 0) return 1.0;
  float p = texelFetch(uParts, ivec2(int(pos.w), uZone), 0).r;
  float i = texelFetch(uIslands, ivec2(int(texelFetch(uNrm, t, 0).w), uZone), 0).r;
  return max(p, i);
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

  vec3 c = uStampColors == 1 && gStampA > 0.0 ? gStampRGB : uColor;
  float v = (fbm(pos.xyz * uToneScale + uSeed) - 0.5) * 2.0 * uTone;
  v += (hash(vec3(vec2(t), uSeed)) - 0.5) * 2.0 * uGrain;
  c = clamp(c * (1.0 + v), 0.0, 1.0);

  if (uShine > 0.0) {
    float sheen = shade(n, uL, 10.0, false) * uShine;
    c = screen(c, clamp(uColor * 3.0 + 0.08, 0.0, 1.0) * sheen);
    float glint = max(shade(n, uL, 90.0, false), shade(n, uL2, 200.0, false)) * uShine * 0.8;
    c = screen(c, vec3(glint));
  }

  vec3 prev = texelFetch(uPrev, t, 0).rgb;

  // The emissive pass: glowing layers add their color, others cover it.
  if (uEmitPass == 1) {
    outColor = vec4(mix(prev, uGlow == 1 ? c : vec3(0.0), m * uOpacity), 1.0);
    return;
  }

  outColor = vec4(clamp(mix(prev, blend(prev, c), m * uOpacity), 0.0, 1.0), 1.0);
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

vec3 block(ivec2 o) {
  ivec2 size = textureSize(uSrc, 0) / uFactor;
  o = clamp(o, ivec2(0), size - 1);
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
