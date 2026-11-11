import type { ReactNode } from 'react'

import { Icon } from '@astryxdesign/core/Icon'
import { VStack } from '@astryxdesign/core/Layout'
import { ListItem } from '@astryxdesign/core/List'
import { Text } from '@astryxdesign/core/Text'

import { WORKSPACE_GAP } from '@/shared/spacing'
import { SHELL_ICON_COLOR } from '@/shared/theme'

type IconComponent = React.ComponentType<React.SVGProps<SVGSVGElement>>

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
