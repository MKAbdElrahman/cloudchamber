import { apiFetch } from '@/shared/api'

import type { ProjectFile, ProjectFileKind } from './projectFiles'

export type DocumentBody = {
  format: ProjectFileKind
  markdown: string
  /** True when the export file is not available and the page shows the stand-in notes. */
  mocked: boolean
}

/** Heading shown while reading. A generic export name stays "Document". */
export function readingTitle(name: string): string {
  const stem = name.replace(/\.[^.]+$/, '')
  if (stem.length > 0 && stem.toLowerCase() !== 'document') return stem
  return 'Document'
}

/** Stand-in for the export body. The pipeline stores a ref and does not yet serve the file. */
export function mockDocumentMarkdown(input: { title: string; sourceLabel: string }): string {
  const named = input.title === 'Document' ? '' : ` for ${input.title}`
  const fromFile = /\.[a-z0-9]+$/i.test(input.sourceLabel)
  const source = fromFile ? `\`${input.sourceLabel}\`` : 'the recording'
  return `Notes${named}, composed from ${source}. The recording was transcribed, then sectioned and paired with the frames that carry the slides.

# Opening

**0:00** — The recording starts with the topic and what these notes will cover.

# Main points

**4:12** — The first claim, in the speaker's words, followed by the example that supports it.

> Frame at 4:18. A slide with three boxes: the recording, the transcript, and this document.

**11:05** — A second point. The term is defined before it is used again.

# Close

**18:40** — The recording ends on what to review before the next session.
`
}

/** Loads the document the pipeline stored. Older jobs without a file keep the stand-in notes. */
export async function loadDocumentBody(
  token: string,
  file: Pick<ProjectFile, 'name' | 'kind' | 'sourceName' | 'fileId'>,
): Promise<DocumentBody> {
  if (file.fileId) {
    const signed = await apiFetch<{ url: string }>(`/v1/files/${encodeURIComponent(file.fileId)}/url`, { token })
    const res = await fetch(signed.url)
    if (res.ok) {
      return { format: file.kind, markdown: await res.text(), mocked: false }
    }
  }
  return {
    format: file.kind,
    markdown: mockDocumentMarkdown({
      title: readingTitle(file.name),
      sourceLabel: file.sourceName ?? file.name,
    }),
    mocked: true,
  }
}
