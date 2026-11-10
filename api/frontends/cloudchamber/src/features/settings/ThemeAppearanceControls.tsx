import type { CSSProperties } from 'react'

import type { DefinedTheme } from '@astryxdesign/core/theme'
import { Grid } from '@astryxdesign/core/Grid'
import { HStack, StackItem, VStack } from '@astryxdesign/core/Layout'
import { AspectRatio } from '@astryxdesign/core/AspectRatio'
import { SelectableCard } from '@astryxdesign/core/SelectableCard'
import { Icon } from '@astryxdesign/core/Icon'
import { Text } from '@astryxdesign/core/Text'
import { Theme } from '@astryxdesign/core/theme'
import {
  ComputerDesktopIcon,
  MoonIcon,
  SunIcon,
} from '@heroicons/react/24/outline'

import type { ColorMode } from '@/app/ThemePreference'

const previewSurface: CSSProperties = {
  backgroundColor: 'var(--color-background-muted)',
  borderRadius: 'var(--radius-sm)',
  height: '100%',
  padding: 8,
  boxSizing: 'border-box',
}

function PreviewBar({ width, accent }: { width: string; accent?: boolean }) {
  return (
    <VStack
      aria-hidden
      style={{
        height: accent ? 6 : 4,
        width,
        maxWidth: '100%',
        borderRadius: 4,
        backgroundColor: accent ? 'var(--color-accent)' : 'var(--color-border)',
      }}
    />
  )
}

/** Mini UI chrome inside a nested Theme (Meta settings-dialog swatch pattern). */
function ThemeModeCanvas({
  mode,
  previewTheme,
}: {
  mode: 'light' | 'dark'
  previewTheme: DefinedTheme
}) {
  return (
    <Theme theme={previewTheme} mode={mode}>
      <VStack gap={1.5} style={previewSurface}>
        <VStack gap={1} hAlign="center">
          <PreviewBar width="72%" accent />
          <PreviewBar width="48%" />
        </VStack>
        <VStack
          gap={1}
          style={{
            flex: 1,
            background: 'var(--color-background-card)',
            padding: 6,
            borderRadius: 'var(--radius-sm)',
          }}
        >
          <PreviewBar width="100%" />
          <PreviewBar width="88%" />
          <PreviewBar width="76%" />
        </VStack>
      </VStack>
    </Theme>
  )
}

function ThemeModePreview({
  scheme,
  previewTheme,
}: {
  scheme: 'light' | 'dark' | 'system'
  previewTheme: DefinedTheme
}) {
  return (
    <AspectRatio ratio={3 / 2}>
      {scheme === 'system' ? (
        <HStack gap={0} height="100%">
          <StackItem size="fill">
            <ThemeModeCanvas mode="light" previewTheme={previewTheme} />
          </StackItem>
          <StackItem size="fill">
            <ThemeModeCanvas mode="dark" previewTheme={previewTheme} />
          </StackItem>
        </HStack>
      ) : (
        <ThemeModeCanvas mode={scheme} previewTheme={previewTheme} />
      )}
    </AspectRatio>
  )
}

const COLOR_MODE_CHOICES: {
  value: ColorMode
  label: string
  scheme: 'light' | 'dark' | 'system'
  icon: typeof SunIcon
}[] = [
  { value: 'light', label: 'Light', scheme: 'light', icon: SunIcon },
  { value: 'dark', label: 'Dark', scheme: 'dark', icon: MoonIcon },
  { value: 'system', label: 'System', scheme: 'system', icon: ComputerDesktopIcon },
]

export function ColorModeChoiceCards({
  value,
  onChange,
  previewTheme,
}: {
  value: ColorMode
  onChange: (mode: ColorMode) => void
  previewTheme: DefinedTheme
}) {
  return (
    <Grid
      role="radiogroup"
      aria-label="Color mode"
      columns={{ minWidth: 96, max: 3 }}
      gap={2}
      maxWidth={480}
    >
      {COLOR_MODE_CHOICES.map((choice) => {
        const selected = value === choice.value
        return (
          <SelectableCard
            key={choice.value}
            label={choice.label}
            isSelected={selected}
            padding={2}
            onChange={() => {
              if (!selected) onChange(choice.value)
            }}
          >
            <VStack gap={1.5}>
              <ThemeModePreview scheme={choice.scheme} previewTheme={previewTheme} />
              <HStack gap={1} vAlign="center" hAlign="center">
                <Icon icon={choice.icon} size="sm" color="secondary" />
                <Text type="supporting" weight={selected ? 'semibold' : undefined}>
                  {choice.label}
                </Text>
              </HStack>
            </VStack>
          </SelectableCard>
        )
      })}
    </Grid>
  )
}
