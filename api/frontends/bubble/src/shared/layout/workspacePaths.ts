import type { SettingsSectionId, WorkspaceSection } from './workspace'

export const WORKSPACE_PATHS = {
  overview: '/overview',
  pipeline: '/pipeline',
  history: '/history',
  historyJob: (jobId: string) => `/history/${encodeURIComponent(jobId)}`,
  historyJobDocument: (jobId: string) => `/history/${encodeURIComponent(jobId)}/read`,
  settings: '/settings',
  settingsSection: (section: SettingsSectionId) => `/settings/${section}`,
} as const

export function workspaceSectionFromPath(pathname: string): WorkspaceSection {
  if (pathname.startsWith('/pipeline')) return 'pipeline'
  if (pathname.startsWith('/history')) return 'history'
  if (pathname.startsWith('/settings')) return 'settings'
  if (pathname.startsWith('/overview')) return 'overview'
  return 'overview'
}

export function pathForWorkspaceSection(section: WorkspaceSection): string {
  switch (section) {
    case 'pipeline':
      return WORKSPACE_PATHS.pipeline
    case 'history':
      return WORKSPACE_PATHS.history
    case 'settings':
      return WORKSPACE_PATHS.settingsSection('appearance')
    default:
      return WORKSPACE_PATHS.overview
  }
}
