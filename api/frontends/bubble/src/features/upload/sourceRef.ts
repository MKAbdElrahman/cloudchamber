import { env } from '@/shared/config'
import { fileIdFromObjectKey } from '@/shared/media'

/**
 * Derive the canonical sourceRef recorded by the backend from a tus upload URL.
 * tus Location / upload.url is like: {endpoint}/{objectKey}+{uploadId}
 * Backend finalizes: s3://{bucket}/{objectKey}  (objectKey = owner-scoped path).
 */
export function sourceRefFromUploadUrl(uploadUrl: string): string | null {
  const objectKey = objectKeyFromUploadUrl(uploadUrl)
  if (!objectKey) return null
  return `s3://${env.artifactsBucket}/${objectKey}`
}

/** File id from a tus upload URL. The storage location stays on the server. */
export function fileIdFromUploadUrl(uploadUrl: string): string | null {
  const objectKey = objectKeyFromUploadUrl(uploadUrl)
  if (!objectKey) return null
  return fileIdFromObjectKey(objectKey)
}

function objectKeyFromUploadUrl(uploadUrl: string): string | null {
  try {
    const u = new URL(uploadUrl)
    const filesPrefix = '/files/'
    const idx = u.pathname.indexOf(filesPrefix)
    if (idx < 0) return null
    let handle = u.pathname.slice(idx + filesPrefix.length)
    handle = handle.replace(/^\/+/, '')
    const plus = handle.indexOf('+')
    const objectKey = plus >= 0 ? handle.slice(0, plus) : handle
    return objectKey || null
  } catch {
    return null
  }
}
