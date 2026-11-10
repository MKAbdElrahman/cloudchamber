import type { ReactNode } from 'react'

import { HStack, VStack } from '@astryxdesign/core/Layout'
import { Heading, Text } from '@astryxdesign/core/Text'

import { WORKSPACE_GAP } from '@/shared/spacing'

type PageHeaderProps = {
  eyebrow?: string
  title: string
  description?: ReactNode
  end?: ReactNode
}

/** Shared page title block — eyebrow + h1 + supporting line (Astryx layout scaffold). */
export function PageHeader({ eyebrow, title, description, end }: PageHeaderProps) {
  return (
    <HStack vAlign="center" justify="between" gap={WORKSPACE_GAP.group} wrap="wrap" width="100%">
      <VStack gap={WORKSPACE_GAP.local}>
        {eyebrow ? <Text type="supporting">{eyebrow}</Text> : null}
        <Heading level={1}>{title}</Heading>
        {description == null ? null : typeof description === 'string' ? (
          <Text type="supporting">{description}</Text>
        ) : (
          description
        )}
      </VStack>
      {end}
    </HStack>
  )
}
