import { createContext, useContext } from 'react'

export type SessionState = {
  token: string | null
  displayName: string | null
  setToken: (token: string | null, displayName?: string | null) => void
  signOut: () => void
}

export const SessionContext = createContext<SessionState | null>(null)

export function useSession(): SessionState {
  const ctx = useContext(SessionContext)
  if (!ctx) throw new Error('useSession must be used within SessionProvider')
  return ctx
}
