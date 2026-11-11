export const SETTINGS_SECTION_IDS = ['appearance', 'account'] as const

export type SettingsSectionId = (typeof SETTINGS_SECTION_IDS)[number]

export function isSettingsSectionId(value: string | undefined): value is SettingsSectionId {
  return SETTINGS_SECTION_IDS.some((id) => id === value)
}
