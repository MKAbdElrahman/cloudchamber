import { Token } from '@astryxdesign/core/Token'

import { jobPhaseLabel, jobStatusTokenColor } from './jobStatus'

/** Job status as a category token — the same word and hue on every surface. */
export function JobStatusLabel({ status }: { status: string }) {
  return <Token label={jobPhaseLabel(status)} size="sm" color={jobStatusTokenColor(status)} />
}
