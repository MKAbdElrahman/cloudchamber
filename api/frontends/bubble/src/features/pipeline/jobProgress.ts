import { jobPhase, projectedStatus, type JobFactStatus, type JobPhase } from './jobStatus'
import type { PipelineProgress } from './types'

/**
 * The five steps a worker reports. A missing row has not started. The page
 * does not infer the live step from file ids.
 */

export type JobStageId = 'source' | 'audio' | 'transcript' | 'keyFrames' | 'document'

export type JobStageState = 'done' | 'current' | 'upcoming' | 'failed' | 'canceled'

export type JobStageView = {
  id: JobStageId
  label: string
  state: JobStageState
  detail: string
  /** Fraction of the current audio step, when the worker has measured one. */
  progress?: number
}

export type JobProgress = {
  phase: JobPhase
  /** Index of the live stage, or the stage count when every stage is done. */
  activeStep: number
  completedCount: number
  stages: JobStageView[]
  headline: string
  /** Server failure text, when the job recorded one. */
  reason?: string
}

export type JobStepInput = {
  fact: string
  message?: string | null
  outcome?: string | null
}

export type JobProgressInput = {
  status: string
  dateCreated?: string
  fileName?: string
  audioFileId?: string | null
  transcriptFileId?: string | null
  keyFramesFileId?: string | null
  documentFileId?: string | null
  documentName?: string | null
  /** Application Failure message. Empty when the run closed as canceled. */
  failureMessage?: string | null
  facts?: (JobFactStatus & { message?: string | null; stage?: string | null; outcome?: string | null })[] | null
  /** Live reports. A newer update than the stage's event is the bar. */
  progress?: PipelineProgress[] | null
}

const STAGE_IDS: JobStageId[] = ['source', 'audio', 'transcript', 'keyFrames', 'document']

/** A progress stage is already a page stage. */
export function stageIdOf(stage: string | undefined): JobStageId | undefined {
  switch (stage) {
    case 'source':
    case 'audio':
    case 'transcript':
    case 'keyFrames':
    case 'document':
      return stage
    default:
      return undefined
  }
}

const LABELS: Record<JobStageId, string> = {
  source: 'Resolve source',
  audio: 'Extract audio',
  transcript: 'Transcribe',
  keyFrames: 'Extract key frames',
  document: 'Compose export',
}

function hasRef(ref: string | null | undefined): boolean {
  return Boolean(ref?.trim())
}

function detailFor(
  id: JobStageId,
  state: JobStageState,
  job: JobProgressInput,
  message?: string | null,
): string {
  switch (state) {
    case 'upcoming':
      return ''
    case 'done':
      if (id === 'keyFrames' && latestFacts(job).get(id)?.outcome === 'skipped') return 'No frames.'
      if (id === 'document' && hasRef(job.documentFileId)) {
        const pdf = job.documentName?.trim().toLowerCase().endsWith('.pdf')
        return pdf ? 'PDF' : 'Markdown'
      }
      return ''
    case 'canceled':
      return said(message, 'Canceled.')
    case 'failed':
      return said(message, 'Failed.')
    case 'current':
      return message?.trim() ?? ''
  }
}

function said(message: string | null | undefined, fallback: string): string {
  return message?.trim() || fallback
}

function latestFacts(job: JobProgressInput): Map<JobStageId, JobStepInput & { position: number }> {
  const byStage = new Map<JobStageId, JobStepInput & { position: number }>()
  for (const fact of job.facts ?? []) {
    const id = stageIdOf(fact.stage ?? undefined)
    if (!id) continue
    const prev = byStage.get(id)
    const position = fact.position ?? 0
    if (!prev || position >= prev.position) {
      byStage.set(id, {
        fact: fact.fact ?? '',
        message: fact.message,
        outcome: fact.outcome,
        position,
      })
    }
  }
  return byStage
}

