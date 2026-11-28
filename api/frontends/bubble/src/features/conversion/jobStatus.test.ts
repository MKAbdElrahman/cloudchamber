import { describe, expect, it } from 'vitest'

import {
  isTerminalJobStatus,
  jobOutcomeBanner,
  jobPhaseLabel,
  jobStatusBannerStatus,
  jobStatusIntent,
  jobStatusTokenColor,
  visibleJobStatus,
} from './jobStatus'

describe('jobStatusIntent', () => {
  it('keeps active work, failure, and a stopped run on different intents', () => {
    expect(jobStatusIntent('completed')).toBe('success')
    expect(jobStatusIntent('running')).toBe('info')
    expect(jobStatusIntent('submitted')).toBe('neutral')
    expect(jobPhaseLabel('submitted')).toBe('Waiting')
    expect(jobStatusIntent('failed')).toBe('error')
    expect(jobStatusIntent('canceled')).toBe('warning')
    expect(jobStatusIntent('cancelled')).toBe('warning')
    expect(jobStatusIntent('queued')).toBe('neutral')
    expect(jobStatusIntent('uploading')).toBe('info')
    expect(jobStatusIntent(undefined)).toBe('neutral')
    expect(jobPhaseLabel('uploading')).toBe('Uploading')
  })

  it('uses one hue and one banner status for each intent', () => {
    expect(jobStatusTokenColor('completed')).toBe('green')
    expect(jobStatusTokenColor('running')).toBe('blue')
    expect(jobStatusTokenColor('failed')).toBe('red')
    expect(jobStatusTokenColor('canceled')).toBe('orange')
    expect(jobStatusTokenColor('queued')).toBe('gray')
    expect(jobStatusBannerStatus('canceled')).toBe('warning')
    expect(jobStatusBannerStatus('failed')).toBe('error')
    expect(jobStatusBannerStatus('queued')).toBe('info')
    expect(jobOutcomeBanner('canceled')).toEqual({ status: 'warning', title: 'Canceled' })
    expect(jobOutcomeBanner('failed')).toEqual({ status: 'error', title: 'Failed' })
    expect(jobOutcomeBanner('running')).toBeNull()
    expect(jobOutcomeBanner('completed')).toBeNull()
  })

  it('stops polling once the run has finished or stopped', () => {
    expect(isTerminalJobStatus('completed')).toBe(true)
    expect(isTerminalJobStatus('failed')).toBe(true)
    expect(isTerminalJobStatus('canceled')).toBe(true)
    expect(isTerminalJobStatus('running')).toBe(false)
    expect(isTerminalJobStatus('queued')).toBe(false)
    expect(isTerminalJobStatus(undefined)).toBe(false)
    expect(isTerminalJobStatus('uploading')).toBe(false)
  })

  it('reads the badge from the stored run', () => {
    expect(visibleJobStatus({ status: 'running' })).toBe('running')
    expect(visibleJobStatus({ status: 'failed' })).toBe('failed')
  })

  it('shows a paused run as open and neutral', () => {
    expect(jobPhaseLabel('paused')).toBe('Paused')
    expect(jobStatusIntent('paused')).toBe('neutral')
    expect(isTerminalJobStatus('paused')).toBe(false)
    expect(jobOutcomeBanner('paused')).toBeNull()
  })
})
