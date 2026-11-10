import { Children, type ReactNode } from 'react'

import { Card } from '@astryxdesign/core/Card'
import { Divider } from '@astryxdesign/core/Divider'
import { VStack } from '@astryxdesign/core/Layout'
import { Icon } from '@astryxdesign/core/Icon'
import { Text } from '@astryxdesign/core/Text'

import './settings-form.css'

type IconComponent = React.ComponentType<React.SVGProps<SVGSVGElement>>

/** Muted card grouping for settings rows (settings-dialog pattern). */
export function SettingsCard({ title, children }: { title?: string; children: ReactNode }) {
  const rows = Children.toArray(children)

  return (
    <VStack gap={1.5} width="100%" className="settings-panel">
      {title != null ? (
        <Text type="supporting" weight="semibold" color="secondary">
          {title}
        </Text>
      ) : null}
      <Card padding={0} width="100%" variant="muted">
        <VStack as="ul" role="list" gap={0} width="100%">
          {rows.map((row, index) => (
            <VStack key={index} as="li" role="listitem" gap={0} width="100%">
              {index > 0 ? <Divider variant="subtle" /> : null}
              {row}
            </VStack>
          ))}
        </VStack>
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
  return (
    <VStack padding={4} gap={2} width="100%" className="settings-row">
      <div className="settings-row-line">
        <div className="settings-row-text">
          <VStack paddingBlockStart={0.5}>
            <Icon icon={icon} size="sm" color="secondary" />
          </VStack>
          <VStack gap={0.5} style={{ flex: 1, minWidth: 0 }}>
            <Text type="label">{title}</Text>
            {description != null ? (
              <Text type="supporting" color="secondary">
                {description}
              </Text>
            ) : null}
          </VStack>
        </div>
        {control != null ? <div className="settings-row-control">{control}</div> : null}
      </div>
      {detail}
    </VStack>
  )
}
