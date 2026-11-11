function trimEndSlash(value: string | undefined, fallback: string): string {
  const v = (value ?? fallback).trim()
  return v.endsWith('/') ? v.slice(0, -1) : v
}

export const env = {
  /** cloudchamber API (jobs, etc.), e.g. http://localhost:3000 */
  cloudchamberApiBase: trimEndSlash(import.meta.env.VITE_CLOUDCHAMBER_API_BASE, 'http://localhost:3000'),
  /** auth service (token issuance). 6100 is the browser port; Chrome refuses 6000. */
  authApiBase: trimEndSlash(import.meta.env.VITE_AUTH_API_BASE, 'http://127.0.0.1:6100'),
  /** tusd endpoint, e.g. http://localhost:8180/files/ */
  tusdEndpoint: (import.meta.env.VITE_TUSD_ENDPOINT as string | undefined) ?? 'http://localhost:8180/files/',
  /** S3-compatible artifacts bucket shown in settings. The browser does not build storage locations. */
  artifactsBucket: (import.meta.env.VITE_ARTIFACTS_BUCKET as string | undefined) ?? 'cloudchamber-artifacts',
}
