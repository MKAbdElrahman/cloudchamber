import { useState } from 'react'

import { Banner } from '@astryxdesign/core/Banner'
import { VStack } from '@astryxdesign/core/Layout'
import { Heading } from '@astryxdesign/core/Text'
import { Button } from '@astryxdesign/core/Button'

import { SIGN_IN_CONTENT_WIDTH } from '@/shared/layout'
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
    <VStack gap={WORKSPACE_GAP.section} width="100%" maxWidth={SIGN_IN_CONTENT_WIDTH} hAlign="stretch">
      <Heading level={1}>Welcome back</Heading>

      {error ? <Banner status="error" title="Could not sign in" description={error} /> : null}

      <Button type="button" label="Continue with Google" variant="primary" size="lg" onClick={continueWithGoogle} />
    </VStack>
  )
}
