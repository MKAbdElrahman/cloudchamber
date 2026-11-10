export type WorkspaceSection = 'overview' | 'pipeline' | 'history' | 'settings'

export const WORKSPACE_LABELS: Record<WorkspaceSection, string> = {
  overview: 'Overview',
  pipeline: 'Pipeline',
  history: 'History',
  settings: 'Settings',
}
