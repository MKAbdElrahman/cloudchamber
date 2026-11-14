import { describe, expect, it } from 'vitest'

import { progressFromJob } from './jobProgress'

const started = '2026-11-10T12:00:00.000Z'

const SENTENCE: Record<string, { stage: string; outcome: string }> = {
  sourceResolved: { stage: 'source', outcome: 'done' },
  sourceFailed: { stage: 'source', outcome: 'failed' },
  sourceCanceled: { stage: 'source', outcome: 'canceled' },
  audioPublished: { stage: 'audio', outcome: 'done' },
  audioFailed: { stage: 'audio', outcome: 'failed' },
  audioCanceled: { stage: 'audio', outcome: 'canceled' },
  transcriptPublished: { stage: 'transcript', outcome: 'done' },
  transcriptFailed: { stage: 'transcript', outcome: 'failed' },
  transcriptCanceled: { stage: 'transcript', outcome: 'canceled' },
  keyFramesPublished: { stage: 'keyFrames', outcome: 'done' },
  keyFramesFailed: { stage: 'keyFrames', outcome: 'failed' },
  keyFramesCanceled: { stage: 'keyFrames', outcome: 'canceled' },
  keyFramesSkipped: { stage: 'keyFrames', outcome: 'skipped' },
  documentPublished: { stage: 'document', outcome: 'done' },
  documentFailed: { stage: 'document', outcome: 'failed' },
  documentCanceled: { stage: 'document', outcome: 'canceled' },
}

function facts(rows: { fact: string; message?: string; progress?: number }[]) {
  const facts: { fact: string; message?: string; position: number; globalPosition: number }[] = []
  const progress: { jobId: string; stage: string; fraction?: number; headline?: string; position: number; globalPosition: number }[] = []
  rows.forEach((row, index) => {
    const position = index + 1
    const stage = workingStage(row.fact)
    if (stage) {
      progress.push({
        jobId: 'job',
        stage,
        fraction: row.progress,
        headline: row.message,
        position,
        globalPosition: position,
      })
      return
    }
    facts.push({
      fact: row.fact,
      message: row.message,
      stage: SENTENCE[row.fact]?.stage,
      outcome: SENTENCE[row.fact]?.outcome,
      position,
      globalPosition: position,
    })
  })
  return { facts, progress }
}

function workingStage(fact: string): string | undefined {
  if (!fact.endsWith('Working')) return undefined
  if (fact.startsWith('keyFrames')) return 'keyFrames'
  if (fact.startsWith('source')) return 'source'
  if (fact.startsWith('audio')) return 'audio'
  if (fact.startsWith('transcript')) return 'transcript'
  if (fact.startsWith('document')) return 'document'
  return undefined
}

