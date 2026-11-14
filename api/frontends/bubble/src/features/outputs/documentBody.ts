import { apiFetch } from '@/shared/api'

import type { ProjectFile, ProjectFileKind } from './projectFiles'

export type DocumentBody = {
  format: ProjectFileKind
  markdown: string
  /** True when the export file is not available and the page shows the stand-in notes. */
  mocked: boolean
}

/** Stand-in for the export body. The pipeline stores a ref and does not yet serve the file. */
export function mockDocumentMarkdown(): string {
  return `# Opening

**0:00** — The recording starts with the topic and what these notes will cover.

# Main points

**4:12** — The first claim, in the speaker's words, followed by the example that supports it.

> A key frame stored for this video. It is not timed to a transcript line.

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
    markdown: mockDocumentMarkdown(),
    mocked: true,
  }
}
