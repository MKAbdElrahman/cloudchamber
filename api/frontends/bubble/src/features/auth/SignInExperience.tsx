import { VStack } from '@astryxdesign/core/Layout'
import { Section } from '@astryxdesign/core/Section'

import { WORKSPACE_CONTENT_PADDING } from '@/shared/layout'

import { LoginPanel } from './LoginPanel'

/**
 * Sign-in fills the viewport with one column: the Google action.
 */
export function SignInExperience() {
  return (
    <Section variant="transparent" padding={WORKSPACE_CONTENT_PADDING} height="100%">
      <VStack height="100%" vAlign="center" hAlign="center" width="100%">
        <LoginPanel />
      </VStack>
    </Section>
  )
}
