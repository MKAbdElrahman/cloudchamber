import type { BadgeProps } from '@astryxdesign/core/Badge'
import type { StepStatus } from '@astryxdesign/core/Stepper'

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

/** Canonical job status color (Badge + Banner). */
export function jobStatusBadgeVariant(status: string): NonNullable<BadgeProps['variant']> {
  switch (status) {
    case 'completed':
      return 'success'
    case 'running':
    case 'submitted':
      return 'info'
    default:
      if (ERROR_STATUSES.has(status)) return 'error'
      return 'neutral'
  }
}
