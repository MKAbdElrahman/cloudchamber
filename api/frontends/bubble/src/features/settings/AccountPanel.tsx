import { useEffect, useMemo, useState } from 'react'

import { Avatar } from '@astryxdesign/core/Avatar'
import { Banner } from '@astryxdesign/core/Banner'
import { Button } from '@astryxdesign/core/Button'
import { HStack, VStack } from '@astryxdesign/core/Layout'
import { List } from '@astryxdesign/core/List'
import { Text } from '@astryxdesign/core/Text'
import { Timestamp } from '@astryxdesign/core/Timestamp'
import { ArrowRightOnRectangleIcon, CalendarIcon, EnvelopeIcon } from '@heroicons/react/24/outline'

import { accountDisplayLabel, isUserUuid, parseServiceJwt, roleLabel, useSession } from '@/shared/auth'
import { WORKSPACE_GAP } from '@/shared/spacing'

import { fetchAccount, type AccountUser } from './accountApi'
import { SettingsRow } from './SettingsFormLayout'

function rolesLine(roles: string[]): string {
  if (roles.length === 0) return 'Member'
  return roles.map(roleLabel).join(', ')
}

/** Account identity: the person, then the record the API holds for them. */
export function AccountPanel() {
  const { token, displayName, signOut } = useSession()
  const claims = useMemo(() => (token ? parseServiceJwt(token) : null), [token])
  const userId = claims && isUserUuid(claims.sub) ? claims.sub : null
  const [account, setAccount] = useState<AccountUser | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!token || !userId) return
    let cancelled = false
    setError(null)
    fetchAccount(token, userId)
      .then((next) => {
        if (!cancelled) setAccount(next)
      })
      .catch(() => {
        if (!cancelled) setError('failed')
      })
    return () => {
      cancelled = true
    }
  }, [token, userId])

  const name = account?.name?.trim() || accountDisplayLabel(claims, displayName)
  const roles = account?.roles ?? claims?.roles ?? []
  const email = account?.email ?? (error ? 'Not available.' : userId ? 'Loading…' : 'No account.')

  return (
    <VStack gap={WORKSPACE_GAP.group} width="100%">
      <HStack gap={WORKSPACE_GAP.group} vAlign="center">
        <Avatar name={name} size="lg" />
        <VStack gap={0}>
          <Text type="label" weight="semibold">
            {name}
          </Text>
          <Text type="supporting">{rolesLine(roles)}</Text>
        </VStack>
      </HStack>

      {error ? <Banner status="error" title="Could not load the account" /> : null}
      {account && !account.enabled ? (
        <Banner status="warning" title="This account is disabled" />
      ) : null}

      <List density="spacious" hasDividers aria-label="Account">
        <SettingsRow title="Email" icon={EnvelopeIcon} description={email} />
        {account ? (
          <SettingsRow
            title="Member since"
            icon={CalendarIcon}
            detail={<Timestamp value={account.dateCreated} format="relative" />}
          />
        ) : null}
        <SettingsRow
          title="This browser"
          icon={ArrowRightOnRectangleIcon}
          control={<Button label="Sign out" variant="ghost" size="sm" onClick={signOut} />}
        />
      </List>
    </VStack>
  )
}
