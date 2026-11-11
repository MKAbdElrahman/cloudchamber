import { sourceMediaKind } from '@/shared/media'

import { jobPhase, type JobPhase } from './jobStatus'

/**
 * The five activities Temporal runs. A step is done when the server names a
 * later activity or the job holds that step's file id. A canceled step that
 * already has its file id is done too: the cancel landed after it, and the
 * next step did not start. A running job with no activity name stays on the
 * first step that has not been recorded.
 */

export type JobStageId = 'source' | 'audio' | 'transcript' | 'keyframes' | 'document'

export type JobStageState = 'done' | 'current' | 'upcoming' | 'failed' | 'paused' | 'canceled'

export type JobStageView = {
  id: JobStageId
  label: string
  state: JobStageState
  detail: string
}

export type JobProgress = {
  phase: JobPhase
  /** Index of the live stage, or the stage count when every stage is done. */
  activeStep: number
  completedCount: number
  stages: JobStageView[]
  /** True while the step in motion is not backed by an artifact ref. */
  estimated: boolean
  headline: string
  /** Server failure text, when the job recorded one. */
  reason?: string
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
  /** Workflow failure text. Names the activity when the server recorded one. */
  errorMessage?: string | null
  /** Activity the open execution reported. Empty once the job is closed. */
  activity?: string | null
}

const STAGE_IDS: JobStageId[] = ['source', 'audio', 'transcript', 'keyframes', 'document']

const LABELS: Record<JobStageId, string> = {
  source: 'Resolve source',
  audio: 'Extract audio',
  transcript: 'Transcribe',
  keyframes: 'Extract key frames',
  document: 'Compose export',
}

function hasRef(ref: string | null | undefined): boolean {
  return Boolean(ref?.trim())
}

function provenFlags(job: JobProgressInput, named: number): boolean[] {
  const passed = (index: number) => named > index
  return [
    passed(0) || hasRef(job.audioFileId) || hasRef(job.transcriptFileId) || hasRef(job.keyFramesFileId) || hasRef(job.documentFileId),
    passed(1) || hasRef(job.audioFileId) || hasRef(job.transcriptFileId) || hasRef(job.keyFramesFileId) || hasRef(job.documentFileId),
    passed(2) || hasRef(job.transcriptFileId) || hasRef(job.keyFramesFileId) || hasRef(job.documentFileId),
    passed(3) || hasRef(job.keyFramesFileId) || hasRef(job.documentFileId),
    passed(4) || hasRef(job.documentFileId),
  ]
}

function detailFor(id: JobStageId, state: JobStageState, job: JobProgressInput): string {
  if (state === 'upcoming') return 'Waiting.'
  const audioOnly = sourceMediaKind(job.fileName ?? '') === 'audio'
  const documentName = job.documentName?.trim().toLowerCase() ?? ''
  const pdf = documentName.endsWith('.pdf')

  switch (id) {
    case 'source':
      switch (state) {
        case 'current':
          return 'The media worker is loading the recording and checking the file.'
        case 'paused':
          return 'Paused while the recording was being checked.'
        case 'canceled':
          return 'This step did not write.'
        case 'failed':
          return 'The recording could not be checked.'
        case 'done':
          return 'The recording is the source for this job.'
      }
      break
    case 'audio':
      switch (state) {
        case 'current':
          return 'The media worker is writing the audio track.'
        case 'paused':
          return 'Paused while the audio track was being written.'
        case 'canceled':
          return 'This step did not write.'
        case 'failed':
          return 'The audio track was not written.'
        case 'done':
          return 'The audio track is in this project.'
      }
      break
    case 'transcript':
      switch (state) {
        case 'current':
          return 'The ASR worker is writing the transcript.'
        case 'paused':
          return 'Paused while the transcript was being written.'
        case 'canceled':
          return 'This step did not write.'
        case 'failed':
          return 'The transcript was not written. The audio stays in the project.'
        case 'done':
          return 'The transcript is in this project.'
      }
      break
    case 'keyframes':
      switch (state) {
        case 'current':
          return audioOnly
            ? 'This recording is audio, so the frames worker records that there are no frames.'
            : 'The frames worker is writing a key frame.'
        case 'paused':
          return 'Paused while a key frame was being written.'
        case 'canceled':
          return 'This step did not write.'
        case 'failed':
          return 'A key frame was not written.'
        case 'done':
          return audioOnly
            ? 'No frames for this recording.'
            : 'A key frame is in this project.'
      }
      break
    case 'document':
      switch (state) {
        case 'current':
          return 'The render worker is writing one document into this project.'
        case 'paused':
          return 'Paused while the document was being written.'
        case 'canceled':
          return 'This step did not write.'
        case 'failed':
          return 'The document was not written.'
        case 'done':
          if (!hasRef(job.documentFileId)) return 'This conversion finished.'
          return pdf ? 'A PDF is in this project.' : 'A Markdown document is in this project.'
      }
      break
  }
}

export type ConversionNotice = {
  status: 'error' | 'warning'
  title: string
  description: string
}

const ACTIVITY_STEP: Record<string, JobStageId> = {
  'pipeline.ResolveSource': 'source',
  'pipeline.ExtractAudio': 'audio',
  'pipeline.Transcribe': 'transcript',
  'pipeline.ExtractKeyFrames': 'keyframes',
  'pipeline.ComposeExport': 'document',
}

