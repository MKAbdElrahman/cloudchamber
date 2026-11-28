import { describe, expect, it } from 'vitest'

import { conversionNotice, progressFromJob as progress } from './jobProgress'
import type { Workflow, RunStep } from '@/shared/runs'

function step(key: string, state: RunStep['state'], extra: Partial<RunStep> = {}): RunStep {
  return {
    key,
    state,
    fraction: 0,
    measured: false,
    dateStarted: '2026-11-28T10:00:00Z',
    dateUpdated: '2026-11-28T10:00:01Z',
    ...extra,
  }
}

const source = 'media2doc.ResolveSource'
const audio = 'media2doc.ExtractAudio'
const transcript = 'media2doc.Transcribe'
const keyFrames = 'media2doc.ExtractKeyFrames'
const document = 'media2doc.ComposeDocument'

const workflow: Pick<Workflow, 'steps'> = {
  steps: [
    { key: source, label: 'Resolve source' },
    { key: audio, label: 'Extract audio' },
    { key: transcript, label: 'Transcribe' },
    { key: keyFrames, label: 'Extract key frames' },
    { key: document, label: 'Compose export' },
  ],
}

/** The conversion's progress with the workflow loaded, as a page has it. */
function progressFromJob(job: Parameters<typeof progress>[0]) {
  return progress(job, workflow)
}

describe('progressFromJob', () => {
  it('shows a run with no steps as waiting on the first step', () => {
    const progress = progressFromJob({ status: 'submitted', steps: [] })
    expect(progress.phase).toBe('waiting')
    expect(progress.activeStep).toBe(0)
    expect(progress.headline).toBe('Waiting.')
    expect(progress.stages.map((stage) => stage.state)).toEqual(['upcoming', 'upcoming', 'upcoming', 'upcoming', 'upcoming'])
  })

  it('shows the working step with its sentence and its measured fraction', () => {
    const progress = progressFromJob({
      status: 'running',
      steps: [
        step(source, 'done'),
        step(audio, 'working', { message: 'Writing the audio.', fraction: 0.62, measured: true }),
      ],
    })
    expect(progress.activeStep).toBe(1)
    expect(progress.completedCount).toBe(1)
    expect(progress.stages[1]).toMatchObject({ state: 'current', detail: 'Writing the audio.', progress: 0.62 })
    expect(progress.headline).toBe('Writing the audio.')
  })

  it('leaves the bar unmeasured when the worker has not measured the step', () => {
    const progress = progressFromJob({
      status: 'running',
      steps: [step(source, 'working', { message: 'Loading the recording.' })],
    })
    expect(progress.stages[0].progress).toBeUndefined()
    expect(progress.stages[0].detail).toBe('Loading the recording.')
  })

  it('points at the next step between two steps', () => {
    const progress = progressFromJob({ status: 'running', steps: [step(source, 'done'), step(audio, 'done')] })
    expect(progress.activeStep).toBe(2)
    expect(progress.headline).toBe('Starting.')
  })

  it('keeps how far a paused step had got', () => {
    const progress = progressFromJob({
      status: 'paused',
      steps: [step(source, 'done'), step(audio, 'done'), step(transcript, 'paused', { fraction: 0.4, measured: true })],
    })
    expect(progress.phase).toBe('paused')
    expect(progress.activeStep).toBe(2)
    expect(progress.stages[2]).toMatchObject({ state: 'paused', detail: 'Paused.', progress: 0.4 })
    expect(progress.headline).toBe('Paused during Transcribe.')
    expect(conversionNotice(progress.phase, progress.stages)).toBeNull()
  })

  it('names the failed step and the reason', () => {
    const progress = progressFromJob({
      status: 'failed',
      failureMessage: 'the recording is not in this project',
      steps: [step(source, 'failed', { message: 'the recording is not in this project' })],
    })
    expect(progress.stages[0].state).toBe('failed')
    expect(progress.reason).toBe('the recording is not in this project')
    expect(conversionNotice(progress.phase, progress.stages, progress.reason)).toEqual({
      status: 'error',
      title: 'Failed',
      description: 'Resolve source failed. the recording is not in this project',
    })
  })

  it('says where a canceled run stopped', () => {
    const during = progressFromJob({ status: 'canceled', steps: [step(source, 'done'), step(audio, 'canceled')] })
    expect(conversionNotice(during.phase, during.stages)?.description).toBe('Canceled during Extract audio.')

    const between = progressFromJob({ status: 'canceled', steps: [step(source, 'done')] })
    expect(conversionNotice(between.phase, between.stages)?.description).toBe('Canceled after Resolve source.')

    const before = progressFromJob({ status: 'canceled', steps: [] })
    expect(conversionNotice(before.phase, before.stages)?.description).toBe('Canceled.')
  })

  it('finishes every step and names the document format', () => {
    const progress = progressFromJob({
      status: 'completed',
      documentFileId: 'doc-1',
      documentName: 'week-03.md',
      steps: [step(source, 'done'), step(audio, 'done'), step(transcript, 'done'), step(keyFrames, 'done'), step(document, 'done')],
    })
    expect(progress.phase).toBe('finished')
    expect(progress.activeStep).toBe(5)
    expect(progress.completedCount).toBe(5)
    expect(progress.stages[4].detail).toBe('Markdown')
    expect(progress.headline).toBe('Markdown')
  })

  it('shows key frames as done with no frames for a recording with no picture', () => {
    const progress = progressFromJob({
      status: 'completed',
      documentFileId: 'doc-1',
      documentName: 'talk.pdf',
      steps: [step(source, 'done'), step(audio, 'done'), step(transcript, 'done'), step(document, 'done')],
    })
    expect(progress.stages[3]).toMatchObject({ state: 'done', detail: 'No frames.' })
    expect(progress.stages[4].detail).toBe('PDF')
    expect(progress.completedCount).toBe(5)
  })

  it('does not call key frames done while the run is still going', () => {
    const progress = progressFromJob({
      status: 'running',
      steps: [step(source, 'done'), step(audio, 'done'), step(transcript, 'done'), step(document, 'working')],
    })
    expect(progress.stages[3].state).toBe('upcoming')
    expect(progress.activeStep).toBe(4)
  })

  it('shows a step the workflow does not name after the others, under its key', () => {
    const result = progressFromJob({ status: 'running', steps: [step('media2doc.NewStep', 'working')] })
    expect(result.stages).toHaveLength(6)
    expect(result.stages[5]).toMatchObject({ key: 'media2doc.NewStep', label: 'media2doc.NewStep', state: 'current' })
  })

  it('shows the reported steps under their keys until the workflow has loaded', () => {
    const result = progress({ status: 'running', steps: [step(source, 'done'), step(audio, 'working')] })
    expect(result.stages.map((stage) => stage.label)).toEqual([source, audio])
    expect(result.activeStep).toBe(1)
  })
})
