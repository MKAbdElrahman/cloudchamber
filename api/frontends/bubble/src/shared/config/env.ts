function trimEndSlash(value: string | undefined, fallback: string): string {
  const v = (value ?? fallback).trim()
  return v.endsWith('/') ? v.slice(0, -1) : v
}

export const env = {
  /** cloudchamber API (jobs, etc.), e.g. http://localhost:3000 */
  cloudchamberApiBase: trimEndSlash(import.meta.env.VITE_CLOUDCHAMBER_API_BASE, 'http://localhost:3000'),
  /** auth service (token issuance), e.g. http://localhost:6000 */
  authApiBase: trimEndSlash(import.meta.env.VITE_AUTH_API_BASE, 'http://localhost:6000'),
  /** key id for GET /v1/auth/token/{kid} */
  authKid: (import.meta.env.VITE_AUTH_KID as string | undefined) ?? '54bb2165-71e1-41a6-af3e-7da4a0e1e2c1',
  /** tusd endpoint, e.g. http://localhost:8180/files/ */
  tusdEndpoint: (import.meta.env.VITE_TUSD_ENDPOINT as string | undefined) ?? 'http://localhost:8180/files/',
  /** S3-compatible artifacts bucket shown in settings. The browser does not build storage locations. */
  artifactsBucket: (import.meta.env.VITE_ARTIFACTS_BUCKET as string | undefined) ?? 'cloudchamber-artifacts',
}
