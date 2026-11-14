import { describe, expect, it } from 'vitest'

import { mergeFactPage, pageHasUpdates, withFactPage } from './facts'
import type { PipelineJob } from './types'

const job: PipelineJob = {
  id: 'job-1',
  status: 'running',
  facts: [{ jobId: 'job-1', position: 1, globalPosition: 1, fact: 'sourceResolved' }],
}

describe('mergeFactPage', () => {
  it('copies the folded row and keeps the step facts', () => {
    const next = withFactPage(job, {
      cursor: 2,
      facts: [{ jobId: 'job-1', position: 2, globalPosition: 2, fact: 'runFailed', message: 'source missing' }],
      projections: [{ id: 'job-1', status: 'failed', failureMessage: 'source missing', runId: 'run-1' }],
    })
    expect(next.status).toBe('failed')
    expect(next.failureMessage).toBe('source missing')
    expect(next.runId).toBe('run-1')
    expect(next.facts?.map((fact) => fact.position)).toEqual([1, 2])

    const listed = mergeFactPage([job], {
      cursor: 2,
      facts: [{ jobId: 'job-1', position: 2, globalPosition: 2, fact: 'runCompleted' }],
      projections: [{ id: 'job-1', status: 'completed', failureMessage: null, documentFileId: 'doc-1' }],
    })
    expect(listed.unknown).toBe(false)
    expect(listed.jobs[0]?.status).toBe('completed')
    expect(listed.jobs[0]?.documentFileId).toBe('doc-1')
    expect(listed.jobs[0]?.failureMessage).toBeNull()
  })

  it('keeps a page that only carries a progress update', () => {
    const page = {
      cursor: 4,
      facts: [],
      progress: [{ jobId: 'job-1', position: 4, globalPosition: 4, stage: 'audio', fraction: 0.4, headline: 'Writing the audio.' }],
    }
    expect(pageHasUpdates(page)).toBe(true)
    const listed = mergeFactPage([job], page)
    expect(listed.jobs[0]?.progress?.map((update) => update.position)).toEqual([4])
  })

  it('ignores a page with nothing new', () => {
    expect(pageHasUpdates({ cursor: 4, facts: [], progress: [] })).toBe(false)
  })
})
