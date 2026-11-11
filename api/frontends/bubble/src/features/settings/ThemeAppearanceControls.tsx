import type { DefinedTheme } from '@astryxdesign/core/theme'
import { Grid } from '@astryxdesign/core/Grid'
import { HStack, StackItem, VStack } from '@astryxdesign/core/Layout'
import { AspectRatio } from '@astryxdesign/core/AspectRatio'
import { Section } from '@astryxdesign/core/Section'
import { SelectableCard } from '@astryxdesign/core/SelectableCard'
import { Skeleton } from '@astryxdesign/core/Skeleton'
import { Icon } from '@astryxdesign/core/Icon'
import { Text } from '@astryxdesign/core/Text'
import { Theme } from '@astryxdesign/core/theme'
import {
  ComputerDesktopIcon,
  MoonIcon,
  SunIcon,
} from '@heroicons/react/24/outline'

import { WORKSPACE_GAP } from '@/shared/spacing'
import { MUTED_ICON_COLOR, type ColorMode } from '@/shared/theme'

function PreviewBar({ wide }: { wide?: boolean }) {
  return <Skeleton width={wide ? '66%' : '50%'} height="0.25rem" radius="rounded" />
}

/** Mini UI chrome inside a nested Theme (settings color-mode swatch). */
function ThemeModeCanvas({
  mode,
  previewTheme,
}: {
  mode: 'light' | 'dark'
  previewTheme: DefinedTheme
}) {
  return (
    <Theme theme={previewTheme} mode={mode}>
      <Section variant="muted" padding={WORKSPACE_GAP.local} height="100%">
        <VStack gap={WORKSPACE_GAP.local} height="100%">
          <VStack gap={WORKSPACE_GAP.local} hAlign="center">
            <PreviewBar wide />
            <PreviewBar />
          </VStack>
          <StackItem size="fill">
            <Section variant="section" padding={WORKSPACE_GAP.local} height="100%">
              <VStack gap={WORKSPACE_GAP.local} vAlign="center" height="100%">
                <PreviewBar />
                <PreviewBar />
                <PreviewBar />
              </VStack>
            </Section>
          </StackItem>
        </VStack>
      </Section>
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
      <VStack aria-hidden height="100%">
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
      </VStack>
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
      gap={WORKSPACE_GAP.group}
      width="100%"
    >
      {COLOR_MODE_CHOICES.map((choice) => {
        const selected = value === choice.value
        return (
          <SelectableCard
            key={choice.value}
            label={choice.label}
            isSelected={selected}
            padding={WORKSPACE_GAP.local}
            onChange={() => {
              if (!selected) onChange(choice.value)
            }}
          >
            <VStack gap={WORKSPACE_GAP.local}>
              <ThemeModePreview scheme={choice.scheme} previewTheme={previewTheme} />
              <HStack gap={WORKSPACE_GAP.local} vAlign="center" hAlign="center">
                <Icon icon={choice.icon} size="sm" color={MUTED_ICON_COLOR} />
                <Text type={selected ? 'label' : 'supporting'}>{choice.label}</Text>
              </HStack>
            </VStack>
          </SelectableCard>
        )
      })}
    </Grid>
  )
}
