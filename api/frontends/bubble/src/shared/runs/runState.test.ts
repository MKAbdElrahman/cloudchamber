import { describe, expect, it } from 'vitest'

import { activeStepIndex, allowedCommands, isRunMoving, pendingRequest, stepViews } from './runState'
import type { Run, RunStep } from './types'

type State = Pick<Run, 'spec' | 'status' | 'settled'>

function run(phase: Run['status']['phase'], spec: Partial<Run['spec']> = {}, status: Partial<Run['status']> = {}): State {
  const full: State = {
    settled: false,
    spec: { suspend: false, cancel: false, attempt: 1, resumeFrom: 'start', ...spec },
    status: { phase, observedGeneration: 1, attempt: 1, steps: [], ...status },
  }
  const closed = ['succeeded', 'failed', 'canceled', 'terminated'].includes(phase)
  full.settled = closed && full.status.attempt >= full.spec.attempt
  return full
}

function step(key: string, state: RunStep['state'], extra: Partial<RunStep> = {}): RunStep {
  return { key, state, fraction: 0, measured: false, dateStarted: '', dateUpdated: '', ...extra }
}

describe('allowedCommands', () => {
  it('offers pause and cancel while a run is running', () => {
    expect(allowedCommands(run('running'))).toEqual(['suspend', 'cancel'])
    expect(allowedCommands(run('pending'))).toEqual(['suspend', 'cancel'])
  })

  it('offers resume once a pause was asked for, whether or not it has landed', () => {
    expect(allowedCommands(run('running', { suspend: true }))).toEqual(['resume', 'cancel'])
    expect(allowedCommands(run('paused', { suspend: true }))).toEqual(['resume', 'cancel'])
  })

  it('offers nothing while a cancel is underway', () => {
    expect(allowedCommands(run('running', { cancel: true }))).toEqual([])
  })

  it('offers retry only for a run that did not succeed', () => {
    expect(allowedCommands(run('failed'))).toEqual(['retry', 'restart'])
    expect(allowedCommands(run('canceled'))).toEqual(['retry', 'restart'])
    expect(allowedCommands(run('succeeded'))).toEqual(['restart'])
  })

  it('treats a run that was asked to run again as open', () => {
    expect(allowedCommands(run('failed', { attempt: 2 }))).toEqual(['suspend', 'cancel'])
  })
})

describe('pendingRequest', () => {
  it('is the difference between what was asked and what was observed', () => {
    expect(pendingRequest(run('running'))).toBeNull()
    expect(pendingRequest(run('running', { suspend: true }))).toBe('pausing')
    expect(pendingRequest(run('paused', { suspend: true }))).toBeNull()
    expect(pendingRequest(run('paused'))).toBe('resuming')
    expect(pendingRequest(run('running', { cancel: true }))).toBe('canceling')
    expect(pendingRequest(run('failed', { attempt: 2 }))).toBe('restarting')
    expect(pendingRequest(run('failed'))).toBeNull()
  })
})

describe('isRunMoving', () => {
  it('stops reading a run once it has closed and nothing more was asked', () => {
    expect(isRunMoving(run('running'))).toBe(true)
    expect(isRunMoving(run('paused', { suspend: true }))).toBe(true)
    expect(isRunMoving(run('succeeded'))).toBe(false)
    expect(isRunMoving(run('failed', { attempt: 2 }))).toBe(true)
  })
})

describe('stepViews', () => {
  const workflow = { steps: [{ key: 'a', label: 'First' }, { key: 'b', label: 'Second' }, { key: 'c', label: 'Third' }] }

  it('orders and names the steps from the workflow, and marks unreported ones upcoming', () => {
    const views = stepViews([step('a', 'done'), step('b', 'working', { message: 'Halfway.', fraction: 0.5, measured: true })], workflow)
    expect(views).toEqual([
      { key: 'a', label: 'First', state: 'done', detail: '', progress: undefined },
      { key: 'b', label: 'Second', state: 'current', detail: 'Halfway.', progress: 0.5 },
      { key: 'c', label: 'Third', state: 'upcoming', detail: '', progress: undefined },
    ])
    expect(activeStepIndex(views, false)).toBe(1)
  })

  it('keeps the fraction of a paused step and says why a step stopped', () => {
    const views = stepViews([step('a', 'paused', { fraction: 0.3, measured: true }), step('b', 'failed', { message: 'No disk.' }), step('c', 'canceled')], workflow)
    expect(views.map((view) => [view.state, view.detail, view.progress])).toEqual([
      ['paused', 'Paused.', 0.3],
      ['failed', 'No disk.', undefined],
      ['canceled', 'Canceled.', undefined],
    ])
  })

  it('points past the last step once a run has finished', () => {
    const views = stepViews([step('a', 'done'), step('b', 'done'), step('c', 'done')], workflow)
    expect(activeStepIndex(views, true)).toBe(3)
  })

  it('points at the next step between two steps', () => {
    expect(activeStepIndex(stepViews([step('a', 'done')], workflow), false)).toBe(1)
  })

  it('works without a workflow, from the steps the run reported', () => {
    const views = stepViews([step('x', 'done'), step('y', 'working')], undefined)
    expect(views.map((view) => view.label)).toEqual(['x', 'y'])
  })
})
