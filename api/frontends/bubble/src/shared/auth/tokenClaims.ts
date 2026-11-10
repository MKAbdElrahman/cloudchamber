/** Parsed claims from CloudChamber service JWTs (`make gentoken`). */
export type ServiceJwtClaims = {
  sub: string
  iss: string | null
  roles: string[]
  issuedAt: Date | null
  expiresAt: Date | null
}

const UUID_RE =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i

function decodePayloadSegment(segment: string): Record<string, unknown> | null {
  try {
    const base64 = segment.replace(/-/g, '+').replace(/_/g, '/')
    const padded = base64 + '='.repeat((4 - (base64.length % 4)) % 4)
    return JSON.parse(atob(padded)) as Record<string, unknown>
  } catch {
    return null
  }
}

function numericSecondsToDate(value: unknown): Date | null {
  if (typeof value !== 'number' || !Number.isFinite(value)) return null
  return new Date(value * 1000)
}

/** Best-effort JWT payload read for display (no signature verification). */
export function parseServiceJwt(token: string): ServiceJwtClaims | null {
  const segment = token.split('.')[1]
  if (!segment) return null

  const json = decodePayloadSegment(segment)
  if (!json) return null

  const sub = typeof json.sub === 'string' ? json.sub.trim() : ''
  if (!sub) return null

  const roles = Array.isArray(json.roles)
    ? json.roles.filter((role): role is string => typeof role === 'string')
    : []

  return {
    sub,
    iss: typeof json.iss === 'string' ? json.iss : null,
    roles,
    issuedAt: numericSecondsToDate(json.iat),
    expiresAt: numericSecondsToDate(json.exp),
  }
}

/** @deprecated Prefer {@link parseServiceJwt}. */
export function jwtSubject(token: string): string | null {
  return parseServiceJwt(token)?.sub ?? null
}

export function isUserUuid(value: string): boolean {
  return UUID_RE.test(value.trim())
}

function formatRoleLabel(role: string): string {
  switch (role.toUpperCase()) {
    case 'USER':
      return 'Pipeline user'
    case 'ADMIN':
      return 'Administrator'
    default:
      return role.charAt(0).toUpperCase() + role.slice(1).toLowerCase()
  }
}

/**
 * Header / menu label. Credential login name wins; else role + short id for UUID subs
 * (avoids showing a lone digit from avatar-style initials on `4…` subjects).
 */
export function accountDisplayLabel(
  claims: ServiceJwtClaims | null,
  preferredName: string | null,
): string {
  const preferred = preferredName?.trim()
  if (preferred) return preferred

  if (!claims) return 'Account'

  const primaryRole = claims.roles[0]
  if (primaryRole) return formatRoleLabel(primaryRole)

  const sub = claims.sub.trim()
  if (/^\d+$/.test(sub)) return 'Account'

  const emailLocal = sub.includes('@') ? sub.split('@')[0]?.trim() : ''
  if (emailLocal) return emailLocal

  if (isUserUuid(sub)) return `User · ${sub.slice(0, 8)}`

  if (sub.length > 32) return `${sub.slice(0, 8)}…`

  return sub
}

export function shortUserId(sub: string): string {
  if (isUserUuid(sub)) return sub
  if (sub.length > 36) return `${sub.slice(0, 8)}…${sub.slice(-4)}`
  return sub
}
