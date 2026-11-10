import type { OutputFileKind } from './types'

export function outputFileKindLabel(kind: OutputFileKind): string {
  switch (kind) {
    case 'video':
      return 'Video'
    case 'audio':
      return 'Audio'
    case 'pdf':
      return 'PDF'
    case 'markdown':
      return 'Markdown'
  }
}
