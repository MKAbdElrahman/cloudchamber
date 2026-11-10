import type { ReactNode } from 'react'

import { Card } from '@astryxdesign/core/Card'
import { Icon } from '@astryxdesign/core/Icon'
import { VStack } from '@astryxdesign/core/Layout'
import { List, ListItem } from '@astryxdesign/core/List'
import { Text } from '@astryxdesign/core/Text'

type IconComponent = React.ComponentType<React.SVGProps<SVGSVGElement>>

/** Muted card around a settings list (settings-sidebar rows). */
export function SettingsCard({ title, children }: { title?: string; children: ReactNode }) {
  return (
    <VStack gap={1.5} width="100%">
      {title != null ? (
        <Text type="label" color="secondary">
          {title}
        </Text>
      ) : null}
      <Card padding={0} width="100%" variant="muted">
        <List density="spacious" edge="flush" hasDividers aria-label={title ?? 'Settings'}>
          {children}
        </List>
      </Card>
    </VStack>
  )
}

export function SettingsRow({
  title,
  description,
  icon,
  control,
  detail,
}: {
  title: string
  description?: string
  icon: IconComponent
  control?: ReactNode
  detail?: ReactNode
}) {
  const supporting =
    description != null || detail != null ? (
      <VStack gap={2}>
        {description != null ? <Text color="secondary">{description}</Text> : null}
        {detail}
      </VStack>
    ) : undefined

  return (
    <ListItem
      label={title}
      description={supporting}
      align="first-line"
      startContent={<Icon icon={icon} size="sm" color="secondary" />}
      endContent={control}
    />
  )
}
