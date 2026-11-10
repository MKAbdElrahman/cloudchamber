import { Navigate, Route, Routes } from 'react-router-dom'

import { SettingsPanel } from '@/features/settings'
import { HistoryJobPage, HistoryPanel, OverviewPanel, PipelineWorkspace } from '@/pages'
import { useSession } from '@/shared/auth/session'
import { WORKSPACE_PATHS } from '@/shared/layout/workspacePaths'

/** Authenticated workspace routes — eager panels to avoid route-level loading chrome. */
export function AuthenticatedRoutes() {
  const { token } = useSession()
  if (!token) return null

  return (
    <Routes>
      <Route path="/" element={<Navigate to={WORKSPACE_PATHS.overview} replace />} />
      <Route path={WORKSPACE_PATHS.overview} element={<OverviewPanel />} />
      <Route path={WORKSPACE_PATHS.pipeline} element={<PipelineWorkspace token={token} />} />
      <Route path={WORKSPACE_PATHS.history} element={<HistoryPanel token={token} />} />
      <Route path={`${WORKSPACE_PATHS.history}/:jobId`} element={<HistoryJobPage token={token} />} />
      <Route path="/settings" element={<Navigate to={WORKSPACE_PATHS.settingsSection('appearance')} replace />} />
      <Route path="/settings/:section" element={<SettingsPanel />} />
      <Route path="*" element={<Navigate to={WORKSPACE_PATHS.overview} replace />} />
    </Routes>
  )
}
