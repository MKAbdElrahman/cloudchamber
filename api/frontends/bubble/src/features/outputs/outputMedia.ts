import { playbackUrlForObjectKey, sourceMediaKind, type SourceMediaKind } from '@/shared/media'

import type { OutputFile } from './types'

export function outputMediaKindForFile(file: OutputFile): SourceMediaKind {
  if (file.kind === 'video' || file.kind === 'audio') return file.kind
  return sourceMediaKind(file.name)
}

export function playbackUrlForOutput(file: OutputFile): string | null {
  return playbackUrlForObjectKey(`outputs/${file.jobId}/${file.name}`)
}
