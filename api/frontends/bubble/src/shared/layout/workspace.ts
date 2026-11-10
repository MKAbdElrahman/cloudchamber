export type WorkspaceSection = 'overview' | 'pipeline' | 'history' | 'settings'

/** Route ids for the settings screen. Shared so the shell does not import a feature. */
export type SettingsSectionId = 'appearance' | 'account' | 'developer'

export function isSettingsSectionId(value: string | undefined): value is SettingsSectionId {
  return value === 'appearance' || value === 'account' || value === 'developer'
}

export const WORKSPACE_LABELS: Record<WorkspaceSection, string> = {
  overview: 'Overview',
  pipeline: 'Pipeline',
  history: 'History',
  settings: 'Settings',
}
