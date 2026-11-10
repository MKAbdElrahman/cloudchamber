import { ApiError } from '@/shared/api'
import { env } from '@/shared/config'

type TokenResponse = { token: string }

export async function fetchToken(params: {
  authBase: string
  kid: string
  username: string
  password: string
}): Promise<string> {
  const base = params.authBase.replace(/\/$/, '')
  const res = await fetch(`${base}/v1/auth/token/${encodeURIComponent(params.kid)}`, {
    method: 'GET',
    headers: {
      Authorization: `Basic ${btoa(`${params.username}:${params.password}`)}`,
    },
  })
  const text = await res.text()
  let body: unknown = null
  try {
    body = text ? JSON.parse(text) : null
  } catch {
    body = text
  }
  if (!res.ok) {
    const msg =
      body && typeof body === 'object' && (body as { message?: string }).message
        ? (body as { message: string }).message
        : `Authentication failed (${res.status})`
    throw new ApiError(msg, res.status, body)
  }
  const data = body as TokenResponse
  if (!data?.token) throw new Error('Auth response missing token')
  return data.token
}

export const authDefaults = {
  authBase: env.authApiBase,
  kid: env.authKid,
}
