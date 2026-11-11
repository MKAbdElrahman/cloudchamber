import type { TokenColor } from '@astryxdesign/core/Token'

/** One status intent. The token hue and any status banner both read this. */
export type JobStatusIntent = 'success' | 'info' | 'warning' | 'error' | 'neutral'

/** Where a job sits. The page stays the same; this is what changes on it. */
export type JobPhase = 'waiting' | 'running' | 'finished' | 'failed' | 'canceled'

const TOKEN_COLOR = {
  success: 'green',
  info: 'blue',
  warning: 'orange',
  error: 'red',
  neutral: 'gray',
} as const satisfies Record<JobStatusIntent, TokenColor>

export function jobPhase(status: string | undefined): JobPhase {
  switch (status) {
    case 'completed':
      return 'finished'
    case 'running':
    case 'submitted':
      return 'running'
    case 'failed':
      return 'failed'
    case 'canceled':
    case 'cancelled':
      return 'canceled'
    default:
      return 'waiting'
  }
}

/** Words on the status token. One label for each way a job can sit. */
export function jobPhaseLabel(status: string | undefined): string {
  switch (status) {
    case 'uploading':
      return 'Uploading'
    case 'cancelled':
      return 'Canceled'
    default:
      switch (jobPhase(status)) {
        case 'waiting':
          return 'Waiting'
        case 'running':
          return 'Running'
        case 'finished':
          return 'Finished'
        case 'failed':
          return 'Failed'
        case 'canceled':
          return 'Canceled'
      }
  }
}

export function jobStatusIntent(status: string | undefined): JobStatusIntent {
  if (status === 'uploading') return 'info'
  switch (jobPhase(status)) {
    case 'finished':
      return 'success'
    case 'running':
      return 'info'
    case 'failed':
      return 'error'
    case 'canceled':
      return 'warning'
    case 'waiting':
      return 'neutral'
  }
}

export function isTerminalJobStatus(status: string | undefined): boolean {
  switch (jobPhase(status)) {
    case 'finished':
    case 'failed':
    case 'canceled':
      return true
    case 'waiting':
    case 'running':
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

/**
 * Status the badge should read. A document means the conversion finished,
 * even when the list still has the status from the moment the job was created.
 */
export function visibleJobStatus(job: { status: string; documentFileId?: string | null }): string {
  const hasDocument = Boolean(job.documentFileId?.trim())
  switch (jobPhase(job.status)) {
    case 'running':
    case 'waiting':
      if (hasDocument) return 'completed'
      return job.status
    case 'finished':
    case 'failed':
    case 'canceled':
      return job.status
  }
}

/** Banner for a job that failed or was canceled. A running job has none. */
export function jobOutcomeBanner(status: string): { status: 'warning' | 'error'; title: string } | null {
  switch (jobPhase(status)) {
    case 'failed':
      return { status: 'error', title: 'Failed' }
    case 'canceled':
      return { status: 'warning', title: jobPhaseLabel(status) }
    case 'waiting':
    case 'running':
    case 'finished':
      return null
  }
}
