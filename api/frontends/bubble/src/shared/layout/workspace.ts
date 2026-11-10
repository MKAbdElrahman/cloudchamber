export type WorkspaceSection = 'projects' | 'settings'

/** Route ids for the settings screen. Shared so the shell does not import a feature. */
export type SettingsSectionId = 'appearance' | 'account' | 'developer'

export function isSettingsSectionId(value: string | undefined): value is SettingsSectionId {
  return value === 'appearance' || value === 'account' || value === 'developer'
}
