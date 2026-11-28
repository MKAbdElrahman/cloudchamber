import { isRunMoving } from '@/shared/runs'

import { getJob } from './api'
import { writeCachedJob } from './conversionCache'
import type { Conversion } from './types'

/** Fetch one job and keep the session copy. Callers do not touch the cache. */
export async function loadJob(token: string, jobId: string): Promise<Conversion> {
  const job = await getJob(token, jobId)
  writeCachedJob(job)
  return job
}

/** A job is worth reading again while its run is moving. */
export function jobNeedsOpen(job: Conversion): boolean {
  return isRunMoving(job.run)
}
