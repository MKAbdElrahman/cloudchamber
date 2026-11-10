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
import { ClockIcon, HomeModernIcon, QueueListIcon } from '@heroicons/react/24/outline'
import { ClockIcon as ClockSolid, HomeIcon, QueueListIcon as QueueListSolid } from '@heroicons/react/24/solid'

import type { WorkspaceSection } from './workspace'
import { WORKSPACE_LABELS } from './workspace'
import { AccountMenu } from './AccountMenu'

type AppChromeProps = {
  children: ReactNode
  section: WorkspaceSection
  accountName: string
  onSectionChange: (section: WorkspaceSection) => void
  onOpenProfile: () => void
  onOpenSettings: () => void
  onSignOut: () => void
}

/** App shell — top nav + collapsible side nav (settings via account menu). */
export function AppChrome({
  children,
  section,
  accountName,
  onSectionChange,
  onOpenProfile,
  onOpenSettings,
  onSignOut,
}: AppChromeProps) {
  return (
    <AppShell
      height="fill"
      contentPadding={0}
      variant="elevated"
      mobileNav={{ breakpoint: 'lg' }}
      topNav={
        <TopNav
          label="CloudChamber"
          heading={
            <TopNavHeading heading="CloudChamber" />
          }
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
            <HStack gap={1} vAlign="center" width="100%">
              <SideNavCollapseButton size="sm" />
            </HStack>
          }
          footer={
            <AccountMenu
              accountName={accountName}
              onProfile={onOpenProfile}
              onSettings={onOpenSettings}
              onSignOut={onSignOut}
            />
          }
        >
          <SideNavSection title="Workspace">
            <SideNavItem
              label={WORKSPACE_LABELS.overview}
              icon={section === 'overview' ? HomeIcon : HomeModernIcon}
              isSelected={section === 'overview'}
              onClick={() => onSectionChange('overview')}
            />
            <SideNavItem
              label={WORKSPACE_LABELS.pipeline}
              icon={section === 'pipeline' ? QueueListSolid : QueueListIcon}
              isSelected={section === 'pipeline'}
              onClick={() => onSectionChange('pipeline')}
            />
            <SideNavItem
              label={WORKSPACE_LABELS.history}
              icon={section === 'history' ? ClockSolid : ClockIcon}
              isSelected={section === 'history'}
              onClick={() => onSectionChange('history')}
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
