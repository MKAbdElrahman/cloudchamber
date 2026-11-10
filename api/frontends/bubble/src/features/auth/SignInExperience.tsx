import type { CSSProperties } from 'react'

import { Center } from '@astryxdesign/core/Center'
import { Card } from '@astryxdesign/core/Card'
import { Grid } from '@astryxdesign/core/Grid'
import { StackItem, VStack } from '@astryxdesign/core/Layout'
import { Section } from '@astryxdesign/core/Section'
import { Text } from '@astryxdesign/core/Text'

import { LoginPanel } from './LoginPanel'

const COLUMN_MIN_WIDTH = 240

const pageStyle: CSSProperties = {
  minHeight: '100%',
  backgroundColor: 'var(--color-background-body)',
}

const cardWrap: CSSProperties = {
  width: '100%',
  maxWidth: 1000,
  marginInline: 'auto',
}

/** Login split — form column + hero panel (`login-split` template). */
export function SignInExperience({
  onAuthenticated,
}: {
  onAuthenticated: (token: string, displayName?: string) => void
}) {
  return (
    <Center axis="both" padding={6} style={pageStyle}>
      <VStack gap={0} width="100%">
        <div style={cardWrap}>
          <Card padding={0} width="100%">
            <Grid
              columns={{ minWidth: COLUMN_MIN_WIDTH, repeat: 'fit' }}
              gap={8}
              align="stretch"
              className="login-split-grid"
            >
              <Section variant="transparent" padding={0} height="100%">
                <VStack gap={4} height="100%">
                  <Text type="body" weight="bold">
                    CloudChamber
                  </Text>
                  <StackItem size="fill">
                    <Center axis="vertical" height="100%" padding={0}>
                      <LoginPanel onAuthenticated={onAuthenticated} />
                    </Center>
                  </StackItem>
                </VStack>
              </Section>

              <div className="login-split-image login-hero-panel" aria-hidden />
            </Grid>
          </Card>
        </div>
      </VStack>
    </Center>
  )
}