describe('progressFromJob', () => {
  it('waits until a worker reports a step', () => {
    const waiting = progressFromJob({ status: 'submitted', dateCreated: started })
    expect(waiting.phase).toBe('waiting')
    expect(waiting.stages.every((stage) => stage.state === 'upcoming')).toBe(true)
    expect(waiting.headline).toBe('Waiting.')

    const starting = progressFromJob({ status: 'running', dateCreated: started, fileName: 'week-03.mp4' })
    expect(starting.headline).toBe('Starting.')
    expect(starting.stages.every((stage) => stage.state === 'upcoming')).toBe(true)
  })

  it('shows the step the worker marked running', () => {
    const progress = progressFromJob({
      status: 'running',
      dateCreated: started,
      ...facts([
        { fact: 'runStarted' },
        { fact: 'sourceResolved' },
        { fact: 'audioWorking', progress: 0.42 },
      ]),
    })
    expect(progress.stages.map((stage) => stage.state)).toEqual([
      'done',
      'current',
      'upcoming',
      'upcoming',
      'upcoming',
    ])
    expect(progress.headline).toBe('')
    expect(progress.stages.find((stage) => stage.id === 'audio')?.progress).toBe(0.42)
    expect(progress.activeStep).toBe(1)
  })

  it('shows a measured step on any worker', () => {
    const progress = progressFromJob({
      status: 'running',
      ...facts([
        { fact: 'runStarted' },
        { fact: 'sourceResolved' },
        { fact: 'audioPublished' },
        { fact: 'transcriptWorking', progress: 0.35, message: 'Writing the transcript.' },
      ]),
    })
    expect(progress.headline).toBe('Writing the transcript.')
    expect(progress.stages.find((stage) => stage.id === 'transcript')?.progress).toBe(0.35)
  })

  it('shows the audio sentence the worker sent', () => {
    const progress = progressFromJob({
      status: 'running',
      ...facts([
        { fact: 'runStarted' },
        { fact: 'audioWorking', progress: 0.4, message: 'Downloading the recording.' },
      ]),
    })
    expect(progress.headline).toBe('Downloading the recording.')
    expect(progress.stages.find((stage) => stage.id === 'audio')?.progress).toBe(0.4)
    expect(progress.reason).toBeUndefined()
  })

  it('keeps a live audio sentence when the run has failed', () => {
    const progress = progressFromJob({
      status: 'failed',
      ...facts([
        { fact: 'audioWorking', progress: 0.4, message: 'Downloading the recording.' },
        { fact: 'runFailed' },
      ]),
    })
    expect(progress.phase).toBe('failed')
    expect(progress.stages.find((stage) => stage.id === 'audio')?.state).toBe('current')
    expect(progress.stages.find((stage) => stage.id === 'audio')?.detail).toBe('Downloading the recording.')
    expect(progress.headline).toBe('This step failed.')
  })

  it('holds a queued job as waiting', () => {
    const progress = progressFromJob({ status: 'queued', dateCreated: started })
    expect(progress.phase).toBe('waiting')
    expect(progress.headline).toBe('Waiting.')
  })

  it('marks every stage done once the job is completed', () => {
    const progress = progressFromJob({
      status: 'completed',
      dateCreated: started,
      documentFileId: 'doc-1',
      documentName: 'document.md',
      ...facts([
        { fact: 'sourceResolved' },
        { fact: 'audioPublished' },
        { fact: 'transcriptPublished' },
        { fact: 'keyFramesPublished' },
        { fact: 'documentPublished' },
        { fact: 'runCompleted' },
      ]),
    })
    expect(progress.stages.every((stage) => stage.state === 'done')).toBe(true)
    expect(progress.activeStep).toBe(5)
    expect(progress.headline).toBe('Markdown')
  })

  it('names a PDF export when the document ref is a PDF', () => {
    const progress = progressFromJob({
      status: 'completed',
      documentFileId: 'doc-pdf',
      documentName: 'week-03.pdf',
      ...facts([{ fact: 'documentPublished' }, { fact: 'runCompleted' }]),
    })
    expect(progress.headline).toBe('PDF')
    expect(progress.stages.find((stage) => stage.id === 'source')?.state).toBe('upcoming')
  })

  it('stops on the step the worker marked failed or canceled', () => {
    const failed = progressFromJob({
      status: 'failed',
      dateCreated: started,
      failureMessage: 'load source',
      ...facts([
        { fact: 'sourceResolved' },
        { fact: 'audioPublished' },
        { fact: 'transcriptFailed', message: 'load source' },
        { fact: 'runFailed', message: 'load source' },
      ]),
    })
    expect(failed.stages.map((stage) => stage.state)).toEqual([
      'done',
      'done',
      'failed',
      'upcoming',
      'upcoming',
    ])
    expect(failed.stages.find((stage) => stage.id === 'transcript')?.detail).toBe('load source')
    expect(failed.headline).toBe('Transcribe failed. load source')

    const stopped = progressFromJob({ status: 'canceled', dateCreated: started })
    expect(stopped.phase).toBe('canceled')
    expect(stopped.headline).toBe('Canceled.')

    const during = progressFromJob({
      status: 'canceled',
      ...facts([
        { fact: 'sourceResolved' },
        { fact: 'audioPublished' },
        { fact: 'transcriptCanceled' },
        { fact: 'runCanceled' },
      ]),
    })
    expect(during.stages[2]?.state).toBe('canceled')
    expect(during.headline).toBe('Canceled during Transcribe.')

    const after = progressFromJob({
      status: 'canceled',
      ...facts([
        { fact: 'sourceResolved' },
        { fact: 'audioPublished' },
        { fact: 'transcriptPublished' },
        { fact: 'runCanceled' },
      ]),
    })
    expect(after.stages.map((stage) => stage.state)).toEqual([
      'done',
      'done',
      'done',
      'upcoming',
      'upcoming',
    ])
    expect(after.headline).toBe('Canceled after Transcribe.')
  })

  it('shows a skipped frames step as done', () => {
    const progress = progressFromJob({
      status: 'running',
      fileName: 'lecture.mp3',
      ...facts([
        { fact: 'runStarted' },
        { fact: 'sourceResolved' },
        { fact: 'audioPublished' },
        { fact: 'transcriptPublished' },
        { fact: 'keyFramesSkipped' },
        { fact: 'documentWorking', message: 'Composing the export.' },
      ]),
    })
    expect(progress.stages.find((stage) => stage.id === 'keyFrames')?.state).toBe('done')
    expect(progress.stages.find((stage) => stage.id === 'keyFrames')?.detail).toBe('No frames.')
    expect(progress.headline).toBe('Composing the export.')
  })

  it('reopens a step when a later fact is running', () => {
    const progress = progressFromJob({
      status: 'running',
      ...facts([
        { fact: 'runStarted' },
        { fact: 'audioFailed' },
        { fact: 'audioWorking', progress: 0.1 },
      ]),
    })
    expect(progress.phase).toBe('running')
    expect(progress.stages.find((stage) => stage.id === 'audio')?.state).toBe('current')
  })

  it('keeps a live step when the run fact is newer and the step has no sentence', () => {
    const progress = progressFromJob({
      status: 'failed',
      failureMessage: 'source missing',
      ...facts([
        { fact: 'sourceWorking' },
        { fact: 'runFailed', message: 'source missing' },
      ]),
    })
    expect(progress.phase).toBe('failed')
    expect(progress.stages.find((stage) => stage.id === 'source')?.state).toBe('current')
    expect(progress.headline).toBe('This step failed. source missing')
  })
})
