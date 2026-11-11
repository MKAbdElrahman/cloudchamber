import { apiFetch } from '@/shared/api'

import { readProjects, removeProject, replaceProjectId } from './store'
import type { Project, ProjectFile } from './types'

const MIGRATED_KEY = 'bubble.projects.migrated.v1'
const PAGE_ROWS = 100

type ProjectResponse = {
  id: string
  ownerID: string
  name: string
  dateCreated: string
  dateUpdated: string
}

type ProjectsQuery = {
  items?: ProjectResponse[]
  total?: number
  page?: number
  rowsPerPage?: number
}

function toProject(row: ProjectResponse): Project {
  return {
    id: row.id,
    ownerID: row.ownerID,
    name: row.name,
    dateCreated: row.dateCreated,
    dateUpdated: row.dateUpdated,
  }
}

function sameName(left: string, right: string): boolean {
  return left.trim().toLocaleLowerCase() === right.trim().toLocaleLowerCase()
}

/** The sentence to show when creating or renaming a project fails. */
export function projectErrorMessage(err: unknown): string {
  const message = err instanceof Error ? err.message : 'Could not save the project.'
  const start = message.indexOf('[')
  if (start >= 0) {
    try {
      const fields = JSON.parse(message.slice(start)) as { error?: string }[]
      const text = fields.find((item) => item.error)?.error
      if (text?.includes('invalid title')) return 'Use a title of 1 to 80 characters.'
      if (text) return text
    } catch {
      /* The message is not a field-error list. */
    }
  }
  return message
}

async function fetchPage(token: string, page: number): Promise<ProjectsQuery> {
  const query = new URLSearchParams({
    page: String(page),
    rows: String(PAGE_ROWS),
    orderBy: 'date_created,DESC',
  })
  return apiFetch<ProjectsQuery>(`/v1/projects?${query}`, { method: 'GET', token })
}

async function fetchAll(token: string): Promise<Project[]> {
  const first = await fetchPage(token, 1)
  const rows = [...(first.items ?? [])]
  const total = first.total ?? rows.length
  let page = 1
  while (rows.length < total && page < 20) {
    page += 1
    const next = await fetchPage(token, page)
    const items = next.items ?? []
    if (items.length === 0) break
    rows.push(...items)
  }
  return rows.map(toProject)
}

async function createOnServer(token: string, name: string): Promise<Project> {
  const row = await apiFetch<ProjectResponse>('/v1/projects', {
    method: 'POST',
    token,
    body: JSON.stringify({ name }),
  })
  return toProject(row)
}

/**
 * Browser projects were the stand-in list. Copy each one onto the server once,
 * matching an existing title so a retry does not create a second copy.
 */
async function migrateLocalProjects(token: string, server: Project[]) {
  const known = [...server]
  for (const project of readProjects()) {
    const existing = known.find((item) => sameName(item.name, project.name))
    const next = existing ?? (await createOnServer(token, project.name))
    if (!existing) known.push(next)
    replaceProjectId(project.id, next.id)
  }
}

let inflight: { token: string; promise: Promise<Project[]> } | null = null

async function loadProjects(token: string): Promise<Project[]> {
  const server = await fetchAll(token)
  if (typeof localStorage === 'undefined' || localStorage.getItem(MIGRATED_KEY)) return server
  await migrateLocalProjects(token, server)
  localStorage.setItem(MIGRATED_KEY, '1')
  return fetchAll(token)
}

export function listProjects(token: string): Promise<Project[]> {
  if (inflight?.token === token) return inflight.promise
  const promise = loadProjects(token).finally(() => {
    if (inflight?.promise === promise) inflight = null
  })
  inflight = { token, promise }
  return promise
}

export function createProject(token: string, name: string): Promise<Project> {
  const trimmed = name.trim()
  if (!trimmed) return Promise.reject(new Error('Project name is required.'))
  return createOnServer(token, trimmed)
}

export function updateProject(token: string, projectId: string, name: string): Promise<Project> {
  const trimmed = name.trim()
  if (!trimmed) return Promise.reject(new Error('Project name is required.'))
  return apiFetch<ProjectResponse>(`/v1/projects/${encodeURIComponent(projectId)}`, {
    method: 'PUT',
    token,
    body: JSON.stringify({ name: trimmed }),
  }).then(toProject)
}

export async function deleteProject(token: string, projectId: string): Promise<void> {
  await apiFetch<null>(`/v1/projects/${encodeURIComponent(projectId)}`, {
    method: 'DELETE',
    token,
  })
  removeProject(projectId)
}

export function listProjectFiles(token: string, projectId: string) {
  return apiFetch<{ items?: ProjectFile[] }>(`/v1/projects/${encodeURIComponent(projectId)}/files`, {
    method: 'GET',
    token,
  }).then((data) => data.items ?? [])
}

