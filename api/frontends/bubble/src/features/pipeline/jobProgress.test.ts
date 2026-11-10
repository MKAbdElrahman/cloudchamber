import { describe, expect, it } from 'vitest'

import { progressFromJob } from './jobProgress'

const started = '2026-11-10T12:00:00.000Z'
const at = (ms: number) => Date.parse(started) + ms

describe('progressFromJob', () => {
  it('starts a running job on audio extraction after the recording is in', () => {
    const progress = progressFromJob({ status: 'running', dateCreated: started, sourceRef: 'week-03.mp4' }, at(0))
    expect(progress.stages.map((stage) => stage.state)).toEqual([
      'done',
      'current',
      'upcoming',
      'upcoming',
      'upcoming',
      'upcoming',
    ])
    expect(progress.activeStep).toBe(1)
    expect(progress.estimated).toBe(true)
    expect(progress.headline).toContain('media worker')
  })

  it('walks the workers while the job is still running', () => {
    const progress = progressFromJob({ status: 'running', dateCreated: started }, at(2_000))
    expect(progress.stages[2]?.state).toBe('current')
    expect(progress.headline).toContain('speech worker')

    const exporting = progressFromJob({ status: 'running', dateCreated: started }, at(10_000))
    expect(exporting.stages.map((stage) => stage.state)).toEqual([
      'done',
      'done',
      'done',
      'done',
      'done',
      'current',
    ])
    expect(exporting.headline).toContain('render worker')
    expect(exporting.estimated).toBe(true)
  })

  it('holds a queued job on the recording', () => {
    const progress = progressFromJob({ status: 'queued', dateCreated: started }, at(60_000))
    expect(progress.phase).toBe('waiting')
    expect(progress.stages[0]?.state).toBe('current')
    expect(progress.stages[0]?.detail).toContain('workflow worker')
    expect(progress.estimated).toBe(false)
    expect(progress.headline).toBe('This job is in the queue.')
  })

  it('marks every stage done once the document exists', () => {
    const progress = progressFromJob(
      {
        status: 'running',
        dateCreated: started,
        documentRef: 's3://cloudchamber-artifacts/jobs/1/exports/document.md',
      },
      at(0),
    )
    expect(progress.stages.every((stage) => stage.state === 'done')).toBe(true)
    expect(progress.activeStep).toBe(6)
    expect(progress.estimated).toBe(false)
    expect(progress.headline).toBe('A Markdown document is in this project.')
  })

  it('names a PDF export when the document ref is a PDF', () => {
    const progress = progressFromJob(
      { status: 'completed', documentRef: 's3://bucket/exports/week-03.pdf' },
      at(0),
    )
    expect(progress.headline).toBe('A PDF is in this project.')
  })

  it('stops on the first stage that never produced a ref', () => {
    const failed = progressFromJob(
      { status: 'failed', dateCreated: started, audioRef: 's3://bucket/audio.wav' },
      at(0),
    )
    expect(failed.stages.map((stage) => stage.state)).toEqual([
      'done',
      'done',
      'failed',
      'upcoming',
      'upcoming',
      'upcoming',
    ])
    expect(failed.stages[2]?.state).toBe('failed')
    expect(failed.headline).toBe('It stopped on this stage.')
    expect(failed.estimated).toBe(false)

    const stopped = progressFromJob({ status: 'canceled', dateCreated: started }, at(60_000))
    expect(stopped.phase).toBe('canceled')
    expect(stopped.stages[1]?.state).toBe('canceled')
    expect(stopped.stages[1]?.detail).toBe('Canceled while the audio track was coming out.')
    expect(stopped.headline).toBe('Canceled on this stage.')
    expect(stopped.completedCount).toBe(1)
  })

  it('holds a paused job on the next unfinished stage and does not walk the clock', () => {
    const progress = progressFromJob(
      { status: 'paused', dateCreated: started, audioRef: 's3://bucket/audio.wav' },
      at(60_000),
    )
    expect(progress.phase).toBe('paused')
    expect(progress.stages.map((stage) => stage.state)).toEqual([
      'done',
      'done',
      'paused',
      'upcoming',
      'upcoming',
      'upcoming',
    ])
    expect(progress.headline).toBe('Paused on this stage.')
    expect(progress.estimated).toBe(false)
  })

  it('does not claim frames for an audio recording', () => {
    const progress = progressFromJob(
      {
        status: 'completed',
        sourceRef: 's3://bucket/uploads/lecture.mp3',
        documentRef: 's3://bucket/exports/document.md',
      },
      at(0),
    )
    expect(progress.stages.find((stage) => stage.id === 'keyframes')?.detail).toContain('No frames')
  })
})
