import type { Risk } from '../types/analysis'

/** Returns true if any risk in the list targets the given module with HIGH severity. */
export function hasHighSeverityRisk(moduleId: string, risks: Risk[]): boolean {
  return risks.some((r) => r.module === moduleId && r.severity === 'high')
}

/**
 * Linearly interpolates between two RGB triples.
 * Used for complexity score → colour mapping.
 */
function lerpRgb(
  a: readonly [number, number, number],
  b: readonly [number, number, number],
  t: number,
): string {
  const r = Math.round(a[0] + (b[0] - a[0]) * t)
  const g = Math.round(a[1] + (b[1] - a[1]) * t)
  const bl = Math.round(a[2] + (b[2] - a[2]) * t)
  return `rgb(${r},${g},${bl})`
}

const GREEN = [74, 222, 128] as const // #4ade80
const YELLOW = [250, 204, 21] as const // #facc15
const RED = [248, 113, 113] as const // #f87171

/**
 * Maps a complexity score [0, 1] to a CSS colour string.
 * 0 → green, 0.5 → yellow, 1 → red.
 */
export function complexityColor(score: number): string {
  const c = Math.max(0, Math.min(1, score))
  if (c <= 0.5) return lerpRgb(GREEN, YELLOW, c * 2)
  return lerpRgb(YELLOW, RED, (c - 0.5) * 2)
}
