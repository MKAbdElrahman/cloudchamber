import { describe, expect, it } from 'vitest'

import { jobFromRun } from './api'
import type { VideoRun } from './types'

function run(partial: Partial<VideoRun['status']>, spec: Partial<VideoRun['spec']> = {}): VideoRun {
  return {
    id: 'run-1',
    workflow: 'video-to-document',
    ownerId: 'owner-1',
    projectId: 'project-1',
    generation: 1,
    settled: false,
    spec: { input: { fileId: 'file-1' }, suspend: false, cancel: false, attempt: 1, resumeFrom: 'start', ...spec },
    status: { phase: 'running', observedGeneration: 1, attempt: 1, steps: [], ...partial },
    dateCreated: '2026-11-28T10:00:00Z',
    dateUpdated: '2026-11-28T10:00:05Z',
  }
}

describe('jobFromRun', () => {
  it('reads the recording from the input and its name from the project files', () => {
    const job = jobFromRun(run({}), new Map([['file-1', 'week-03.mp4']]))
    expect(job).toMatchObject({ id: 'run-1', projectId: 'project-1', fileId: 'file-1', fileName: 'week-03.mp4', status: 'running' })
    expect(job.documentFileId).toBeUndefined()
  })

  it('keeps one status vocabulary for the pages', () => {
    expect(jobFromRun(run({ phase: 'pending' })).status).toBe('submitted')
    expect(jobFromRun(run({ phase: 'succeeded' })).status).toBe('completed')
    expect(jobFromRun(run({ phase: 'terminated' })).status).toBe('canceled')
    expect(jobFromRun(run({ phase: 'paused' })).status).toBe('paused')
  })

  it('reads the produced files from the result of a finished run', () => {
    const job = jobFromRun(
      run({ phase: 'succeeded', result: { sourceFileId: 'file-1', audioFileId: 'audio-1', transcriptFileId: 'transcript-1', documentFileId: 'doc-1' } }),
      new Map([['doc-1', 'week-03.md']]),
    )
    expect(job).toMatchObject({ audioFileId: 'audio-1', transcriptFileId: 'transcript-1', documentFileId: 'doc-1', documentName: 'week-03.md' })
    expect(job.keyFramesFileId).toBeUndefined()
  })

  it('carries the failure message only for a failed run', () => {
    expect(jobFromRun(run({ phase: 'failed', message: 'boom' })).failureMessage).toBe('boom')
    expect(jobFromRun(run({ phase: 'canceled', message: 'ignored' })).failureMessage).toBeNull()
  })

  it('carries the run itself, for the parts of a page that work on any run', () => {
    const source = run({}, { suspend: true })
    expect(jobFromRun(source).run).toBe(source)
  })
})
