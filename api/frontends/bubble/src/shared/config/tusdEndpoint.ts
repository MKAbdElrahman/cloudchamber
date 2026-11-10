import { env } from './env'

const STORAGE_KEY = 'bubble.tusd.endpoint.v1'

/** The endpoint an upload uses. A developer override wins over the built-in one. */
export function readTusdEndpoint(): string {
  const custom = readCustomTusdEndpoint()
  return custom && custom.length > 0 ? custom : env.tusdEndpoint
}

export function readCustomTusdEndpoint(): string | null {
  try {
    const value = localStorage.getItem(STORAGE_KEY)?.trim() ?? ''
    return value.length > 0 ? value : null
  } catch {
    return null
  }
}

export function writeCustomTusdEndpoint(value: string | null) {
  try {
    const next = value?.trim() ?? ''
    if (!next) {
      localStorage.removeItem(STORAGE_KEY)
      return
    }
    localStorage.setItem(STORAGE_KEY, next)
  } catch {
    // The built-in endpoint still works when storage is unavailable.
  }
}
