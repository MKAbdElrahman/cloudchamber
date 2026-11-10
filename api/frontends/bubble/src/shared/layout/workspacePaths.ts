import type { SettingsSectionId, WorkspaceSection } from './workspace'

function projectPrefix(projectId: string): string {
  return `/p/${encodeURIComponent(projectId)}`
}

export const WORKSPACE_PATHS = {
  home: '/',
  project: (projectId: string) => projectPrefix(projectId),
  pipeline: (projectId: string) => `${projectPrefix(projectId)}/pipeline`,
  historyJob: (projectId: string, jobId: string) =>
    `${projectPrefix(projectId)}/history/${encodeURIComponent(jobId)}`,
  historyJobDocument: (projectId: string, jobId: string) =>
    `${projectPrefix(projectId)}/history/${encodeURIComponent(jobId)}/read`,
  settings: '/settings',
  settingsSection: (section: SettingsSectionId) => `/settings/${section}`,
} as const

export function projectIdFromPath(pathname: string): string | null {
  const match = /^\/p\/([^/]+)/.exec(pathname)
  if (!match) return null
  return decodeURIComponent(match[1])
}

export function workspaceSectionFromPath(pathname: string): WorkspaceSection {
  if (pathname.startsWith('/settings')) return 'settings'
  return 'projects'
}
