import { useEffect, useState } from 'react'

import { VStack } from '@astryxdesign/core/Layout'
import { Heading, Text } from '@astryxdesign/core/Text'
import { Icon } from '@astryxdesign/core/Icon'
import { Button } from '@astryxdesign/core/Button'
import { TextInput } from '@astryxdesign/core/TextInput'
import { TextArea } from '@astryxdesign/core/TextArea'
import { Tab, TabList } from '@astryxdesign/core/TabList'
import { Switch } from '@astryxdesign/core/Switch'
import { EmptyState } from '@astryxdesign/core/EmptyState'
import { CheckCircleIcon } from '@heroicons/react/24/outline'

import { authHandoffDelayMs } from '@/shared/auth'
import { WORKSPACE_GAP } from '@/shared/spacing'

import { authDefaults, fetchToken } from './api'

type AuthTab = 'credentials' | 'token'

export function LoginPanel({
  onAuthenticated,
}: {
  onAuthenticated: (token: string, displayName?: string) => void
}) {
  const [tab, setTab] = useState<AuthTab>('credentials')
  const [showAdvanced, setShowAdvanced] = useState(false)
  const [authBase, setAuthBase] = useState(authDefaults.authBase)
  const [kid, setKid] = useState(authDefaults.kid)
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [manualToken, setManualToken] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [success, setSuccess] = useState<{ token: string; displayName?: string } | null>(null)

  useEffect(() => {
    if (!success) return
    const t = window.setTimeout(() => {
      onAuthenticated(success.token, success.displayName)
    }, authHandoffDelayMs())
    return () => window.clearTimeout(t)
  }, [success, onAuthenticated])

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      const token = await fetchToken({ authBase, kid, username, password })
      setSuccess({ token, displayName: username.trim() || undefined })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Authentication failed')
    } finally {
      setLoading(false)
    }
  }

  if (success) {
    return (
      <VStack gap={WORKSPACE_GAP.section} height="100%" vAlign="center" hAlign="stretch" width="100%">
        <EmptyState
          icon={<Icon icon={CheckCircleIcon} size="lg" color="success" />}
          title="Signed in"
          description="Opening workspace…"
        />
      </VStack>
    )
  }

  function useManualToken() {
    const token = manualToken.trim()
    if (!token) return
    setSuccess({ token })
  }

  return (
    <VStack gap={WORKSPACE_GAP.section} height="100%" vAlign="center" hAlign="stretch" width="100%">
      <VStack gap={WORKSPACE_GAP.local} width="100%">
        <Heading level={2}>Sign in</Heading>
        <Text type="supporting">Pipeline studio for your organization</Text>
      </VStack>

      <TabList value={tab} onChange={(v) => setTab(v as AuthTab)}>
        <Tab value="credentials" label="Credentials" />
        <Tab value="token" label="Bearer token" />
      </TabList>

      {tab === 'credentials' ? (
        <form onSubmit={onSubmit}>
          <VStack gap={WORKSPACE_GAP.group}>
            <TextInput
              label="Email or username"
              placeholder="name@company.com"
              value={username}
              onChange={setUsername}
              size="lg"
              autoComplete="username"
            />
            <TextInput
              label="Password"
              placeholder="Enter your password"
              type="password"
              value={password}
              onChange={(v) => {
                setPassword(v)
                setError(null)
              }}
              size="lg"
              autoComplete="current-password"
              status={
                error
                  ? {
                      type: 'error',
                      message: error,
                    }
                  : undefined
              }
            />

            <Switch label="Developer endpoints" value={showAdvanced} onChange={setShowAdvanced} />

            {showAdvanced ? (
              <VStack gap={WORKSPACE_GAP.group}>
                <TextInput label="Auth service base URL" value={authBase} onChange={setAuthBase} />
                <TextInput label="Key ID (kid)" value={kid} onChange={setKid} />
              </VStack>
            ) : null}

            <Button
              type="submit"
              label={loading ? 'Signing in…' : 'Continue'}
              variant="primary"
              size="lg"
              isDisabled={loading}
              isLoading={loading}
            />
          </VStack>
        </form>
      ) : (
        <VStack gap={WORKSPACE_GAP.group}>
          <TextArea
            label="Bearer token"
            value={manualToken}
            onChange={setManualToken}
            placeholder="eyJhbGciOi..."
            rows={5}
          />
          <Button
            type="button"
            label="Use token"
            variant="primary"
            size="lg"
            onClick={useManualToken}
            isDisabled={!manualToken.trim()}
          />
        </VStack>
      )}
    </VStack>
  )
}
