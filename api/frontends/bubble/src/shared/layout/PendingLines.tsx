import { VStack } from '@astryxdesign/core/Layout'
import { Skeleton } from '@astryxdesign/core/Skeleton'

import { WORKSPACE_GAP } from '@/shared/spacing'

/** Placeholder rows that keep a page's content slot while data is in flight. */
export function PendingLines() {
  return (
    <VStack gap={WORKSPACE_GAP.group} width="100%" aria-busy="true" aria-label="Loading">
      <Skeleton height={16} width="36%" index={0} />
      <Skeleton height={12} index={1} />
      <Skeleton height={12} width="72%" index={2} />
    </VStack>
  )
}
