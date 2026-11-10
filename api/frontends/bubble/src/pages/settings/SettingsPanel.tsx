import { useEffect, useMemo } from 'react'
import { useMediaQuery } from '@astryxdesign/core/hooks'
import { useNavigate, useParams } from 'react-router-dom'

import { LayoutPanel, HStack, VStack } from '@astryxdesign/core/Layout'
import { Heading, Text } from '@astryxdesign/core/Text'
import { List, ListItem } from '@astryxdesign/core/List'
import { Icon } from '@astryxdesign/core/Icon'
import { MetadataList, MetadataListItem } from '@astryxdesign/core/MetadataList'
import { Token } from '@astryxdesign/core/Token'
import { Timestamp } from '@astryxdesign/core/Timestamp'
import { Link } from '@astryxdesign/core/Link'
import {
  CodeBracketIcon,
  PaintBrushIcon,
  UserCircleIcon,
} from '@heroicons/react/24/outline'

import { AppearancePanel, SettingsSectionSearch, useSettingsSectionSearch } from '@/features/settings'
import { accountDisplayLabel, parseServiceJwt, useSession } from '@/shared/auth'
import { env } from '@/shared/config'
import {
  isSettingsSectionId,
  PageHeader,
  SHELL_NARROW_QUERY,
  WorkspacePage,
  WORKSPACE_PATHS,
  type SettingsSectionId,
} from '@/shared/layout'
import { WORKSPACE_GAP } from '@/shared/spacing'
import { META_TOKEN_COLOR, SHELL_ICON_COLOR, useThemePreference } from '@/shared/theme'

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

/** Settings sidebar — section links beside the active pane. Below the shell's lg point the list stacks above the pane. */
export function SettingsPanel() {
  const isNarrow = useMediaQuery(SHELL_NARROW_QUERY)
  const navigate = useNavigate()
  const { section: sectionParam } = useParams<{ section: string }>()
  const section: SettingsSectionId = isSettingsSectionId(sectionParam) ? sectionParam : 'appearance'
  const { query, setQuery, results, isFiltering } = useSettingsSectionSearch()
  const { themeId, setThemeId, colorMode, setColorMode } = useThemePreference()
  const { token, displayName } = useSession()

  useEffect(() => {
    if (!isSettingsSectionId(sectionParam)) {
      navigate(WORKSPACE_PATHS.settingsSection('appearance'), { replace: true })
    }
  }, [sectionParam, navigate])

  const claims = useMemo(() => (token ? parseServiceJwt(token) : null), [token])
  const accountLabel = accountDisplayLabel(claims, displayName)
  const activeNav = NAV.find((n) => n.id === section)

  const sectionTitle = activeNav?.label ?? 'Settings'

  const sectionBody = (
    <VStack gap={WORKSPACE_GAP.section}>
      <Heading level={2}>{sectionTitle}</Heading>
      {section === 'appearance' ? (
        <AppearancePanel
          themeId={themeId}
          setThemeId={setThemeId}
          colorMode={colorMode}
          setColorMode={setColorMode}
        />
      ) : null}

      {section === 'account' ? (
        <MetadataList>
          <MetadataListItem label="Display name">{accountLabel}</MetadataListItem>
          <MetadataListItem label="User ID (sub)">{claims?.sub ?? '—'}</MetadataListItem>
          <MetadataListItem label="Roles">
            {claims && claims.roles.length > 0 ? (
              <HStack gap={WORKSPACE_GAP.local} wrap="wrap">
                {claims.roles.map((role) => (
                  <Token key={role} label={role} size="sm" color={META_TOKEN_COLOR} />
                ))}
              </HStack>
            ) : (
              '—'
            )}
          </MetadataListItem>
          <MetadataListItem label="Issuer">{claims?.iss ?? '—'}</MetadataListItem>
          <MetadataListItem label="Expires">
            {claims?.expiresAt ? (
              <Timestamp value={claims.expiresAt.toISOString()} format="relative" />
            ) : (
              '—'
            )}
          </MetadataListItem>
        </MetadataList>
      ) : null}

      {section === 'developer' ? (
        <VStack gap={WORKSPACE_GAP.group}>
          <MetadataList>
            <MetadataListItem label="CloudChamber API">{env.cloudchamberApiBase}</MetadataListItem>
            <MetadataListItem label="Auth">{env.authApiBase}</MetadataListItem>
            <MetadataListItem label="tusd">{env.tusdEndpoint}</MetadataListItem>
            <MetadataListItem label="Artifacts bucket">{env.artifactsBucket}</MetadataListItem>
          </MetadataList>
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
      <List density="spacious" edge="flush" hasDividers aria-label="Settings sections">
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
      <WorkspacePage>
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