function latestUpdates(job: JobProgressInput): Map<JobStageId, PipelineProgress & { position: number }> {
  const byStage = new Map<JobStageId, PipelineProgress & { position: number }>()
  for (const update of job.progress ?? []) {
    const id = stageIdOf(update.stage)
    if (!id) continue
    const position = update.position ?? 0
    const prev = byStage.get(id)
    if (!prev || position >= prev.position) {
      byStage.set(id, { ...update, position })
    }
  }
  return byStage
}

export type ConversionNotice = {
  status: 'error' | 'warning'
  title: string
  description: string
}

function headlineNote(message: string | null | undefined): string {
  return message?.trim() ?? ''
}

/** The persistent message for a job that stopped. A running job has none. */
export function conversionNotice(
  phase: JobPhase,
  stages: JobStageView[],
  failureMessage?: string | null,
): ConversionNotice | null {
  const stopped = stages.find(
    (stage) => stage.state === 'failed' || stage.state === 'canceled',
  )
  const name = stopped?.label ?? 'This step'
  switch (phase) {
    case 'failed': {
      const reason = headlineNote(failureMessage)
      return {
        status: 'error',
        title: 'Failed',
        description: reason ? `${name} failed. ${reason}` : `${name} failed.`,
      }
    }
    case 'canceled':
      if (!stopped) {
        const finished = stages.filter((stage) => stage.state === 'done')
        const after = finished[finished.length - 1]?.label
        return {
          status: 'warning',
          title: 'Canceled',
          description: after ? `Canceled after ${after}.` : 'Canceled.',
        }
      }
      return {
        status: 'warning',
        title: 'Canceled',
        description: `Canceled during ${name}.`,
      }
    case 'waiting':
    case 'running':
    case 'finished':
      return null
  }
}

function headlineFor(phase: JobPhase, stages: JobStageView[], failureMessage?: string | null): string {
  const notice = conversionNotice(phase, stages, failureMessage)
  if (notice) return notice.description
  const live = stages.find((stage) => stage.state === 'current')
  switch (phase) {
    case 'finished':
      return stages[stages.length - 1]?.detail ?? ''
    case 'running':
      if (!live) return 'Starting.'
      return live.detail
    case 'waiting':
      return 'Waiting.'
    case 'failed':
    case 'canceled':
      return ''
  }
}

function stageState(outcome: string): JobStageState {
  switch (outcome) {
    case 'working':
      return 'current'
    case 'done':
    case 'skipped':
      return 'done'
    case 'failed':
      return 'failed'
    case 'canceled':
      return 'canceled'
    default:
      return 'upcoming'
  }
}

export function progressFromJob(job: JobProgressInput): JobProgress {
  const phase = jobPhase(projectedStatus(job))

  const byStage = latestFacts(job)
  const byUpdate = latestUpdates(job)

  const stages: JobStageView[] = STAGE_IDS.map((id) => {
    const row = byStage.get(id)
    const update = byUpdate.get(id)
    const live = update != null && update.position > (row?.position ?? -1)
    const outcome = live ? 'working' : (row?.outcome ?? '')
    const state = stageState(outcome)
    const message = live ? update.headline : row?.message
    const measured = live ? update.fraction : null
    const known = state === 'current' && measured != null && measured >= 0 && measured <= 1
    return {
      id,
      label: LABELS[id],
      state,
      detail: detailFor(id, state, job, message),
      progress: known ? measured : undefined,
    }
  })

  const currentIndex = stages.findIndex((stage) => stage.state === 'current' || stage.state === 'failed' || stage.state === 'canceled')
  const reason = job.failureMessage?.trim() || undefined

  return {
    phase,
    activeStep: currentIndex >= 0 ? currentIndex : phase === 'finished' ? STAGE_IDS.length : 0,
    completedCount: stages.filter((stage) => stage.state === 'done').length,
    stages,
    headline: headlineFor(phase, stages, reason),
    reason,
  }
}
