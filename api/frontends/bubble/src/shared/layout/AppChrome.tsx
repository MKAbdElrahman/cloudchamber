import type { ReactNode } from 'react'

import { AppShell } from '@astryxdesign/core/AppShell'
import { HStack, Layout, LayoutContent } from '@astryxdesign/core/Layout'
import { TopNav, TopNavHeading } from '@astryxdesign/core/TopNav'
import {
  SideNav,
  SideNavCollapseButton,
  SideNavItem,
  SideNavSection,
} from '@astryxdesign/core/SideNav'
import { Cog6ToothIcon, FolderIcon } from '@heroicons/react/24/outline'
import { FolderIcon as FolderSolid } from '@heroicons/react/24/solid'

import { WORKSPACE_GAP } from '@/shared/spacing'

import { AccountMenu } from './AccountMenu'
import type { WorkspaceSection } from './workspace'
import { WORKSPACE_PATHS } from './workspacePaths'

type AppChromeProps = {
  children: ReactNode
  section: WorkspaceSection
  accountName: string
  onSignOut: () => void
}

/**
 * App shell. Fills the viewport (height fill = 100dvh) and swaps regions:
 *   >= md (768)   SideNav, resizable 200–400, default 260
 *   < md          SideNav moves into the mobile drawer
 * Settings drops its own start panel at lg (1024); see SHELL_NARROW_QUERY.
 */
export function AppChrome({ children, section, accountName, onSignOut }: AppChromeProps) {
  return (
    <AppShell
      height="fill"
      contentPadding={0}
      variant="elevated"
      mobileNav={{ breakpoint: 'md' }}
      topNav={
        <TopNav
          label="CloudChamber"
          heading={<TopNavHeading heading="CloudChamber" />}
        />
      }
      sideNav={
        <SideNav
          aria-label="Workspace navigation"
          collapsible={{ hasButton: false }}
          resizable={{
            defaultWidth: 260,
            minWidth: 200,
            maxWidth: 400,
            autoSaveId: 'bubble.shell-sidenav',
          }}
          header={
            <HStack gap={WORKSPACE_GAP.local} vAlign="center" width="100%">
              <SideNavCollapseButton size="sm" />
            </HStack>
          }
          footer={<AccountMenu accountName={accountName} onSignOut={onSignOut} />}
        >
          <SideNavSection title="Workspace">
            <SideNavItem
              label="Projects"
              icon={FolderIcon}
              selectedIcon={FolderSolid}
              isSelected={section === 'projects'}
              href={WORKSPACE_PATHS.home}
            />
            <SideNavItem
              label="Settings"
              icon={Cog6ToothIcon}
              isSelected={section === 'settings'}
              href={WORKSPACE_PATHS.settings}
            />
          </SideNavSection>
        </SideNav>
      }
    >
      <Layout height="fill">
        <LayoutContent padding={0}>{children}</LayoutContent>
      </Layout>
    </AppShell>
  )
}
