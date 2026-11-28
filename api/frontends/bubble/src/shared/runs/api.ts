import { apiFetch } from '@/shared/api'
import type { Workflow, Run, RunCommand, RunsPage } from './types'

/** A run of every workflow goes through these calls. A workflow differs in its input and its result, not in its endpoints. */

export type NewRun<Input> = {
  workflow: string
  projectId: string
  input: Input
  idempotencyKey?: string
}

export function createRun<Input, Result = unknown>(token: string, run: NewRun<Input>) {
  return apiFetch<Run<Input, Result>>('/v1/runs', {
    method: 'POST',
    token,
    body: JSON.stringify({ ...run, idempotencyKey: run.idempotencyKey || undefined }),
  })
}

export function getRun<Input = unknown, Result = unknown>(token: string, runId: string) {
  return apiFetch<Run<Input, Result>>(`/v1/runs/${encodeURIComponent(runId)}`, { method: 'GET', token })
}

export type RunsFilter = {
  projectId?: string
  workflow?: string
  rows?: number
}

/** Runs the caller owns, newest first. */
export async function listRuns<Input = unknown, Result = unknown>(token: string, filter: RunsFilter = {}) {
  const query = new URLSearchParams({
    page: '1',
    rows: String(filter.rows ?? 50),
    orderBy: 'date_created,DESC',
  })
  if (filter.projectId) query.set('project_id', filter.projectId)
  if (filter.workflow) query.set('workflow', filter.workflow)
  const page = await apiFetch<RunsPage<Input, Result>>(`/v1/runs?${query.toString()}`, { method: 'GET', token })
  return page.items ?? []
}

export function commandRun<Input = unknown, Result = unknown>(token: string, runId: string, command: RunCommand) {
  return apiFetch<Run<Input, Result>>(`/v1/runs/${encodeURIComponent(runId)}/${command}`, { method: 'POST', token })
}

/** The workflows the service can start a run of, with the names of their steps. */
export async function listWorkflows(token: string) {
  const data = await apiFetch<{ items?: Workflow[] }>('/v1/workflows', { method: 'GET', token })
  return data.items ?? []
}
