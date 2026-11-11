import { sourceMediaKind, type SourceMediaKind } from '@/shared/media'

const MEDIA_FILE = /\.(mp4|webm|mov|m4v|mkv|avi|ogv|mp3|wav|ogg|m4a|aac|flac)$/i
const GENERIC_DOCUMENT = /^document\.(md|pdf)$/i

export function mediaLabel(kind: SourceMediaKind): string {
  switch (kind) {
    case 'video':
      return 'Video'
    case 'audio':
      return 'Audio'
    default:
      return 'Recording'
  }
}

export function mediaKindOf(fileName: string | undefined, recordingName?: string): SourceMediaKind {
  const fromName = sourceMediaKind(recordingName ?? '')
  if (fromName !== 'unknown') return fromName
  return sourceMediaKind(fileName ?? '')
}

/** A recording filename we can show. Opaque upload ids are not one. */
export function knownMediaName(fileName: string | undefined, recordingName?: string): string | null {
  const named = recordingName?.trim() ?? ''
  if (MEDIA_FILE.test(named)) return named
  const fromFile = fileName?.trim() ?? ''
  if (MEDIA_FILE.test(fromFile)) return fromFile
  return null
}

/** Clock label for a file whose upload never kept its original name. */
function unnamedFileStamp(date: string | undefined): string | null {
  if (!date) return null
  const parsed = new Date(date)
  if (Number.isNaN(parsed.getTime())) return null
  return new Intl.DateTimeFormat(undefined, {
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  }).format(parsed)
}

export type FileFace = 'video' | 'document' | 'pipeline' | 'library'

/** The name of the face that is open. The library names the document, which is what the row opens. */
export function faceFileName(input: {
  face: FileFace
  fileName?: string
  recordingName?: string
  documentName?: string
  date?: string
}): string {
  const media = knownMediaName(input.fileName, input.recordingName)
  const stamp = unnamedFileStamp(input.date)
  const documentName = input.documentName?.trim() ?? ''
  const namedDocument = documentName.length > 0 && !GENERIC_DOCUMENT.test(documentName) ? documentName : null
  const extension = documentName.includes('.') ? documentName.slice(documentName.lastIndexOf('.') + 1) : ''
  const datedDocument = stamp && extension ? `${stamp}.${extension}` : null

  switch (input.face) {
    case 'video':
      return media ?? stamp ?? 'Recording'
    case 'document':
      return namedDocument ?? datedDocument ?? (documentName || media || stamp || 'Document')
    case 'pipeline':
    case 'library':
      return namedDocument ?? datedDocument ?? (documentName || media || stamp || 'Recording')
  }
}
