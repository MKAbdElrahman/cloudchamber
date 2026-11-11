const VIDEO_EXT = /\.(mp4|webm|mov|m4v|mkv|avi|ogv)$/i
const AUDIO_EXT = /\.(mp3|wav|ogg|m4a|aac|flac)$/i
const FILE_ID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

/** The last path segment when it is the file id baked into a source object key. */
export function fileIdFromObjectKey(objectKey: string): string | null {
  const parts = objectKey.split('/').filter(Boolean)
  const last = parts[parts.length - 1] ?? ''
  return FILE_ID.test(last) ? last.toLowerCase() : null
}

/** File id carried by an older s3:// source ref. New recordings store the id directly. */
export function fileIdFromSourceRef(sourceRef: string | undefined): string | null {
  if (!sourceRef) return null
  const key = objectKeyFromSourceRef(sourceRef) ?? sourceRef
  return fileIdFromObjectKey(key)
}

export type SourceMediaKind = 'video' | 'audio' | 'unknown'

function objectKeyFromSourceRef(sourceRef: string): string | null {
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

