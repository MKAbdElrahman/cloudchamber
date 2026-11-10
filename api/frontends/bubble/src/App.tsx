import { useEffect } from 'react'
import { useLocation } from 'react-router-dom'

import { SignInExperience } from '@/features/auth'
import { rememberProject } from '@/features/projects'
import { AuthenticatedRoutes } from '@/app/AuthenticatedRoutes'
import { accountDisplayLabel, parseServiceJwt, useSession } from '@/shared/auth'
import { AppChrome, projectIdFromPath, workspaceSectionFromPath } from '@/shared/layout'

function AuthenticatedApp() {
  const location = useLocation()
  const { token, displayName, signOut } = useSession()

  if (!token) return null

  const section = workspaceSectionFromPath(location.pathname)
  const claims = parseServiceJwt(token)
  const accountName = accountDisplayLabel(claims, displayName)

  return (
    <AppChrome section={section} accountName={accountName} onSignOut={signOut}>
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
