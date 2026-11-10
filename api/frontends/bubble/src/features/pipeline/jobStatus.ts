import type { StepStatus } from '@astryxdesign/core/Stepper'
import type { TokenColor } from '@astryxdesign/core/Token'

const ERROR_STATUSES = new Set(['failed', 'canceled', 'cancelled', 'timed_out', 'terminated'])

export function isTerminalJobStatus(status: string | undefined): boolean {
  if (!status) return false
  return status === 'completed' || ERROR_STATUSES.has(status)
}

export function isJobSuccess(status: string | undefined): boolean {
  return status === 'completed'
}

/** Stepper semantic status — terminal outcomes only; in-progress uses active step styling. */
export function jobStepStatus(status: string | undefined): StepStatus | undefined {
  if (!status) return undefined
  if (isJobSuccess(status)) return 'success'
  if (ERROR_STATUSES.has(status)) return 'error'
  return undefined
}

/** Status category color. Tokens label repeated states; Badge is for counts only. */
export function jobStatusTokenColor(status: string): TokenColor {
  switch (status) {
    case 'completed':
      return 'green'
    case 'running':
    case 'submitted':
      return 'blue'
    default:
      return ERROR_STATUSES.has(status) ? 'red' : 'gray'
  }
}
