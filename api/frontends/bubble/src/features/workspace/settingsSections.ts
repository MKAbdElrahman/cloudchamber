export type SettingsSectionId = 'appearance' | 'account' | 'developer'

export function isSettingsSectionId(value: string | undefined): value is SettingsSectionId {
  return value === 'appearance' || value === 'account' || value === 'developer'
}