function stepNamedIn(message: string | null | undefined): number {
  if (!message) return -1
  const found = Object.entries(ACTIVITY_STEP).find(([activity]) => message.includes(activity))
  if (!found) return -1
  return STAGE_IDS.indexOf(found[1])
}

function failureReason(message: string | null | undefined): string {
  if (!message) return ''
  if (message.includes('SourceNotFound')) return 'The recording was not found. '
  if (message.includes('InvalidInput')) return 'The file id was not accepted. '
  return ''
}

/** The persistent message for a job that stopped. A running job has none. */
export function conversionNotice(
  phase: JobPhase,
  stages: JobStageView[],
  errorMessage?: string | null,
): ConversionNotice | null {
  const stopped = stages.find(
    (stage) => stage.state === 'failed' || stage.state === 'canceled' || stage.state === 'paused',
  )
  const name = stopped?.label ?? 'This step'
  switch (phase) {
    case 'failed':
      return {
        status: 'error',
        title: 'Failed',
        description: `${name} stopped and was not run again. ${failureReason(errorMessage)}Files already in the project stay there.`,
      }
    case 'canceled':
      if (!stopped) {
        const finished = stages.filter((stage) => stage.state === 'done')
        const next = stages.find((stage) => stage.state === 'upcoming')
        const after = finished[finished.length - 1]?.label ?? 'This step'
        const nextSentence = next ? ` ${next.label} did not start.` : ''
        return {
          status: 'warning',
          title: 'Canceled',
          description: `Canceled after ${after}.${nextSentence} Files already in the project stay there.`,
        }
      }
      return {
        status: 'warning',
        title: 'Canceled',
        description: `Canceled during ${name}. That step did not write. Files already in the project stay there.`,
      }
    case 'paused':
      return {
        status: 'warning',
        title: 'Paused',
        description: `Paused on ${name}.`,
      }
    case 'waiting':
    case 'running':
    case 'finished':
      return null
  }
}

function headlineFor(phase: JobPhase, stages: JobStageView[], errorMessage?: string | null): string {
  const notice = conversionNotice(phase, stages, errorMessage)
  if (notice) return notice.description
  const live = stages.find((stage) => stage.state === 'current')
  switch (phase) {
    case 'finished':
      return stages[stages.length - 1]?.detail ?? ''
    case 'running':
      return live?.detail ?? ''
    case 'waiting':
      return 'This job is in the queue.'
    case 'paused':
    case 'failed':
    case 'canceled':
      return ''
  }
}

function stepWrote(job: JobProgressInput, id: JobStageId): boolean {
  switch (id) {
    case 'source':
      return false
    case 'audio':
      return hasRef(job.audioFileId)
    case 'transcript':
      return hasRef(job.transcriptFileId)
    case 'keyframes':
      return hasRef(job.keyFramesFileId)
    case 'document':
      return hasRef(job.documentFileId)
  }
}

function namedStep(job: JobProgressInput): number {
  const fromActivity = stepNamedIn(job.activity)
  if (fromActivity >= 0) return fromActivity
  return stepNamedIn(job.errorMessage)
}

export function progressFromJob(job: JobProgressInput): JobProgress {
  const named = namedStep(job)
  const flags = provenFlags(job, named)
  const phase: JobPhase = flags[flags.length - 1] ? 'finished' : jobPhase(job.status)
  const namedId = named >= 0 ? STAGE_IDS[named] : undefined
  const namedWrote = phase === 'canceled' && namedId !== undefined && stepWrote(job, namedId)

  let doneThrough = -1
  let currentIndex = -1
  let held: JobStageState | null = null

  if (phase === 'finished') {
    doneThrough = STAGE_IDS.length - 1
  } else if (phase === 'waiting') {
    currentIndex = 0
  } else if (namedWrote) {
    doneThrough = named
    currentIndex = named + 1
  } else if (named >= 0) {
    doneThrough = named - 1
    currentIndex = named
    if (phase === 'paused' || phase === 'failed' || phase === 'canceled') held = phase
  } else if (phase === 'paused' || phase === 'failed' || phase === 'canceled') {
    const stopAt = flags.indexOf(false)
    doneThrough = stopAt === -1 ? STAGE_IDS.length - 1 : stopAt - 1
    if (stopAt !== -1) {
      currentIndex = stopAt
      held = phase
    }
  } else {
    const firstMissing = flags.indexOf(false)
    doneThrough = firstMissing <= 0 ? -1 : firstMissing - 1
    if (firstMissing !== -1) currentIndex = firstMissing
  }

  const stages: JobStageView[] = STAGE_IDS.map((id, index) => {
    let state: JobStageState = 'upcoming'
    if (index <= doneThrough) state = 'done'
    else if (index === currentIndex && held) state = held
    else if (index === currentIndex && !namedWrote) state = 'current'
    return {
      id,
      label: LABELS[id],
      state,
      detail: detailFor(id, state, job),
    }
  })

  return {
    phase,
    activeStep: currentIndex >= 0 ? currentIndex : STAGE_IDS.length,
    completedCount: stages.filter((stage) => stage.state === 'done').length,
    stages,
    estimated: phase === 'running' && named < 0,
    headline: headlineFor(phase, stages, job.errorMessage),
    reason: job.errorMessage?.trim() || undefined,
  }
}
