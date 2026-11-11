import { DropdownMenu } from '@astryxdesign/core/DropdownMenu'
import { Icon } from '@astryxdesign/core/Icon'
import { useSideNavCollapse } from '@astryxdesign/core/SideNav'
import {
  ArrowRightOnRectangleIcon,
  Cog6ToothIcon,
  UserCircleIcon,
} from '@heroicons/react/24/outline'

import { SHELL_ICON_COLOR } from '@/shared/theme'

import { WORKSPACE_PATHS } from './workspacePaths'

type AccountMenuProps = {
  accountName: string
  onSignOut: () => void
}

/** SideNav footer account control — icon-only when the rail is collapsed. */
export function AccountMenu({ accountName, onSignOut }: AccountMenuProps) {
  const { isCollapsed } = useSideNavCollapse()

  return (
    <DropdownMenu
      button={{
        label: accountName,
        variant: 'ghost',
        size: isCollapsed ? 'sm' : 'md',
        isIconOnly: isCollapsed,
        icon: <Icon icon={UserCircleIcon} size="sm" color={SHELL_ICON_COLOR} />,
      }}
      hasChevron={false}
      // The rail clips overflow. Matching the trigger keeps the open menu
      // inside that padding. A collapsed rail is only an icon, so the menu
      // opens beside it, where the radius is not cut by the rail edge.
      placement={isCollapsed ? 'end' : 'above'}
      alignment="start"
      menuWidth={isCollapsed ? '16.25rem' : undefined}
      presentation="popover"
      items={[
        {
          label: 'Profile',
          description: 'Session identity',
          icon: UserCircleIcon,
          href: WORKSPACE_PATHS.settingsSection('account'),
        },
        {
          label: 'Settings',
          icon: Cog6ToothIcon,
          href: WORKSPACE_PATHS.settingsSection('appearance'),
        },
        { type: 'divider' },
        {
          label: 'Sign out',
          icon: ArrowRightOnRectangleIcon,
          variant: 'destructive',
          onClick: onSignOut,
        },
      ]}
    />
  )
}
