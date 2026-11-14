import { useState } from 'react'

import { Banner } from '@astryxdesign/core/Banner'
import { VStack } from '@astryxdesign/core/Layout'
import { Heading, Text } from '@astryxdesign/core/Text'
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
        <Text type="supporting">CloudChamber</Text>
        <Heading level={1}>Welcome back</Heading>
        <Text type="supporting">Sign in to continue.</Text>
      </VStack>

      {error ? <Banner status="error" title="Could not sign in" description={error} /> : null}

      <Button type="button" label="Continue with Google" variant="primary" size="lg" onClick={continueWithGoogle} />

      <Text type="supporting">By continuing, you agree to our Terms of Service and Privacy Policy.</Text>
    </VStack>
  )
}
