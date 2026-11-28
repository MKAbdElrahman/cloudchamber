import type { Conversion } from './types'

/**
 * A session copy of each conversion, so a page that opens one shows it at once
 * and then reads the current state. The key carries a version: a copy written by
 * an older build has another shape and is not read.
 */
const CACHE_PREFIX = 'bubble.conversion.v2.'

function storageKey(jobId: string) {
  return `${CACHE_PREFIX}${jobId}`
}

export function readCachedJob(jobId: string): Conversion | undefined {
  try {
    const raw = sessionStorage.getItem(storageKey(jobId))
    if (!raw) return undefined
    const job = JSON.parse(raw) as Conversion
    if (typeof job.id === 'string' && typeof job.status === 'string' && job.run?.status?.phase) return job
  } catch {
    /* ignore */
  }
  return undefined
}

export function writeCachedJob(job: Conversion) {
  try {
    sessionStorage.setItem(storageKey(job.id), JSON.stringify(job))
  } catch {
    /* ignore */
  }
}
