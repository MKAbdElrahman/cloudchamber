import { apiFetch, apiFetchResult } from '@/shared/api'
import type { JobsQuery, PipelineJob } from './types'

export function createJob(token: string, fileId: string, idempotencyKey?: string) {
  return apiFetch<PipelineJob>('/v1/pipeline/jobs', {
    method: 'POST',
    token,
    body: JSON.stringify({ fileId, idempotencyKey: idempotencyKey || undefined }),
  })
}

export async function readFileURL(token: string, fileId: string) {
  const result = await apiFetchResult<{ url: string; expiresAt: string }>(
    `/v1/files/${encodeURIComponent(fileId)}/url`,
    { method: 'GET', token },
  )
  return {
    url: result.body.url,
    expiresAt: result.body.expiresAt,
    traceparent: result.headers.get('traceparent'),
  }
}

export function reportPlayback(
  token: string,
  fileId: string,
  traceparent: string,
  report: { outcome: 'loaded' | 'error'; readyState: number; waitedMs: number; mediaError?: number },
) {
  return apiFetch<null>(`/v1/files/${encodeURIComponent(fileId)}/playback`, {
    method: 'POST',
    token,
    headers: { traceparent },
    body: JSON.stringify(report),
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
