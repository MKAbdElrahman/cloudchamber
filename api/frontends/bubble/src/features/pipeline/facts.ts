import type { QueryClient } from '@tanstack/react-query'

import type { FactPage, JobProjection, PipelineFact, PipelineJob, PipelineProgress } from './types'

export function maxFactPosition(rows: { position: number }[] | undefined): number {
  return (rows ?? []).reduce((max, row) => Math.max(max, row.position), 0)
}

/** A follow page is news when it carries a business fact or a progress update. */
export function pageHasUpdates(page: FactPage): boolean {
  return (page.facts?.length ?? 0) > 0 || (page.progress?.length ?? 0) > 0
}

export function factsByJob(facts: PipelineFact[]): Map<string, PipelineFact[]> {
  const byJob = new Map<string, PipelineFact[]>()
  for (const fact of facts) {
    const list = byJob.get(fact.jobId) ?? []
    list.push(fact)
    byJob.set(fact.jobId, list)
  }
  return byJob
}

export function mergeProgress(existing: PipelineProgress[] | undefined, incoming: PipelineProgress[]): PipelineProgress[] {
  const byGlobal = new Map<number, PipelineProgress>()
  for (const update of existing ?? []) byGlobal.set(update.globalPosition, update)
  for (const update of incoming) byGlobal.set(update.globalPosition, update)
  return [...byGlobal.values()].sort((a, b) => a.position - b.position || a.globalPosition - b.globalPosition)
}

export function mergeFacts(existing: PipelineFact[] | undefined, incoming: PipelineFact[]): PipelineFact[] {
  const byGlobal = new Map<number, PipelineFact>()
  for (const fact of existing ?? []) byGlobal.set(fact.globalPosition, fact)
  for (const fact of incoming) byGlobal.set(fact.globalPosition, fact)
  return [...byGlobal.values()].sort((a, b) => a.position - b.position || a.globalPosition - b.globalPosition)
}

/** Copies the folded row onto the job. Facts stay on the job for the step display. */
export function applyProjection(job: PipelineJob, projection: JobProjection): PipelineJob {
  return {
    ...job,
    status: projection.status,
    failureMessage: projection.failureMessage ?? null,
    runId: projection.runId ?? '',
    audioFileId: projection.audioFileId ?? '',
    transcriptFileId: projection.transcriptFileId ?? '',
    keyFramesFileId: projection.keyFramesFileId ?? '',
    documentFileId: projection.documentFileId ?? '',
    dateUpdated: projection.dateUpdated ?? job.dateUpdated,
  }
}

/** Merges one follow page into a job, including the folded row when this page carries it. */
export function withFactPage(job: PipelineJob, page: FactPage): PipelineJob {
  const next = {
    ...job,
    facts: mergeFacts(job.facts, page.facts),
    progress: mergeProgress(job.progress, page.progress ?? []),
  }
  const projection = page.projections?.find((item) => item.id === job.id)
  if (!projection) return next
  return applyProjection(next, projection)
}

/** Folds one follow page into a job list. Unknown is true when a fact names a job the list does not have. */
export function mergeFactPage(jobs: PipelineJob[], page: FactPage): { jobs: PipelineJob[]; unknown: boolean } {
  const merged = mergeJobFacts(jobs, page.facts ?? [])
  const withUpdates = mergeJobProgress(merged.jobs, page.progress ?? [])
  const known = new Set(withUpdates.jobs.map((job) => job.id))
  let unknown = merged.unknown || withUpdates.unknown
  for (const projection of page.projections ?? []) {
    if (!known.has(projection.id)) unknown = true
  }
  return {
    unknown,
    jobs: withUpdates.jobs.map((job) => {
      const projection = page.projections?.find((item) => item.id === job.id)
      return projection ? applyProjection(job, projection) : job
    }),
  }
}

/** Folds incoming facts into a job list. Unknown is true when a fact names a job the list does not have. */
export function mergeJobFacts(jobs: PipelineJob[], incoming: PipelineFact[]): { jobs: PipelineJob[]; unknown: boolean } {
  const byJob = factsByJob(incoming)
  const known = new Set(jobs.map((job) => job.id))
  let unknown = false
  for (const id of byJob.keys()) {
    if (!known.has(id)) unknown = true
  }
  return {
    unknown,
    jobs: jobs.map((job) => withFacts(job, byJob.get(job.id))),
  }
}

/** Folds facts into the project list and any open job. Returns true when a fact names a job the list does not have. */
export function applyProjectFacts(queryClient: QueryClient, projectId: string, page: FactPage): boolean {
  const byJob = factsByJob(page.facts ?? [])
  const updatesByJob = new Map<string, PipelineProgress[]>()
  for (const update of page.progress ?? []) {
    const list = updatesByJob.get(update.jobId) ?? []
    list.push(update)
    updatesByJob.set(update.jobId, list)
  }

  let unknown = false
  queryClient.setQueryData<PipelineJob[]>(['pipeline-jobs', projectId], (jobs) => {
    if (!jobs) return jobs
    const merged = mergeFactPage(jobs, page)
    unknown = merged.unknown
    return merged.jobs
  })

  const ids = new Set([...byJob.keys(), ...updatesByJob.keys()])
  for (const id of ids) {
    queryClient.setQueryData<PipelineJob>(['pipeline-job', id], (job) =>
      job
        ? withFactPage(job, {
            facts: byJob.get(id) ?? [],
            progress: updatesByJob.get(id) ?? [],
            cursor: page.cursor,
            projections: page.projections,
          })
        : job,
    )
  }
  return unknown
}

function withFacts(job: PipelineJob, add: PipelineFact[] | undefined): PipelineJob {
  if (!add?.length) return job
  return { ...job, facts: mergeFacts(job.facts, add) }
}

function mergeJobProgress(jobs: PipelineJob[], incoming: PipelineProgress[]): { jobs: PipelineJob[]; unknown: boolean } {
  const byJob = new Map<string, PipelineProgress[]>()
  for (const update of incoming) {
    const list = byJob.get(update.jobId) ?? []
    list.push(update)
    byJob.set(update.jobId, list)
  }
  const known = new Set(jobs.map((job) => job.id))
  let unknown = false
  for (const id of byJob.keys()) {
    if (!known.has(id)) unknown = true
  }
  return {
    unknown,
    jobs: jobs.map((job) => {
      const add = byJob.get(job.id)
      if (!add?.length) return job
      return { ...job, progress: mergeProgress(job.progress, add) }
    }),
  }
}
