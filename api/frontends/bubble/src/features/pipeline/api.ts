import { apiFetch } from '@/shared/api/client'
import type { JobsQuery, PipelineJob } from './types'

export function createJob(token: string, sourceRef: string, idempotencyKey?: string) {
  return apiFetch<PipelineJob>('/v1/pipeline/jobs', {
    method: 'POST',
    token,
    body: JSON.stringify({ sourceRef, idempotencyKey: idempotencyKey || undefined }),
  })
}

export function listJobs(token: string) {
  return apiFetch<JobsQuery>('/v1/pipeline/jobs?page=1&rows=50&orderBy=date_created,DESC', {
    method: 'GET',
    token,
  })
}

export function getJob(token: string, jobId: string) {
  return apiFetch<PipelineJob>(`/v1/pipeline/jobs/${encodeURIComponent(jobId)}`, {
    method: 'GET',
    token,
  })
}

export function normalizeJobs(data: JobsQuery): PipelineJob[] {
  if (Array.isArray(data)) return data
  if (data && Array.isArray(data.items)) return data.items
  return []
}
