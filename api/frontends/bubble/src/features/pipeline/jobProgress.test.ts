import { describe, expect, it } from 'vitest'

import { progressFromJob } from './jobProgress'

const started = '2026-11-10T12:00:00.000Z'

describe('progressFromJob', () => {
  it('starts a running job on the recording until the server names a step', () => {
    const progress = progressFromJob({ status: 'running', dateCreated: started, fileName: 'week-03.mp4' })
    expect(progress.stages.map((stage) => stage.state)).toEqual([
      'current',
      'upcoming',
      'upcoming',
      'upcoming',
      'upcoming',
    ])
    expect(progress.activeStep).toBe(0)
    expect(progress.estimated).toBe(true)
    expect(progress.headline).toContain('media worker')
  })

  it('follows the activity the server names', () => {
    const progress = progressFromJob(
      { status: 'running', dateCreated: started, activity: 'pipeline.Transcribe', audioFileId: 'audio-1' },
    )
    expect(progress.stages.map((stage) => stage.state)).toEqual([
      'done',
      'done',
      'current',
      'upcoming',
      'upcoming',
    ])
    expect(progress.headline).toContain('ASR worker')
    expect(progress.estimated).toBe(false)

    const exporting = progressFromJob(
      {
        status: 'running',
        activity: 'pipeline.ComposeExport',
        audioFileId: 'audio-1',
        transcriptFileId: 'transcript-1',
        keyFramesFileId: 'frames-1',
      },
    )
    expect(exporting.stages.map((stage) => stage.state)).toEqual([
      'done',
      'done',
      'done',
      'done',
      'current',
    ])
    expect(exporting.headline).toContain('render worker')
    expect(exporting.estimated).toBe(false)
  })

  it('holds a queued job on the recording', () => {
    const progress = progressFromJob({ status: 'queued', dateCreated: started })
    expect(progress.phase).toBe('waiting')
    expect(progress.stages[0]?.state).toBe('current')
    expect(progress.stages[0]?.detail).toContain('media worker')
    expect(progress.estimated).toBe(false)
    expect(progress.headline).toBe('This job is in the queue.')
  })

  it('marks every stage done once the document exists', () => {
    const progress = progressFromJob(
      {
        status: 'running',
        dateCreated: started,
        documentFileId: 'doc-1',
        documentName: 'document.md',
      },
    )
    expect(progress.stages.every((stage) => stage.state === 'done')).toBe(true)
    expect(progress.activeStep).toBe(5)
    expect(progress.estimated).toBe(false)
    expect(progress.headline).toBe('A Markdown document is in this project.')
  })

  it('names a PDF export when the document ref is a PDF', () => {
    const progress = progressFromJob(
      { status: 'completed', documentFileId: 'doc-pdf', documentName: 'week-03.pdf' },
    )
    expect(progress.headline).toBe('A PDF is in this project.')
  })

  it('stops on the first stage that never produced a ref', () => {
    const failed = progressFromJob(
      { status: 'failed', dateCreated: started, audioFileId: 'audio-1' },
    )
    expect(failed.stages.map((stage) => stage.state)).toEqual([
      'done',
      'done',
      'failed',
      'upcoming',
      'upcoming',
    ])
    expect(failed.stages[2]?.state).toBe('failed')
    expect(failed.headline).toBe('Transcribe stopped and was not run again. Files already in the project stay there.')
    expect(failed.stages[2]?.detail).toContain('audio stays')
    expect(failed.estimated).toBe(false)

    const stopped = progressFromJob({ status: 'canceled', dateCreated: started })
    expect(stopped.phase).toBe('canceled')
    expect(stopped.stages[0]?.state).toBe('canceled')
    expect(stopped.stages[0]?.detail).toBe('This step did not write.')
    expect(stopped.headline).toBe(
      'Canceled during Resolve source. That step did not write. Files already in the project stay there.',
    )
    expect(stopped.completedCount).toBe(0)

    const named = progressFromJob(
      { status: 'canceled', activity: 'pipeline.Transcribe', audioFileId: 'audio-1' },
    )
    expect(named.stages.map((stage) => stage.state)).toEqual([
      'done',
      'done',
      'canceled',
      'upcoming',
      'upcoming',
    ])
    expect(named.stages[2]?.detail).toBe('This step did not write.')
    expect(named.headline).toBe(
      'Canceled during Transcribe. That step did not write. Files already in the project stay there.',
    )

    const after = progressFromJob(
      {
        status: 'canceled',
        errorMessage: 'pipeline.Transcribe',
        audioFileId: 'audio-1',
        transcriptFileId: 'transcript-1',
      },
    )
    expect(after.stages.map((stage) => stage.state)).toEqual([
      'done',
      'done',
      'done',
      'upcoming',
      'upcoming',
    ])
    expect(after.activeStep).toBe(3)
    expect(after.headline).toBe(
      'Canceled after Transcribe. Extract key frames did not start. Files already in the project stay there.',
    )
  })

  it('holds a paused job on the next unfinished stage and does not walk the clock', () => {
    const progress = progressFromJob(
      { status: 'paused', dateCreated: started, audioFileId: 'audio-1' },
    )
    expect(progress.phase).toBe('paused')
    expect(progress.stages.map((stage) => stage.state)).toEqual([
      'done',
      'done',
      'paused',
      'upcoming',
      'upcoming',
    ])
    expect(progress.headline).toBe('Paused on Transcribe.')
    expect(progress.estimated).toBe(false)
  })

  it('stops on the activity named in the failure', () => {
    const failed = progressFromJob(
      {
        status: 'failed',
        dateCreated: started,
        errorMessage: 'activity error (type: pipeline.ResolveSource): load source (type: SourceNotFound, retryable: false)',
      },
    )
    expect(failed.stages.map((stage) => stage.state)).toEqual(['failed', 'upcoming', 'upcoming', 'upcoming', 'upcoming'])
    expect(failed.stages[0]?.label).toBe('Resolve source')
    expect(failed.headline).toBe(
      'Resolve source stopped and was not run again. The recording was not found. Files already in the project stay there.',
    )
  })

  it('does not claim frames for an audio recording', () => {
    const progress = progressFromJob(
      {
        status: 'completed',
        fileName: 'lecture.mp3',
        documentFileId: 'doc-audio',
        documentName: 'lecture.md',
      },
    )
    expect(progress.stages.find((stage) => stage.id === 'keyframes')?.detail).toContain('No frames')
  })
})
