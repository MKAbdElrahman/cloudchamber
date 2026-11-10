import { env } from '@/shared/config/env'

export class ApiError extends Error {
  status: number
  body: unknown

  constructor(message: string, status: number, body: unknown) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.body = body
  }
}

async function parseBody(res: Response): Promise<unknown> {
  const text = await res.text()
  if (!text) return null
  try {
    return JSON.parse(text)
  } catch {
    return text
  }
}

function errorMessage(body: unknown, fallback: string): string {
  if (body && typeof body === 'object') {
    const b = body as Record<string, unknown>
    const err = b.error as Record<string, unknown> | undefined
    if (typeof b.message === 'string') return b.message
    if (err && typeof err.message === 'string') return err.message
  }
  if (typeof body === 'string' && body) return body
  return fallback
}

export async function apiFetch<T>(
  path: string,
  init: RequestInit & { token?: string | null; base?: string } = {},
): Promise<T> {
  const { token, base = env.cloudchamberApiBase, headers, ...rest } = init
  const res = await fetch(`${base}${path}`, {
    ...rest,
    headers: {
      ...(rest.body ? { 'Content-Type': 'application/json' } : {}),
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(headers ?? {}),
    },
  })
  const body = await parseBody(res)
  if (!res.ok) {
    throw new ApiError(errorMessage(body, `Request failed (${res.status})`), res.status, body)
  }
  return body as T
}
