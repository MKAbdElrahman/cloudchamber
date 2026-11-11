import { Grid } from '@astryxdesign/core/Grid'
import { VStack } from '@astryxdesign/core/Layout'
import { Section } from '@astryxdesign/core/Section'

import { WORKSPACE_CONTENT_PADDING } from '@/shared/layout'

import { LoginPanel } from './LoginPanel'

// Grid's minWidth is a px floor. 320 keeps two columns from clipping a phone
// once the form column is allowed to shrink under that floor.
const COLUMN_MIN_WIDTH = 320

/**
 * Sign-in fills the viewport. At two columns the hero covers its half;
 * below the two-column floor the grid drops to one column and the form stays on screen.
 */
export function SignInExperience() {
  return (
    <Grid
      columns={{ minWidth: COLUMN_MIN_WIDTH, max: 2 }}
      gap={0}
      align="stretch"
      width="100%"
      height="100%"
    >
      <Section variant="transparent" padding={WORKSPACE_CONTENT_PADDING} height="100%">
        <VStack height="100%" vAlign="center" hAlign="center" width="100%">
          <LoginPanel />
        </VStack>
      </Section>
      <Section padding={0} height="100%" className="relative overflow-hidden">
        <img src="/cloud-chamber.jpg" alt="" className="absolute inset-0 h-full w-full object-cover" />
      </Section>
    </Grid>
  )
}
