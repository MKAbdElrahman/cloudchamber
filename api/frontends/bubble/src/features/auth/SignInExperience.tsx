import { AppShell } from '@astryxdesign/core/AppShell'
import { VStack } from '@astryxdesign/core/Layout'

import { WORKSPACE_CONTENT_PADDING } from '@/shared/layout'

import { LoginPanel } from './LoginPanel'

/**
 * Sign-in fills the viewport with one column: the Google action.
 * AppShell owns the page color and the viewport height.
 */
export function SignInExperience() {
  return (
    <AppShell height="fill" contentPadding={WORKSPACE_CONTENT_PADDING} variant="elevated">
      <VStack height="100%" vAlign="center" hAlign="center" width="100%">
        <LoginPanel />
      </VStack>
    </AppShell>
  )
}
