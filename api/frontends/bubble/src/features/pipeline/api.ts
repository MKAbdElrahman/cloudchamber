import { apiFetch } from '@/shared/api'
import type { FactPage, JobsQuery, PipelineJob } from './types'

export function createJob(token: string, fileId: string, idempotencyKey?: string) {
  return apiFetch<PipelineJob>('/v1/pipeline/jobs', {
    method: 'POST',
    token,
    body: JSON.stringify({ fileId, idempotencyKey: idempotencyKey || undefined }),
  })
}

export function readFileURL(token: string, fileId: string) {
  return apiFetch<{ url: string; expiresAt: string }>(`/v1/files/${encodeURIComponent(fileId)}/url`, {
    method: 'GET',
    token,
  })
}

export function listJobs(token: string, projectId?: string) {
  const query = new URLSearchParams({
    page: '1',
    rows: '50',
    orderBy: 'date_created,DESC',
  })
  if (projectId) query.set('project_id', projectId)
  return apiFetch<JobsQuery>(`/v1/pipeline/jobs?${query.toString()}`, {
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

export function cancelJob(token: string, jobId: string) {
  return apiFetch<PipelineJob>(`/v1/pipeline/jobs/${encodeURIComponent(jobId)}/cancel`, {
    method: 'POST',
    token,
  })
}

export function followProjectFacts(token: string, projectId: string, after: number) {
  const query = new URLSearchParams({
    project_id: projectId,
    after: String(after),
  })
  return apiFetch<FactPage>(`/v1/pipeline/facts?${query.toString()}`, {
    method: 'GET',
    token,
  })
}

export function followJobFacts(token: string, jobId: string, after: number) {
  const query = new URLSearchParams({ after: String(after) })
  return apiFetch<FactPage>(`/v1/pipeline/jobs/${encodeURIComponent(jobId)}/facts?${query.toString()}`, {
    method: 'GET',
    token,
  })
}

export function factsCursor(data: JobsQuery): number {
  if (Array.isArray(data) || data?.factsCursor == null) return 0
  return data.factsCursor
}

export function normalizeJobs(data: JobsQuery): PipelineJob[] {
  if (Array.isArray(data)) return data
  if (data && Array.isArray(data.items)) return data.items
  return []
}
