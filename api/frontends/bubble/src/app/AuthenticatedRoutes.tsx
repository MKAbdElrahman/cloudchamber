import type { ReactNode } from 'react'
import { Navigate, Route, Routes, useParams } from 'react-router-dom'

import { readCurrentProjectId, useProjects } from '@/features/projects'
import { JobFilePage, ProjectHomePage, ProjectsHomePage, SettingsPanel } from '@/pages'
import { useSession } from '@/shared/auth'
import { PendingLines, WorkspacePage, WORKSPACE_PATHS } from '@/shared/layout'

function ProjectHomeRedirect() {
  const { projectId = '' } = useParams()
  return <Navigate to={WORKSPACE_PATHS.project(projectId)} replace />
}

function LegacyJobRedirect() {
  const { jobId = '' } = useParams()
  const projectId = readCurrentProjectId()
  return <Navigate to={WORKSPACE_PATHS.job(projectId, jobId)} replace />
}

function ProjectReadRedirect() {
  const { projectId = '', jobId = '' } = useParams()
  return <Navigate to={WORKSPACE_PATHS.job(projectId, jobId)} replace />
}

function ProjectScreen({ children }: { children: ReactNode }) {
  const { projectId = '' } = useParams()
  const projects = useProjects()
  if (projects.isPending) {
    return (
      <WorkspacePage>
        <PendingLines />
      </WorkspacePage>
    )
  }
  const known = projects.data?.some((project) => project.id === projectId)
  if (!known) return <Navigate to={WORKSPACE_PATHS.home} replace />
  return children
}

function AddRecordingRedirect() {
  const { projectId = '' } = useParams()
  return <Navigate to={`${WORKSPACE_PATHS.project(projectId)}?add=recording`} replace />
}

/** Authenticated workspace routes — eager panels to avoid route-level loading chrome. */
export function AuthenticatedRoutes() {
  const { token } = useSession()
  if (!token) return null

  return (
    <Routes>
      <Route path="/" element={<ProjectsHomePage token={token} />} />
      <Route path="/overview" element={<Navigate to={WORKSPACE_PATHS.home} replace />} />
      <Route path="/pipeline" element={<Navigate to={WORKSPACE_PATHS.home} replace />} />
      <Route path="/history/:jobId/read" element={<LegacyJobRedirect />} />
      <Route path="/history/:jobId" element={<LegacyJobRedirect />} />
      <Route path="/history" element={<Navigate to={WORKSPACE_PATHS.home} replace />} />
      <Route path="/p/:projectId/overview" element={<ProjectHomeRedirect />} />
      <Route path="/p/:projectId/pipeline" element={<ProjectScreen><AddRecordingRedirect /></ProjectScreen>} />
      <Route path="/p/:projectId/history" element={<ProjectHomeRedirect />} />
      <Route path="/p/:projectId/history/:jobId/read" element={<ProjectReadRedirect />} />
      <Route path="/p/:projectId/history/:jobId" element={<ProjectReadRedirect />} />
      <Route path="/p/:projectId/files/:jobId/:kind" element={<ProjectReadRedirect />} />
      <Route path="/p/:projectId/jobs/:jobId" element={<ProjectScreen><JobFilePage token={token} /></ProjectScreen>} />
      <Route path="/p/:projectId/recordings/:recordingId" element={<ProjectScreen><JobFilePage token={token} /></ProjectScreen>} />
      <Route path="/p/:projectId" element={<ProjectScreen><ProjectHomePage token={token} /></ProjectScreen>} />
      <Route path="/settings" element={<Navigate to={WORKSPACE_PATHS.settingsSection('appearance')} replace />} />
      <Route path="/settings/:section" element={<SettingsPanel />} />
      <Route path="*" element={<Navigate to={WORKSPACE_PATHS.home} replace />} />
    </Routes>
  )
}
