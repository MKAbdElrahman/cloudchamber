import { describe, expect, it } from 'vitest'

import { factLines } from './factLog'
import type { PipelineFact, PipelineProgress } from './types'

function fact(partial: Partial<PipelineFact> & Pick<PipelineFact, 'position' | 'fact'>): PipelineFact {
  return {
    jobId: 'job',
    globalPosition: partial.position,
    ...partial,
  }
}

function update(partial: Partial<PipelineProgress> & Pick<PipelineProgress, 'position' | 'stage'>): PipelineProgress {
  return {
    jobId: 'job',
    globalPosition: partial.position,
    ...partial,
  }
}

describe('factLines', () => {
  it('keeps one row while the same sentence is updated', () => {
    const lines = factLines(
      'audio',
      [],
      [
        update({ position: 1, stage: 'audio', headline: 'Writing the audio.', fraction: 0.05, time: 't1' }),
        update({ position: 2, stage: 'audio', headline: 'Writing the audio.', fraction: 0.1, time: 't2' }),
      ],
    )
    expect(lines).toEqual([{ key: '1', label: 'Writing the audio.', time: 't2' }])
  })

  it('starts a new row when the sentence changes, and omits a finished step', () => {
    const lines = factLines(
      'audio',
      [
        fact({ position: 3, fact: 'audioPublished', stage: 'audio', outcome: 'done', time: 't3' }),
        fact({ position: 4, fact: 'keyFramesSkipped', stage: 'keyFrames', outcome: 'skipped', time: 't4' }),
        fact({ position: 5, fact: 'audioFailed', stage: 'audio', outcome: 'failed', message: 'Audio was not written.', time: 't5' }),
      ],
      [
        update({ position: 1, stage: 'audio', headline: 'Downloading the recording.', time: 't1' }),
        update({ position: 2, stage: 'audio', headline: 'Writing the audio.', fraction: 0.2, time: 't2' }),
      ],
    )
    expect(lines.map((line) => line.label)).toEqual([
      'Downloading the recording.',
      'Writing the audio.',
      'Audio was not written.',
    ])
  })

  it('ignores facts from other steps', () => {
    const lines = factLines(
      'transcript',
      [],
      [
        update({ position: 1, stage: 'audio', headline: 'Writing the audio.' }),
        update({ position: 2, stage: 'transcript', headline: 'Writing the transcript.', fraction: 0.35 }),
      ],
    )
    expect(lines).toEqual([{ key: '2', label: 'Writing the transcript.', time: undefined }])
  })
})
