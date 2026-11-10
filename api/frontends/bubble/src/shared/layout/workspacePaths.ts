import type { SettingsSectionId, WorkspaceSection } from './workspace'

function projectPrefix(projectId: string): string {
  return `/p/${encodeURIComponent(projectId)}`
}

export const WORKSPACE_PATHS = {
  home: '/',
  project: (projectId: string) => projectPrefix(projectId),
  pipeline: (projectId: string) => `${projectPrefix(projectId)}/pipeline`,
  recording: (projectId: string, recordingId: string) =>
    `${projectPrefix(projectId)}/recordings/${encodeURIComponent(recordingId)}`,
  job: (projectId: string, jobId: string) =>
    `${projectPrefix(projectId)}/jobs/${encodeURIComponent(jobId)}`,
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
