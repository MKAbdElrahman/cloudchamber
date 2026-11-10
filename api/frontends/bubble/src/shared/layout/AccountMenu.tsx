import { DropdownMenu } from '@astryxdesign/core/DropdownMenu'
import { Icon } from '@astryxdesign/core/Icon'
import { VStack } from '@astryxdesign/core/Layout'
import { Text } from '@astryxdesign/core/Text'
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
      placement="above"
      alignment={isCollapsed ? 'center' : 'start'}
      menuWidth={260}
      presentation="popover"
      header={
        <VStack gap={0}>
          <Text type="label">{accountName}</Text>
        </VStack>
      }
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
