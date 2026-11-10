import { describe, expect, it } from 'vitest'

import {
  isTerminalJobStatus,
  jobOutcomeBanner,
  jobStatusBannerStatus,
  jobStatusIntent,
  jobStatusTokenColor,
} from './jobStatus'

describe('jobStatusIntent', () => {
  it('keeps active work, failure, and a stopped run on different intents', () => {
    expect(jobStatusIntent('completed')).toBe('success')
    expect(jobStatusIntent('running')).toBe('info')
    expect(jobStatusIntent('submitted')).toBe('info')
    expect(jobStatusIntent('failed')).toBe('error')
    expect(jobStatusIntent('canceled')).toBe('warning')
    expect(jobStatusIntent('cancelled')).toBe('warning')
    expect(jobStatusIntent('timed_out')).toBe('warning')
    expect(jobStatusIntent('terminated')).toBe('warning')
    expect(jobStatusIntent('queued')).toBe('neutral')
    expect(jobStatusIntent(undefined)).toBe('neutral')
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
    expect(jobOutcomeBanner('canceled')).toEqual({ status: 'warning', title: 'Stopped' })
    expect(jobOutcomeBanner('failed')).toEqual({ status: 'error', title: 'Error' })
  })

  it('stops polling once the run has finished or stopped', () => {
    expect(isTerminalJobStatus('completed')).toBe(true)
    expect(isTerminalJobStatus('failed')).toBe(true)
    expect(isTerminalJobStatus('timed_out')).toBe(true)
    expect(isTerminalJobStatus('running')).toBe(false)
    expect(isTerminalJobStatus('queued')).toBe(false)
    expect(isTerminalJobStatus(undefined)).toBe(false)
  })
})