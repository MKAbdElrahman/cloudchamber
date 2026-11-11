import { useEffect, useState } from 'react'
import { useMediaQuery } from '@astryxdesign/core/hooks'
import { useNavigate, useParams } from 'react-router-dom'

import { LayoutPanel, VStack } from '@astryxdesign/core/Layout'
import { Heading, Text } from '@astryxdesign/core/Text'
import { List, ListItem } from '@astryxdesign/core/List'
import { Icon } from '@astryxdesign/core/Icon'
import { MetadataList, MetadataListItem } from '@astryxdesign/core/MetadataList'
import { Switch } from '@astryxdesign/core/Switch'
import { TextInput } from '@astryxdesign/core/TextInput'
import { Link } from '@astryxdesign/core/Link'
import {
  CodeBracketIcon,
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
import { env, readCustomTusdEndpoint, writeCustomTusdEndpoint } from '@/shared/config'
import { PageHeader, SHELL_NARROW_QUERY, TEXT_CONTENT_WIDTH, WorkspacePage, WORKSPACE_PATHS } from '@/shared/layout'
import { WORKSPACE_GAP } from '@/shared/spacing'
import { SHELL_ICON_COLOR } from '@/shared/theme'

const NAV: { id: SettingsSectionId; label: string; icon: typeof PaintBrushIcon }[] = [
  { id: 'appearance', label: 'Appearance', icon: PaintBrushIcon },
  { id: 'account', label: 'Account', icon: UserCircleIcon },
  { id: 'developer', label: 'Developer', icon: CodeBracketIcon },
]

const DEVELOPER_LINKS = [
  { label: 'Astryx repository', href: 'https://github.com/meta/astryx' },
  { label: 'Design conventions (wiki)', href: 'https://github.com/meta/astryx/wiki/Design-Conventions' },
  { label: 'Template sync checklist', href: 'https://github.com/meta/astryx/tree/main/packages/cli/assets/templates' },
] as const

function TusdEndpointSetting() {
  const stored = readCustomTusdEndpoint()
  const [custom, setCustom] = useState(stored !== null)
  const [value, setValue] = useState(stored ?? env.tusdEndpoint)

  function toggle(on: boolean) {
    setCustom(on)
    if (!on) {
      writeCustomTusdEndpoint(null)
      return
    }
    writeCustomTusdEndpoint(value.trim() || env.tusdEndpoint)
  }

  function change(next: string) {
    setValue(next)
    writeCustomTusdEndpoint(next)
  }

  return (
    <VStack gap={WORKSPACE_GAP.group}>
      <Switch
        label="Custom tus endpoint"
        description="Uploads use this instead of the endpoint configured for the app."
        value={custom}
        onChange={toggle}
      />
      {custom ? <TextInput label="tusd endpoint" value={value} onChange={change} /> : null}
    </VStack>
  )
}

/**
 * Settings regions, and the mechanism for each:
 *   > lg (1024)  start panel 17.5rem holds its width; content extends to the open edge
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
    <VStack gap={WORKSPACE_GAP.section} width="100%">
      <Heading level={2}>{sectionTitle}</Heading>
      {section === 'appearance' ? (
        <AppearancePanel />
      ) : null}

      {section === 'account' ? <AccountPanel /> : null}

      {section === 'developer' ? (
        <VStack gap={WORKSPACE_GAP.group}>
          <MetadataList>
            <MetadataListItem label="CloudChamber API">{env.cloudchamberApiBase}</MetadataListItem>
            <MetadataListItem label="Auth">{env.authApiBase}</MetadataListItem>
            <MetadataListItem label="tusd">{env.tusdEndpoint}</MetadataListItem>
            <MetadataListItem label="Artifacts bucket">{env.artifactsBucket}</MetadataListItem>
          </MetadataList>
          <TusdEndpointSetting />
          <VStack gap={WORKSPACE_GAP.local}>
            <Text type="label">Astryx references</Text>
            <Text type="supporting">
              Local checklist: <Text type="code">references/astryx/TEMPLATE_SYNC.md</Text>
            </Text>
            <VStack gap={WORKSPACE_GAP.local} paddingBlockStart={WORKSPACE_GAP.local}>
              {DEVELOPER_LINKS.map((item) => (
                <Link key={item.href} href={item.href} target="_blank" rel="noreferrer">
                  {item.label}
                </Link>
              ))}
            </VStack>
          </VStack>
        </VStack>
      ) : null}
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
      <WorkspacePage contentWidth={TEXT_CONTENT_WIDTH}>
        <VStack gap={WORKSPACE_GAP.section}>
          <Heading level={1}>Settings</Heading>
          {sectionList}
          {sectionBody}
        </VStack>
      </WorkspacePage>
    )
  }

  return (
    <WorkspacePage
      start={
        <LayoutPanel width="17.5rem" hasDivider>
          {sectionList}
        </LayoutPanel>
      }
      header={<PageHeader title="Settings" />}
    >
      {sectionBody}
    </WorkspacePage>
  )
}
