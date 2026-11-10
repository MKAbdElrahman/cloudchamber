import { Badge } from '@astryxdesign/core/Badge'

import { jobStatusBadgeVariant } from './jobStatus'

/** Job status — single semantic Badge (same token family as Banner / Stepper outcomes). */
export function JobStatusLabel({ status }: { status: string }) {
  return <Badge variant={jobStatusBadgeVariant(status)} label={status} />
}
