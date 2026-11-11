import { getJob } from './api'
import { isTerminalJobStatus } from './jobStatus'
import { writeCachedJob } from './pipelineJobCache'
import type { PipelineJob } from './types'

/** Fetch one job and keep the session copy. Callers do not touch the cache. */
export async function loadJob(token: string, jobId: string): Promise<PipelineJob> {
  const job = await getJob(token, jobId)
  writeCachedJob(job)
  return job
}

/** A list row stays as it was stored. Opening the job is what copies a closed execution. */
export function jobNeedsOpen(job: PipelineJob): boolean {
  return !isTerminalJobStatus(job.status) && !job.documentFileId?.trim()
}

/** Open each job the list still has open, so a finished execution can copy its document. */
export async function openUnsettledJobs(token: string, jobs: PipelineJob[]): Promise<PipelineJob[]> {
  return Promise.all(
    jobs.map(async (job) => {
      if (!jobNeedsOpen(job)) return job
      try {
        return await loadJob(token, job.id)
      } catch {
        return job
      }
    }),
  )
}
