const DEFAULT_MS = 300
const REDUCED_MS = 0

/** Delay before post-login navigation; immediate when reduced motion is requested. */
export function authHandoffDelayMs(): number {
  if (typeof window === 'undefined') return DEFAULT_MS
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return REDUCED_MS
  return DEFAULT_MS
}
