/** Build-time 30° axonometric projector (DESIGN.md §6). */

const CX = Math.cos(Math.PI / 6);

export type Pt = [number, number];

export const P = (x: number, y: number, z: number): Pt => [
  (x - y) * CX,
  (x + y) * 0.5 - z,
];

export const fmt = (pts: Pt[]) =>
  pts.map(([a, b]) => `${a.toFixed(1)},${b.toFixed(1)}`).join(' ');

export function box(x: number, y: number, z: number, w: number, d: number, h: number) {
  const A = P(x, y, z + h);
  const B = P(x + w, y, z + h);
  const C = P(x + w, y + d, z + h);
  const D = P(x, y + d, z + h);
  const B0 = P(x + w, y, z);
  const C0 = P(x + w, y + d, z);
  const D0 = P(x, y + d, z);
  return {
    top: fmt([A, B, C, D]),
    front: fmt([D, C, C0, D0]),
    side: fmt([B, C, C0, B0]),
  };
}
