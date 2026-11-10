import type { TokenColor } from '@astryxdesign/core/Token'

const ERROR_STATUSES = new Set(['failed', 'canceled', 'cancelled', 'timed_out', 'terminated'])

export function isTerminalJobStatus(status: string | undefined): boolean {
  if (!status) return false
  return status === 'completed' || ERROR_STATUSES.has(status)
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
