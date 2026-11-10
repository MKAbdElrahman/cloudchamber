import { VStack } from '@astryxdesign/core/Layout'
import { Skeleton } from '@astryxdesign/core/Skeleton'

import { WORKSPACE_GAP } from '@/shared/spacing'

/** Placeholder rows that keep a page's content slot while data is in flight. */
export function PendingLines() {
  return (
    <VStack gap={WORKSPACE_GAP.group} width="100%" aria-busy="true" aria-label="Loading">
      <Skeleton height="1rem" width="36%" index={0} />
      <Skeleton height="0.75rem" index={1} />
      <Skeleton height="0.75rem" width="72%" index={2} />
    </VStack>
  )
}
