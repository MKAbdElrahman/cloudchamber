import { useEffect } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'

import { SignInExperience } from '@/features/auth/SignInExperience'
import { PipelineRunProvider } from '@/features/pipeline/PipelineRunContext'
import { AuthenticatedRoutes } from '@/app/AuthenticatedRoutes'
import { useSession } from '@/shared/auth/session'
import { accountDisplayLabel, parseServiceJwt } from '@/shared/auth/tokenClaims'
import { AppChrome } from '@/shared/layout/AppChrome'
import {
  WORKSPACE_PATHS,
  pathForWorkspaceSection,
  workspaceSectionFromPath,
} from '@/shared/layout/workspacePaths'

function AuthenticatedApp() {
  const navigate = useNavigate()
  const location = useLocation()
  const { token, displayName, signOut } = useSession()

  if (!token) return null

  const section = workspaceSectionFromPath(location.pathname)
  const claims = parseServiceJwt(token)
  const accountName = accountDisplayLabel(claims, displayName)

  return (
    <PipelineRunProvider>
      <AppChrome
        section={section}
        accountName={accountName}
        onSectionChange={(next) => navigate(pathForWorkspaceSection(next))}
        onOpenProfile={() => navigate(WORKSPACE_PATHS.settingsSection('account'))}
        onOpenSettings={() => navigate(WORKSPACE_PATHS.settingsSection('appearance'))}
        onSignOut={signOut}
      >
        <AuthenticatedRoutes />
      </AppChrome>
    </PipelineRunProvider>
  )
}

export default function App() {
  const { token, setToken } = useSession()
  const navigate = useNavigate()
  const location = useLocation()

  useEffect(() => {
    if (token && location.pathname === '/') {
      navigate(WORKSPACE_PATHS.overview, { replace: true })
    }
  }, [token, location.pathname, navigate])

  if (!token) {
    return (
      <SignInExperience
        onAuthenticated={(nextToken, name) => {
          setToken(nextToken, name)
        }}
      />
    )
  }

  return <AuthenticatedApp />
}
