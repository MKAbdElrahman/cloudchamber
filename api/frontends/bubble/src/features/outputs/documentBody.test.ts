import { describe, expect, it } from 'vitest'

import { loadDocumentBody, readingTitle } from './documentBody'

describe('readingTitle', () => {
  it('uses the recording stem and keeps a generic export unnamed', () => {
    expect(readingTitle('week-03.md')).toBe('week-03')
    expect(readingTitle('document.md')).toBe('Document')
  })
})

describe('loadDocumentBody', () => {
  it('returns readable markdown for a document the API cannot serve yet', async () => {
    const body = await loadDocumentBody({
      name: 'week-03.md',
      kind: 'markdown',
      sourceName: 'week-03.mp4',
    })
    expect(body.mocked).toBe(true)
    expect(body.format).toBe('markdown')
    expect(body.markdown).toContain('Notes for week-03')
    expect(body.markdown).toContain('`week-03.mp4`')
    expect(body.markdown).toContain('# Opening')
  })
})
