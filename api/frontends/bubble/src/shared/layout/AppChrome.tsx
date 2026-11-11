import type { ReactNode } from 'react'

import { AppShell } from '@astryxdesign/core/AppShell'
import { HStack, Layout, LayoutContent } from '@astryxdesign/core/Layout'
import {
  SideNav,
  SideNavCollapseButton,
  SideNavHeading,
  SideNavItem,
  SideNavSection,
} from '@astryxdesign/core/SideNav'
import { FolderIcon } from '@heroicons/react/24/outline'
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
 * One bar. The product name lives in the side nav; there is no top bar.
 *   >= md (768)   SideNav, resizable 200–400, default 260
 *   < md          that nav becomes the mobile bar and drawer
 * Settings drops its own start panel at lg (1024); see SHELL_NARROW_QUERY.
 */
export function AppChrome({ children, section, accountName, onSignOut }: AppChromeProps) {
  return (
    <AppShell
      height="fill"
      contentPadding={0}
      variant="elevated"
      mobileNav={{ breakpoint: 'md' }}
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
            <HStack gap={WORKSPACE_GAP.local} vAlign="center" justify="between" width="100%">
              <SideNavHeading heading="CloudChamber" headingHref={WORKSPACE_PATHS.home} />
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
