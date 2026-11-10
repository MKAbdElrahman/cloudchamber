import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from 'react'

const STORAGE_KEY = 'bubble.token'
const LEGACY_STORAGE_KEY = 'cloudchamber.token'
const DISPLAY_NAME_KEY = 'bubble.displayName'
const LEGACY_DISPLAY_NAME_KEY = 'cloudchamber.displayName'

type SessionState = {
  token: string | null
  displayName: string | null
  setToken: (token: string | null, displayName?: string | null) => void
  signOut: () => void
}

const SessionContext = createContext<SessionState | null>(null)

function readMigrated(key: string, legacyKey: string): string | null {
  try {
    const current = localStorage.getItem(key)
    if (current) return current
    const legacy = localStorage.getItem(legacyKey)
    if (!legacy) return null
    localStorage.setItem(key, legacy)
    localStorage.removeItem(legacyKey)
    return legacy
  } catch {
    return null
  }
}

function readStoredToken(): string | null {
  return readMigrated(STORAGE_KEY, LEGACY_STORAGE_KEY)
}

function readStoredDisplayName(): string | null {
  return readMigrated(DISPLAY_NAME_KEY, LEGACY_DISPLAY_NAME_KEY)
}

export function SessionProvider({ children }: { children: ReactNode }) {
  const [token, setTokenState] = useState<string | null>(() => readStoredToken())
  const [displayName, setDisplayNameState] = useState<string | null>(() => readStoredDisplayName())

  const setToken = useCallback((next: string | null, nextDisplayName?: string | null) => {
    setTokenState(next)
    try {
      if (next) {
        localStorage.setItem(STORAGE_KEY, next)
        if (nextDisplayName != null && nextDisplayName.trim() !== '') {
          const name = nextDisplayName.trim()
          setDisplayNameState(name)
          localStorage.setItem(DISPLAY_NAME_KEY, name)
        }
      } else {
        localStorage.removeItem(STORAGE_KEY)
        localStorage.removeItem(DISPLAY_NAME_KEY)
        setDisplayNameState(null)
      }
    } catch {
      // ignore storage failures
    }
  }, [])

  const signOut = useCallback(() => setToken(null), [setToken])

  const value = useMemo(
    () => ({ token, displayName, setToken, signOut }),
    [token, displayName, setToken, signOut],
  )

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>
}

export function useSession(): SessionState {
  const ctx = useContext(SessionContext)
  if (!ctx) throw new Error('useSession must be used within SessionProvider')
  return ctx
}
