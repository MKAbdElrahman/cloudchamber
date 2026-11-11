import { sourceMediaKind } from '@/shared/media'

import { jobPhase, type JobPhase } from './jobStatus'

/**
 * Stand-in for a workflow query the API does not expose yet.
 * The job record is one status plus artifact refs. Temporal runs
 * ResolveSource, ExtractAudio, Transcribe, ExtractKeyFrames, and
 * ComposeExport on the workflow, media, speech, and render workers.
 * Until `GET /v1/pipeline/jobs/{id}/progress` reports each activity, a running
 * job walks these stages on a clock. A ref, or a finished job, replaces it.
 */
export const JOB_STAGE_MS = 2_000

export type JobStageId = 'input' | 'audio' | 'transcript' | 'keyframes' | 'compose' | 'export'

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
}

const STAGE_IDS: JobStageId[] = ['input', 'audio', 'transcript', 'keyframes', 'compose', 'export']

const LABELS: Record<JobStageId, string> = {
  input: 'Input',
  audio: 'Extract audio',
  transcript: 'Transcribe',
  keyframes: 'Extract key frames',
  compose: 'Compose document',
  export: 'Export',
}

function hasRef(ref: string | null | undefined): boolean {
  return Boolean(ref?.trim())
}

function provenFlags(job: JobProgressInput): boolean[] {
  const document = hasRef(job.documentFileId)
  return [
    true,
    hasRef(job.audioFileId),
    hasRef(job.transcriptFileId),
    hasRef(job.keyFramesFileId),
    document,
    document,
  ]
}

/** How far a running job may look, capped on the last stage until a ref or a finish. */
function clockCursor(job: JobProgressInput, now: number): number {
  const created = job.dateCreated ? Date.parse(job.dateCreated) : Number.NaN
  const elapsed = Number.isFinite(created) ? Math.max(0, now - created) : 0
  return Math.min(1 + Math.floor(elapsed / JOB_STAGE_MS), STAGE_IDS.length - 1)
}

function detailFor(id: JobStageId, state: JobStageState, job: JobProgressInput): string {
  if (state === 'upcoming') return 'Waiting.'
  const audioOnly = sourceMediaKind(job.fileName ?? '') === 'audio'
  const documentName = job.documentName?.trim().toLowerCase() ?? ''
  const pdf = documentName.endsWith('.pdf')

  switch (id) {
    case 'input':
      switch (state) {
        case 'current':
          return 'The workflow worker has the recording.'
        case 'paused':
          return 'Paused before the workflow took the recording.'
        case 'canceled':
          return 'Canceled before the workflow took the recording.'
        case 'failed':
          return 'The recording could not be accepted.'
        case 'done':
          return 'One recording is in this job.'
      }
      break
    case 'audio':
      switch (state) {
        case 'current':
          return 'The media worker is checking the file and pulling the audio track.'
        case 'paused':
          return 'Paused while the audio track was coming out.'
        case 'canceled':
          return 'Canceled while the audio track was coming out.'
        case 'failed':
          return 'The audio track could not be extracted.'
        case 'done':
          return 'The audio track is out.'
      }
      break
    case 'transcript':
      switch (state) {
        case 'current':
          return 'The speech worker is writing a timestamped transcript.'
        case 'paused':
          return 'Paused while the transcript was being written.'
        case 'canceled':
          return 'Canceled during transcription.'
        case 'failed':
          return 'The transcript could not be written.'
        case 'done':
          return 'The transcript is timestamped.'
      }
      break
    case 'keyframes':
      switch (state) {
        case 'current':
          return audioOnly
            ? 'This recording is audio, so the media worker records that there are no frames.'
            : 'The media worker is choosing frames and lining them up with the transcript.'
        case 'paused':
          return 'Paused while frames were being chosen.'
        case 'canceled':
          return 'Canceled while frames were being chosen.'
        case 'failed':
          return 'Key frames could not be chosen.'
        case 'done':
          return audioOnly
            ? 'No frames to place. The transcript carries this recording.'
            : 'Key frames line up with the transcript.'
      }
      break
    case 'compose':
      switch (state) {
        case 'current':
          return audioOnly
            ? 'The render worker is organizing the transcript into sections.'
            : 'The render worker is organizing the transcript and frames into sections.'
        case 'paused':
          return 'Paused while the document was being sectioned.'
        case 'canceled':
          return 'Canceled while the document was being sectioned.'
        case 'failed':
          return 'The document could not be composed.'
        case 'done':
          return 'The document has sections and timestamps.'
      }
      break
    case 'export':
      switch (state) {
        case 'current':
          return 'The render worker is writing one document into this project.'
        case 'paused':
          return 'Paused while the document was being written.'
        case 'canceled':
          return 'Canceled while the document was being written.'
        case 'failed':
          return 'The document could not be written.'
        case 'done':
          if (!hasRef(job.documentFileId)) return 'This conversion finished.'
          return pdf ? 'A PDF is in this project.' : 'A Markdown document is in this project.'
      }
      break
  }
}

function headlineFor(phase: JobPhase, stages: JobStageView[]): string {
  const live = stages.find(
    (stage) =>
      stage.state === 'current' ||
      stage.state === 'failed' ||
      stage.state === 'paused' ||
      stage.state === 'canceled',
  )
  switch (phase) {
    case 'finished':
      return stages[stages.length - 1]?.detail ?? ''
    case 'running':
      return live?.detail ?? ''
    case 'waiting':
      return 'This job is in the queue.'
    case 'paused':
      return 'Paused on this stage.'
    case 'failed':
      return 'It stopped on this stage.'
    case 'canceled':
      return 'Canceled on this stage.'
  }
}

export function progressFromJob(job: JobProgressInput, now: number): JobProgress {
  const flags = provenFlags(job)
  const phase: JobPhase = flags[flags.length - 1] ? 'finished' : jobPhase(job.status)

  let doneThrough = -1
  let currentIndex = -1
  let held: JobStageState | null = null

  if (phase === 'finished') {
    doneThrough = STAGE_IDS.length - 1
  } else if (phase === 'waiting') {
    currentIndex = 0
  } else if (phase === 'paused' || phase === 'failed' || phase === 'canceled') {
    const firstMissing = flags.indexOf(false)
    doneThrough = firstMissing === -1 ? STAGE_IDS.length - 1 : firstMissing - 1
    if (firstMissing !== -1) {
      currentIndex = firstMissing
      held = phase
    }
  } else {
    const clock = clockCursor(job, now)
    let cursor = 0
    while (cursor < STAGE_IDS.length && (flags[cursor] || cursor < clock)) {
      if (cursor === STAGE_IDS.length - 1 && !flags[cursor]) break
      cursor += 1
    }
    doneThrough = cursor - 1
    if (cursor < STAGE_IDS.length) currentIndex = cursor
  }

  const stages: JobStageView[] = STAGE_IDS.map((id, index) => {
    let state: JobStageState = 'upcoming'
    if (index <= doneThrough) state = 'done'
    else if (index === currentIndex && held) state = held
    else if (index === currentIndex) state = 'current'
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
    estimated: phase === 'running' && currentIndex >= 0 && !flags[currentIndex],
    headline: headlineFor(phase, stages),
  }
}
