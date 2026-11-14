import { getJob } from './api'
import { isTerminalJobStatus, visibleJobStatus } from './jobStatus'
import { writeCachedJob } from './pipelineJobCache'
import type { PipelineJob } from './types'

/** Fetch one job and keep the session copy. Callers do not touch the cache. */
export async function loadJob(token: string, jobId: string): Promise<PipelineJob> {
  const job = await getJob(token, jobId)
  writeCachedJob(job)
  return job
}

/** A job still needs its stream while the stored row says it is open. */
export function jobNeedsOpen(job: PipelineJob): boolean {
  return !isTerminalJobStatus(visibleJobStatus(job))
}
