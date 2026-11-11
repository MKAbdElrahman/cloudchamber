import { apiFetch } from '@/shared/api'

export type AccountUser = {
  id: string
  name: string
  email: string
  roles: string[]
  department: string
  enabled: boolean
  dateCreated: string
}

export function fetchAccount(token: string, userId: string): Promise<AccountUser> {
  return apiFetch<AccountUser>(`/v1/users/${encodeURIComponent(userId)}`, { token })
}
