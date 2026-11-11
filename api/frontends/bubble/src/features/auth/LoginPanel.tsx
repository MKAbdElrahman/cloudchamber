import { useState } from 'react'

import { VStack } from '@astryxdesign/core/Layout'
import { Text } from '@astryxdesign/core/Text'
import { Button } from '@astryxdesign/core/Button'

import { WORKSPACE_GAP } from '@/shared/spacing'

import { authDefaults } from './api'

function continueWithGoogle() {
  const base = authDefaults.authBase.replace(/\/$/, '')
  window.location.assign(`${base}/api/auth/google`)
}

export function LoginPanel() {
  const [error] = useState<string | null>(() => {
    const message = new URLSearchParams(window.location.search).get('error')
    return message || null
  })

  return (
    <VStack gap={WORKSPACE_GAP.section} width="100%" maxWidth="24rem" hAlign="stretch">
      <VStack gap={WORKSPACE_GAP.local}>
        <Text type="body" weight="bold">
          CloudChamber
        </Text>
        <Text type="display-1" as="h1">
          Welcome back
        </Text>
        <Text type="supporting" color="secondary">
          Sign in to continue.
        </Text>
      </VStack>

      {error ? (
        <Text type="supporting" color="secondary">
          {error}
        </Text>
      ) : null}

      <VStack gap={WORKSPACE_GAP.group} hAlign="stretch">
        <Button type="button" label="Continue with Google" variant="primary" size="lg" onClick={continueWithGoogle} />
      </VStack>

      <Text type="supporting" color="secondary">
        By continuing, you agree to our Terms of Service and Privacy Policy.
      </Text>
    </VStack>
  )
}
