import type { ReactNode } from 'react'

import { HStack, VStack } from '@astryxdesign/core/Layout'
import { Heading, Text } from '@astryxdesign/core/Text'

type PageHeaderProps = {
  eyebrow?: string
  title: string
  description?: ReactNode
  end?: ReactNode
}

/** Shared page title block — eyebrow + h1 + supporting line (Astryx layout scaffold). */
export function PageHeader({ eyebrow, title, description, end }: PageHeaderProps) {
  return (
    <HStack vAlign="center" justify="between" gap={4}>
      <VStack gap={1}>
        {eyebrow ? <Text type="supporting">{eyebrow}</Text> : null}
        <Heading level={1}>{title}</Heading>
        {description == null ? null : typeof description === 'string' ? (
          <Text color="secondary">{description}</Text>
        ) : (
          description
        )}
      </VStack>
      {end}
    </HStack>
  )
}
