import type { PipelineJob } from './types'

const CACHE_PREFIX = 'cloudchamber.pipeline.job.v1.'

function storageKey(jobId: string) {
  return `${CACHE_PREFIX}${jobId}`
}

export function readCachedJob(jobId: string): PipelineJob | undefined {
  try {
    const raw = sessionStorage.getItem(storageKey(jobId))
    if (!raw) return undefined
    const job = JSON.parse(raw) as PipelineJob
    if (typeof job.id === 'string' && typeof job.status === 'string') return job
  } catch {
    /* ignore */
  }
  return undefined
}

export function writeCachedJob(job: PipelineJob) {
  try {
    sessionStorage.setItem(storageKey(job.id), JSON.stringify(job))
  } catch {
    /* ignore */
  }
}
