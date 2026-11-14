import { VStack } from '@astryxdesign/core/Layout'
import { Skeleton } from '@astryxdesign/core/Skeleton'

import { WORKSPACE_GAP } from '@/shared/spacing'

type PendingLinesProps = {
  /**
   * Shape of the content that is about to appear.
   * text — a document. rows — a library list.
   */
  variant?: 'text' | 'rows'
}

/** Placeholder that keeps a page's content slot while data is in flight. */
export function PendingLines({ variant = 'text' }: PendingLinesProps) {
  if (variant === 'rows') {
    return (
      <VStack gap={WORKSPACE_GAP.group} width="100%" aria-busy="true" aria-label="Loading">
        {[0, 1, 2].map((index) => (
          <VStack key={index} gap={WORKSPACE_GAP.local}>
            <Skeleton height="1rem" width="46%" index={index * 2} />
            <Skeleton height="0.75rem" width="28%" index={index * 2 + 1} />
          </VStack>
        ))}
      </VStack>
    )
  }

  return (
    <VStack gap={WORKSPACE_GAP.group} width="100%" aria-busy="true" aria-label="Loading">
      <Skeleton height="1rem" width="36%" index={0} />
      <Skeleton height="0.75rem" index={1} />
      <Skeleton height="0.75rem" width="72%" index={2} />
    </VStack>
  )
}
