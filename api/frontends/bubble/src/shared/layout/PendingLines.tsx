import { Card } from '@astryxdesign/core/Card'
import { Grid } from '@astryxdesign/core/Grid'
import { VStack } from '@astryxdesign/core/Layout'
import { Skeleton } from '@astryxdesign/core/Skeleton'

import { WORKSPACE_CARD, WORKSPACE_GAP } from '@/shared/spacing'

type PendingLinesProps = {
  /**
   * Shape of the content that is about to appear.
   * text — a document. rows — a library list. cards — a library grid.
   */
  variant?: 'text' | 'rows' | 'cards'
}

/** Placeholder that keeps a page's content slot while data is in flight. */
export function PendingLines({ variant = 'text' }: PendingLinesProps) {
  if (variant === 'cards') {
    return (
      <Grid columns={{ minWidth: 240, max: 4 }} gap={WORKSPACE_GAP.group} width="100%" aria-busy="true" aria-label="Loading">
        {[0, 1, 2, 3].map((index) => (
          <Card key={index} {...WORKSPACE_CARD} height="100%">
            <VStack gap={WORKSPACE_GAP.local}>
              <Skeleton height="1rem" width="72%" index={index * 2} />
              <Skeleton height="0.75rem" width="40%" index={index * 2 + 1} />
            </VStack>
          </Card>
        ))}
      </Grid>
    )
  }

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
