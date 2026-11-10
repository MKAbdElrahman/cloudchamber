import { Token } from '@astryxdesign/core/Token'

import { jobStatusTokenColor } from './jobStatus'

/** Job status as a category token — quiet enough to repeat on every row. */
export function JobStatusLabel({ status }: { status: string }) {
  return <Token label={status} size="sm" color={jobStatusTokenColor(status)} />
}
