import { env } from '@/shared/config'

/**
 * Derive the canonical sourceRef recorded by the backend from a tus upload URL.
 * tus Location / upload.url is like: {endpoint}/{objectKey}+{uploadId}
 * Backend finalizes: s3://{bucket}/{objectKey}  (objectKey = owner-scoped path).
 */
export function sourceRefFromUploadUrl(uploadUrl: string): string | null {
  try {
    const u = new URL(uploadUrl)
    const filesPrefix = '/files/'
    const idx = u.pathname.indexOf(filesPrefix)
    if (idx < 0) return null
    let handle = u.pathname.slice(idx + filesPrefix.length) // objectKey+uploadId
    handle = handle.replace(/^\/+/, '')
    const plus = handle.indexOf('+')
    const objectKey = plus >= 0 ? handle.slice(0, plus) : handle
    if (!objectKey) return null
    return `s3://${env.artifactsBucket}/${objectKey}`
  } catch {
    return null
  }
}
