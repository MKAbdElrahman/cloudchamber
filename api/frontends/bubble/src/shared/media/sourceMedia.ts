import { env } from '@/shared/config'

const VIDEO_EXT = /\.(mp4|webm|mov|m4v|mkv|avi|ogv)$/i
const AUDIO_EXT = /\.(mp3|wav|ogg|m4a|aac|flac)$/i

export type SourceMediaKind = 'video' | 'audio' | 'unknown'

export function objectKeyFromSourceRef(sourceRef: string): string | null {
  const trimmed = sourceRef.trim()
  const match = /^s3:\/\/([^/]+)\/(.+)$/.exec(trimmed)
  if (!match) return null
  const key = match[2]
  try {
    return decodeURIComponent(key)
  } catch {
    return key
  }
}

export function fileNameFromSourceRef(sourceRef: string): string {
  const key = objectKeyFromSourceRef(sourceRef)
  if (!key) {
    const slash = sourceRef.lastIndexOf('/')
    if (slash >= 0 && slash < sourceRef.length - 1) return sourceRef.slice(slash + 1)
    return sourceRef
  }
  const slash = key.lastIndexOf('/')
  return slash >= 0 ? key.slice(slash + 1) : key
}

export function sourceMediaKind(sourceRef: string): SourceMediaKind {
  const name = fileNameFromSourceRef(sourceRef).toLowerCase()
  if (VIDEO_EXT.test(name)) return 'video'
  if (AUDIO_EXT.test(name)) return 'audio'
  return 'unknown'
}

/** Playable URL for an object key in the tusd S3 store. */
export function playbackUrlForObjectKey(objectKey: string): string {
  const base = env.tusdEndpoint.endsWith('/') ? env.tusdEndpoint : `${env.tusdEndpoint}/`
  const encoded = objectKey.split('/').map(encodeURIComponent).join('/')
  return new URL(encoded, base).href
}

/**
 * Best-effort playable URL for an uploaded object (tusd S3 store).
 * May require same-network access; playback fails gracefully in the UI when unavailable.
 */
export function playbackUrlFromSourceRef(sourceRef: string): string | null {
  const key = objectKeyFromSourceRef(sourceRef)
  if (!key) return null
  return playbackUrlForObjectKey(key)
}
