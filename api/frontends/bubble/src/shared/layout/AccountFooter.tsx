import { useLocation } from 'react-router-dom'

import { Avatar } from '@astryxdesign/core/Avatar'
import { Button } from '@astryxdesign/core/Button'
import { Icon } from '@astryxdesign/core/Icon'
import { SideNavItem, SideNavSection } from '@astryxdesign/core/SideNav'
import { Cog6ToothIcon } from '@heroicons/react/24/outline'

import { SHELL_ICON_COLOR } from '@/shared/theme'

import { WORKSPACE_PATHS } from './workspacePaths'

type AccountFooterProps = {
  accountName: string
}

/**
 * Rail footer. The person opens Account. The gear opens Settings and is
 * hidden while the rail is collapsed, which is how a SideNav row action works.
 */
export function AccountFooter({ accountName }: AccountFooterProps) {
  const { pathname } = useLocation()
  const accountPath = WORKSPACE_PATHS.settingsSection('account')

  return (
    <SideNavSection title="Account" isHeaderHidden>
      <SideNavItem
        label={accountName}
        icon={<Avatar name={accountName} size="md" tooltip={false} aria-hidden />}
        href={accountPath}
        isSelected={pathname === accountPath}
        actions={
          <Button
            label="Settings"
            variant="ghost"
            isIconOnly
            icon={<Icon icon={Cog6ToothIcon} size="sm" color={SHELL_ICON_COLOR} />}
            href={WORKSPACE_PATHS.settingsSection('appearance')}
          />
        }
      />
    </SideNavSection>
  )
}
