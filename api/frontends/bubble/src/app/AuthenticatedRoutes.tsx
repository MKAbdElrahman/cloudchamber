import { Navigate, Route, Routes } from 'react-router-dom'

import { PipelineRunProvider } from '@/features/pipeline'
import { HistoryJobPage, HistoryPanel, OverviewPanel, PipelineWorkspace, SettingsPanel } from '@/pages'
import { useSession } from '@/shared/auth'
import { WORKSPACE_PATHS } from '@/shared/layout'

/** Authenticated workspace routes — eager panels to avoid route-level loading chrome. */
export function AuthenticatedRoutes() {
  const { token } = useSession()
  if (!token) return null

  return (
    <Routes>
      <Route path="/" element={<Navigate to={WORKSPACE_PATHS.overview} replace />} />
      <Route path={WORKSPACE_PATHS.overview} element={<OverviewPanel />} />
      <Route
        path={WORKSPACE_PATHS.pipeline}
        element={
          <PipelineRunProvider>
            <PipelineWorkspace token={token} />
          </PipelineRunProvider>
        }
      />
      <Route path={WORKSPACE_PATHS.history} element={<HistoryPanel token={token} />} />
      <Route path={`${WORKSPACE_PATHS.history}/:jobId`} element={<HistoryJobPage token={token} />} />
      <Route path="/settings" element={<Navigate to={WORKSPACE_PATHS.settingsSection('appearance')} replace />} />
      <Route path="/settings/:section" element={<SettingsPanel />} />
      <Route path="*" element={<Navigate to={WORKSPACE_PATHS.overview} replace />} />
    </Routes>
  )
}
