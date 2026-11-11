import type { ReactNode } from 'react'

import { Card } from '@astryxdesign/core/Card'
import { Icon } from '@astryxdesign/core/Icon'
import { VStack } from '@astryxdesign/core/Layout'
import { List, ListItem } from '@astryxdesign/core/List'
import { Text } from '@astryxdesign/core/Text'

import { WORKSPACE_CARD, WORKSPACE_GAP } from '@/shared/spacing'
import { SHELL_ICON_COLOR } from '@/shared/theme'

type IconComponent = React.ComponentType<React.SVGProps<SVGSVGElement>>

/** Muted card around a settings list (settings-sidebar rows). */
export function SettingsCard({ title, children }: { title?: string; children: ReactNode }) {
  return (
    <VStack gap={WORKSPACE_GAP.local} width="100%">
      {title != null ? <Text type="label">{title}</Text> : null}
      <Card width="100%" {...WORKSPACE_CARD}>
        <List density="spacious" hasDividers aria-label={title ?? 'Settings'}>
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
      <VStack gap={WORKSPACE_GAP.group}>
        {description != null ? <Text type="supporting">{description}</Text> : null}
        {detail}
      </VStack>
    ) : undefined

  return (
    <ListItem
      label={title}
      description={supporting}
      align="first-line"
      startContent={<Icon icon={icon} size="sm" color={SHELL_ICON_COLOR} />}
      endContent={control}
    />
  )
}
