import { useEffect, useMemo, useState } from 'react'

import { Avatar } from '@astryxdesign/core/Avatar'
import { HStack, VStack } from '@astryxdesign/core/Layout'
import { Text } from '@astryxdesign/core/Text'
import { Timestamp } from '@astryxdesign/core/Timestamp'
import { CalendarIcon, EnvelopeIcon } from '@heroicons/react/24/outline'

import { accountDisplayLabel, isUserUuid, parseServiceJwt, roleLabel, useSession } from '@/shared/auth'
import { WORKSPACE_GAP } from '@/shared/spacing'

import { fetchAccount, type AccountUser } from './accountApi'
import { SettingsCard, SettingsRow } from './SettingsFormLayout'

function rolesLine(roles: string[]): string {
  if (roles.length === 0) return 'Member'
  return roles.map(roleLabel).join(', ')
}

/** Account identity: the person, then the record the API holds for them. */
export function AccountPanel() {
  const { token, displayName } = useSession()
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
        if (!cancelled) setError('The account record could not be loaded.')
      })
    return () => {
      cancelled = true
    }
  }, [token, userId])

  const name = account?.name?.trim() || accountDisplayLabel(claims, displayName)
  const roles = account?.roles ?? claims?.roles ?? []

  return (
    <VStack gap={WORKSPACE_GAP.section} width="100%">
      <HStack gap={WORKSPACE_GAP.group} vAlign="center">
        <Avatar name={name} size="lg" />
        <VStack gap={0}>
          <Text type="body" weight="bold">
            {name}
          </Text>
          <Text type="supporting" color="secondary">
            {rolesLine(roles)}
          </Text>
        </VStack>
      </HStack>

      <SettingsCard>
        <SettingsRow
          title="Email"
          icon={EnvelopeIcon}
          description={
            account?.email ??
            (error ? error : userId ? 'Loading the account…' : 'No account is attached to this session.')
          }
        />
        {account ? (
          <SettingsRow
            title="Member since"
            icon={CalendarIcon}
            detail={<Timestamp value={account.dateCreated} format="relative" />}
          />
        ) : null}
      </SettingsCard>

      {account && !account.enabled ? (
        <Text type="supporting" color="secondary">
          This account is disabled.
        </Text>
      ) : null}
    </VStack>
  )
}
