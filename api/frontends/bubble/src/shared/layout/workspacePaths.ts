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
  historyJob: (projectId: string, jobId: string) =>
    `${projectPrefix(projectId)}/history/${encodeURIComponent(jobId)}`,
  projectFile: (projectId: string, jobId: string, kind: string) =>
    `${projectPrefix(projectId)}/files/${encodeURIComponent(jobId)}/${encodeURIComponent(kind)}`,
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
