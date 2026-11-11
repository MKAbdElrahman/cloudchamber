import { useEffect } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'

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
  const navigate = useNavigate()
  const handedToken = new URLSearchParams(location.search).get('token')
  const handedName = new URLSearchParams(location.search).get('name') ?? undefined

  useEffect(() => {
    if (!handedToken) return
    if (!token) setToken(handedToken, handedName)
    const params = new URLSearchParams(location.search)
    params.delete('token')
    params.delete('name')
    const query = params.toString()
    navigate(query ? `${location.pathname}?${query}` : location.pathname, { replace: true })
  }, [handedName, handedToken, location.pathname, location.search, navigate, setToken, token])

  useEffect(() => {
    const routeProjectId = projectIdFromPath(location.pathname)
    if (routeProjectId) rememberProject(routeProjectId)
  }, [location.pathname])

  if (!token) {
    return (
      <SignInExperience />
    )
  }

  return <AuthenticatedApp />
}
