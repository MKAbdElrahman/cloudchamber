import { Grid } from '@astryxdesign/core/Grid'
import { VStack } from '@astryxdesign/core/Layout'
import { Section } from '@astryxdesign/core/Section'
import { Text } from '@astryxdesign/core/Text'

import { LoginPanel } from './LoginPanel'

// max lets the last column shrink under the floor, so a phone gets one
// full-width column instead of a 320px track that overflows the viewport.
const COLUMN_MIN_WIDTH = 320

/**
 * Sign-in fills the viewport. At two columns the hero covers its half;
 * below ~640px the grid drops to one column and the form stays on screen.
 */
export function SignInExperience({
  onAuthenticated,
}: {
  onAuthenticated: (token: string, displayName?: string) => void
}) {
  return (
    <Grid
      columns={{ minWidth: COLUMN_MIN_WIDTH, max: 2 }}
      gap={0}
      align="stretch"
      width="100%"
      height="100%"
    >
      <Section variant="transparent" padding={6} height="100%">
        <VStack gap={4} height="100%" vAlign="center" hAlign="center" width="100%">
          <VStack gap={4} width="100%" maxWidth={480}>
            <Text type="body" weight="bold">
              CloudChamber
            </Text>
            <LoginPanel onAuthenticated={onAuthenticated} />
          </VStack>
        </VStack>
      </Section>
      <Section padding={0} height="100%" className="relative overflow-hidden">
        <img src="/cloud-chamber.jpg" alt="" className="absolute inset-0 h-full w-full object-cover" />
      </Section>
    </Grid>
  )
}
