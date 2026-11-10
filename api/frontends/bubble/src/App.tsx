import { useEffect } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'

import { SignInExperience } from '@/features/auth'
import { rememberProject } from '@/features/projects'
import { AuthenticatedRoutes } from '@/app/AuthenticatedRoutes'
import { accountDisplayLabel, parseServiceJwt, useSession } from '@/shared/auth'
import { AppChrome, projectIdFromPath, WORKSPACE_PATHS, workspaceSectionFromPath } from '@/shared/layout'

function AuthenticatedApp() {
  const navigate = useNavigate()
  const location = useLocation()
  const { token, displayName, signOut } = useSession()

  if (!token) return null

  const section = workspaceSectionFromPath(location.pathname)
  const claims = parseServiceJwt(token)
  const accountName = accountDisplayLabel(claims, displayName)

  return (
    <AppChrome
      section={section}
      accountName={accountName}
      onOpenProjects={() => navigate(WORKSPACE_PATHS.home)}
      onOpenProfile={() => navigate(WORKSPACE_PATHS.settingsSection('account'))}
      onOpenSettings={() => navigate(WORKSPACE_PATHS.settingsSection('appearance'))}
      onSignOut={signOut}
    >
      <AuthenticatedRoutes />
    </AppChrome>
  )
}

export default function App() {
  const { token, setToken } = useSession()
  const location = useLocation()

  useEffect(() => {
    const routeProjectId = projectIdFromPath(location.pathname)
    if (routeProjectId) rememberProject(routeProjectId)
  }, [location.pathname])

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
