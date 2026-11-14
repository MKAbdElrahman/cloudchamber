import { describe, expect, it } from 'vitest'

import { loadDocumentBody } from './documentBody'

describe('loadDocumentBody', () => {
  it('returns readable markdown for a document the API cannot serve yet', async () => {
    const body = await loadDocumentBody('token', {
      name: 'week-03.md',
      kind: 'markdown',
      sourceName: 'week-03.mp4',
      fileId: '',
    })
    expect(body.mocked).toBe(true)
    expect(body.format).toBe('markdown')
    expect(body.markdown.startsWith('# Opening')).toBe(true)
  })
})
