import { useEffect } from 'react'
import { useMediaQuery } from '@astryxdesign/core/hooks'
import { useNavigate, useParams } from 'react-router-dom'

import { LayoutPanel, VStack } from '@astryxdesign/core/Layout'
import { Heading, Text } from '@astryxdesign/core/Text'
import { List, ListItem } from '@astryxdesign/core/List'
import { Icon } from '@astryxdesign/core/Icon'
import {
  PaintBrushIcon,
  UserCircleIcon,
} from '@heroicons/react/24/outline'

import {
  AccountPanel,
  AppearancePanel,
  SettingsSectionSearch,
  isSettingsSectionId,
  useSettingsSectionSearch,
  type SettingsSectionId,
} from '@/features/settings'
import { PageHeader, SHELL_NARROW_QUERY, SIDE_PANEL_WIDTH, TEXT_CONTENT_WIDTH, WorkspacePage, WORKSPACE_PATHS } from '@/shared/layout'
import { WORKSPACE_GAP } from '@/shared/spacing'
import { SHELL_ICON_COLOR } from '@/shared/theme'

const NAV: { id: SettingsSectionId; label: string; icon: typeof PaintBrushIcon }[] = [
  { id: 'appearance', label: 'Appearance', icon: PaintBrushIcon },
  { id: 'account', label: 'Account', icon: UserCircleIcon },
]

/**
 * Settings regions, and the mechanism for each:
 *   > lg (1024)  start panel holds its width; content extends to the open edge
 *   <= lg        the panel is swapped off; the section list stacks above the pane,
 *                and that single region centers on the 640 settings measure
 */

/** Settings sidebar — section links beside the active pane. Below the shell's lg point the list stacks above the pane. */
export function SettingsPanel() {
  const isNarrow = useMediaQuery(SHELL_NARROW_QUERY)
  const navigate = useNavigate()
  const { section: sectionParam } = useParams<{ section: string }>()
  const section: SettingsSectionId = isSettingsSectionId(sectionParam) ? sectionParam : 'appearance'
  const { query, setQuery, results, isFiltering } = useSettingsSectionSearch()

  useEffect(() => {
    if (!isSettingsSectionId(sectionParam)) {
      navigate(WORKSPACE_PATHS.settingsSection('appearance'), { replace: true })
    }
  }, [sectionParam, navigate])

  const activeNav = NAV.find((n) => n.id === section)

  const sectionTitle = activeNav?.label ?? 'Settings'

  const sectionBody = (
    <VStack gap={WORKSPACE_GAP.section} width="100%" maxWidth={TEXT_CONTENT_WIDTH}>
      <Heading level={2}>{sectionTitle}</Heading>
      {section === 'appearance' ? (
        <AppearancePanel />
      ) : null}

      {section === 'account' ? <AccountPanel /> : null}
    </VStack>
  )

  const navItems = isFiltering
    ? NAV.filter((item) => results.some((hit) => hit.id === item.id))
    : NAV

  const sectionList = (
    <VStack gap={WORKSPACE_GAP.group}>
      <SettingsSectionSearch query={query} onQueryChange={setQuery} />
      {isFiltering && navItems.length === 0 ? (
        <Text type="supporting">No matching sections.</Text>
      ) : null}
      <List density="spacious" aria-label="Settings sections">
        {navItems.map((item) => (
          <ListItem
            key={item.id}
            label={item.label}
            href={WORKSPACE_PATHS.settingsSection(item.id)}
            isSelected={section === item.id}
            startContent={<Icon icon={item.icon} size="sm" color={SHELL_ICON_COLOR} />}
          />
        ))}
      </List>
    </VStack>
  )

  if (isNarrow) {
    return (
      <WorkspacePage contentWidth={TEXT_CONTENT_WIDTH} header={<PageHeader title="Settings" />}>
        <VStack gap={WORKSPACE_GAP.section}>
          {sectionList}
          {sectionBody}
        </VStack>
      </WorkspacePage>
    )
  }

  return (
    <WorkspacePage
      start={
        <LayoutPanel width={SIDE_PANEL_WIDTH} hasDivider>
          {sectionList}
        </LayoutPanel>
      }
      header={<PageHeader title="Settings" />}
    >
      {sectionBody}
    </WorkspacePage>
  )
}
