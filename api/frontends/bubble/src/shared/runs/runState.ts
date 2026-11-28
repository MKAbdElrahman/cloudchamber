import type { Workflow, Run, RunCommand, RunPhase, RunStep } from './types'

/** A run whose execution has closed. A new attempt may still have been asked for. */
export function isClosedPhase(phase: RunPhase): boolean {
  switch (phase) {
    case 'succeeded':
    case 'failed':
    case 'canceled':
    case 'terminated':
      return true
    case 'pending':
    case 'running':
    case 'paused':
      return false
  }
}

/**
 * A run is moving while its execution is open, or while the platform has not
 * yet acted on the last request. A moving run is worth reading again.
 */
export function isRunMoving(run: Pick<Run, 'settled' | 'status'>): boolean {
  return !run.settled || !isClosedPhase(run.status.phase)
}

/** What was asked for and has not happened yet. */
export type RunRequest = 'pausing' | 'resuming' | 'canceling' | 'restarting'

/**
 * The request the platform is still acting on. It is the difference between
 * what the run is asked to do and what was last observed.
 */
export function pendingRequest(run: Pick<Run, 'spec' | 'status' | 'settled'>): RunRequest | null {
  const { spec, status } = run
  if (run.settled) return null
  if (spec.attempt > status.attempt && status.attempt > 0) return spec.cancel ? 'canceling' : 'restarting'
  if (spec.cancel) return 'canceling'
  if (spec.suspend && status.phase === 'running') return 'pausing'
  if (!spec.suspend && status.phase === 'paused') return 'resuming'
  return null
}

/**
 * The commands that apply to a run now. It mirrors the rules the service
 * enforces, so a button that is shown is one the service will accept.
 */
export function allowedCommands(run: Pick<Run, 'spec' | 'status' | 'settled'>): RunCommand[] {
  const { spec, status } = run
  if (run.settled) {
    return status.phase === 'succeeded' ? ['restart'] : ['retry', 'restart']
  }
  if (spec.cancel) return []
  return spec.suspend ? ['resume', 'cancel'] : ['suspend', 'cancel']
}

export type StepState = 'done' | 'current' | 'paused' | 'upcoming' | 'failed' | 'canceled'

/** One step as a page shows it. */
export type StepView = {
  key: string
  label: string
  state: StepState
  /** The sentence for the step: what it is doing, or why it stopped. */
  detail: string
  /** Fraction of the step, when the worker has measured one. */
  progress?: number
}

function stepState(step: RunStep | undefined): StepState {
  switch (step?.state) {
    case 'working':
      return 'current'
    case 'paused':
      return 'paused'
    case 'done':
    case 'skipped':
      return 'done'
    case 'failed':
      return 'failed'
    case 'canceled':
      return 'canceled'
    case undefined:
      return 'upcoming'
  }
}

function stepDetail(state: StepState, step: RunStep | undefined): string {
  const message = step?.message?.trim() ?? ''
  switch (state) {
    case 'upcoming':
    case 'done':
      return ''
    case 'canceled':
      return message || 'Canceled.'
    case 'failed':
      return message || 'Failed.'
    case 'paused':
      return message || 'Paused.'
    case 'current':
      return message
  }
}

/**
 * The steps of a run in the order the workflow runs them. A step the workflow names
 * and the run has not reported is upcoming. A step the run reported and the
 * workflow does not name is shown after the others, under its key.
 */
export function stepViews(steps: RunStep[] | null | undefined, workflow: Pick<Workflow, 'steps'> | undefined): StepView[] {
  const reported = new Map((steps ?? []).map((step) => [step.key, step]))
  const catalog = workflow?.steps ?? []
  const known = new Set(catalog.map((step) => step.key))
  const ordered = [
    ...catalog,
    ...(steps ?? []).filter((step) => !known.has(step.key)).map((step) => ({ key: step.key, label: step.key })),
  ]

  return ordered.map(({ key, label }) => {
    const step = reported.get(key)
    const state = stepState(step)
    const live = state === 'current' || state === 'paused'
    return {
      key,
      label,
      state,
      detail: stepDetail(state, step),
      progress: live && step?.measured ? step.fraction : undefined,
    }
  })
}

/** Index of the step to point at: the one that is live or stopped, else the next one to start. */
export function activeStepIndex(views: StepView[], closedWell: boolean): number {
  const live = views.findIndex((view) => view.state !== 'done' && view.state !== 'upcoming')
  if (live >= 0) return live
  const next = views.findIndex((view) => view.state === 'upcoming')
  if (closedWell || next < 0) return views.length
  return next
}
