import { describe, expect, it } from 'vitest'

import { filesFromJob, filesFromJobs, type JobArtifacts } from './projectFiles'

function job(partial: JobArtifacts): JobArtifacts {
  return partial
}

describe('filesFromJob', () => {
  it('lists the document and leaves the pipeline artifacts out', () => {
    const files = filesFromJob(
      job({
        id: 'job-1',
        sourceRef: 's3://cloudchamber-artifacts/uploads/week-03.mp4',
        documentRef: 's3://cloudchamber-artifacts/jobs/job-1/exports/document.md',
        dateUpdated: '2026-11-10T18:00:00Z',
      }),
    )
    expect(files).toEqual([
      {
        id: 'job-1:document',
        jobId: 'job-1',
        kind: 'markdown',
        name: 'week-03.md',
        sourceName: 'week-03.mp4',
        ref: 's3://cloudchamber-artifacts/jobs/job-1/exports/document.md',
        dateUpdated: '2026-11-10T18:00:00Z',
      },
    ])
  })

  it('names a pdf export from the recording', () => {
    const files = filesFromJob(
      job({
        id: 'job-pdf',
        sourceRef: 's3://bucket/uploads/week-03.mp4',
        documentRef: 's3://bucket/jobs/job-pdf/exports/document.pdf',
      }),
    )
    expect(files.map((file) => file.kind)).toEqual(['pdf'])
    expect(files.map((file) => file.name)).toEqual(['week-03.pdf'])
  })

  it('keeps the export name when the recording has no filename', () => {
    const files = filesFromJob(
      job({
        id: 'job-3',
        sourceRef: 's3://bucket/uploads/04595d1b-6c37-4fdb-959a-a326597a82d4',
        documentRef: 's3://bucket/jobs/job-3/exports/document.md',
      }),
    )
    expect(files.map((file) => file.name)).toEqual(['document.md'])
  })

  it('returns nothing until the job has written a document', () => {
    expect(filesFromJob(job({ id: 'job-2' }))).toEqual([])
  })
})

describe('filesFromJobs', () => {
  it('orders newer documents first', () => {
    const files = filesFromJobs([
      job({
        id: 'older',
        documentRef: 's3://bucket/jobs/older/exports/document.md',
        dateUpdated: '2026-11-10T12:00:00Z',
      }),
      job({
        id: 'newer',
        documentRef: 's3://bucket/jobs/newer/exports/document.md',
        dateUpdated: '2026-11-10T18:00:00Z',
      }),
    ])
    expect(files.map((file) => file.jobId)).toEqual(['newer', 'older'])
  })
})
