import type { TokenColor } from '@astryxdesign/core/Token'

/** One status intent. The token hue and any status banner both read this. */
export type JobStatusIntent = 'success' | 'info' | 'warning' | 'error' | 'neutral'

const TOKEN_COLOR = {
  success: 'green',
  info: 'blue',
  warning: 'orange',
  error: 'red',
  neutral: 'gray',
} as const satisfies Record<JobStatusIntent, TokenColor>

export function jobStatusIntent(status: string | undefined): JobStatusIntent {
  switch (status) {
    case 'completed':
      return 'success'
    case 'running':
    case 'submitted':
      return 'info'
    case 'failed':
      return 'error'
    case 'canceled':
    case 'cancelled':
    case 'timed_out':
    case 'terminated':
      return 'warning'
    default:
      return 'neutral'
  }
}

export function isTerminalJobStatus(status: string | undefined): boolean {
  switch (jobStatusIntent(status)) {
    case 'success':
    case 'error':
    case 'warning':
      return true
    default:
      return false
  }
}

/** Status category color. Tokens label repeated states; Badge is for counts only. */
export function jobStatusTokenColor(status: string): TokenColor {
  return TOKEN_COLOR[jobStatusIntent(status)]
}

/** Banner status for a job outcome. Neutral (queued, unknown) stays informational. */
export function jobStatusBannerStatus(status: string): 'success' | 'info' | 'warning' | 'error' {
  switch (jobStatusIntent(status)) {
    case 'neutral':
      return 'info'
    case 'success':
      return 'success'
    case 'info':
      return 'info'
    case 'warning':
      return 'warning'
    case 'error':
      return 'error'
  }
}

/** Banner for a job's error text. A stopped run stays warning; a failure stays error. */
export function jobOutcomeBanner(status: string): { status: 'warning' | 'error'; title: string } {
  if (jobStatusIntent(status) === 'warning') {
    return { status: 'warning', title: 'Stopped' }
  }
  return { status: 'error', title: 'Error' }
}
