import { activeStepIndex, stepViews, type Workflow, type RunStep, type StepView } from '@/shared/runs'

import { jobPhase, type JobPhase } from './jobStatus'

/**
 * A conversion's steps. The order and the names come from the workflow, and the
 * state of each step comes from the run. What this file adds is what only a
 * conversion knows: what a finished document is, and that a recording with no
 * picture has no frames.
 */

/** The step keys this workflow's page treats specially. They are the activity type names. */
const KEY_FRAMES_STEP = 'video.ExtractKeyFrames'
const DOCUMENT_STEP = 'video.ComposeDocument'

export type JobProgress = {
  phase: JobPhase
  /** Index of the live step, or the step count when every step is done. */
  activeStep: number
  completedCount: number
  stages: StepView[]
  headline: string
  /** Server failure text, when the run recorded one. */
  reason?: string
}

export type JobProgressInput = {
  status: string
  documentFileId?: string | null
  documentName?: string | null
  /** Failure message. Empty when the run closed as canceled. */
  failureMessage?: string | null
  steps?: RunStep[] | null
}

export type ConversionNotice = {
  status: 'error' | 'warning'
  title: string
  description: string
}

/** The persistent message for a job that stopped. A running job has none. */
export function conversionNotice(
  phase: JobPhase,
  stages: StepView[],
  failureMessage?: string | null,
): ConversionNotice | null {
  const stopped = stages.find((stage) => stage.state === 'failed' || stage.state === 'canceled')
  const name = stopped?.label ?? 'This step'
  switch (phase) {
    case 'failed': {
      const reason = failureMessage?.trim() ?? ''
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
    case 'paused':
    case 'finished':
      return null
  }
}

function headlineFor(phase: JobPhase, stages: StepView[], failureMessage?: string | null): string {
  const notice = conversionNotice(phase, stages, failureMessage)
  if (notice) return notice.description
  const live = stages.find((stage) => stage.state === 'current' || stage.state === 'paused')
  switch (phase) {
    case 'finished':
      return stages[stages.length - 1]?.detail ?? ''
    case 'running':
      return live ? live.detail : 'Starting.'
    case 'paused':
      return live ? `Paused during ${live.label}.` : 'Paused.'
    case 'waiting':
      return 'Waiting.'
    case 'failed':
    case 'canceled':
      return ''
  }
}

/** The format a finished document is in, named for the last step. */
function documentFormat(job: JobProgressInput): string {
  if (!job.documentFileId?.trim()) return ''
  return job.documentName?.trim().toLowerCase().endsWith('.pdf') ? 'PDF' : 'Markdown'
}

/** `workflow` is the video-to-document workflow. Until it has loaded, the steps the run reported are shown under their keys. */
export function progressFromJob(job: JobProgressInput, workflow?: Pick<Workflow, 'steps'>): JobProgress {
  const phase = jobPhase(job.status)

  const stages = stepViews(job.steps, workflow).map((stage): StepView => {
    // A recording with no picture has no frames to choose, so the run never schedules that step.
    if (stage.key === KEY_FRAMES_STEP && stage.state === 'upcoming' && phase === 'finished') {
      return { ...stage, state: 'done', detail: 'No frames.' }
    }
    if (stage.key === DOCUMENT_STEP && stage.state === 'done') {
      return { ...stage, detail: documentFormat(job) }
    }
    return stage
  })

  const reason = job.failureMessage?.trim() || undefined

  return {
    phase,
    activeStep: activeStepIndex(stages, phase === 'finished'),
    completedCount: stages.filter((stage) => stage.state === 'done').length,
    stages,
    headline: headlineFor(phase, stages, reason),
    reason,
  }
}
