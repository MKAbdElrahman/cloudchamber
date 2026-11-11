const STORAGE_KEY = 'bubble.projects.v1'

/** A project saved in the browser before the server owned the list. */
export type StoredProject = {
  id: string
  name: string
  dateCreated: string
}

type ProjectStore = {
  projects: StoredProject[]
  currentProjectId: string
}

const EMPTY: ProjectStore = {
  projects: [],
  currentProjectId: '',
}

let memory: ProjectStore | null = null

function isProject(value: unknown): value is StoredProject {
  if (!value || typeof value !== 'object') return false
  const project = value as Partial<StoredProject>
  return typeof project.id === 'string' && typeof project.name === 'string' && typeof project.dateCreated === 'string'
}

function parseStore(raw: string): ProjectStore | null {
  const parsed = JSON.parse(raw) as Partial<ProjectStore>
  if (!Array.isArray(parsed.projects)) return null
  const projects = parsed.projects.filter(isProject)
  const current = typeof parsed.currentProjectId === 'string' ? parsed.currentProjectId : ''
  return { projects, currentProjectId: current }
}

function writeStore(store: ProjectStore) {
  memory = store
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(store))
  } catch {
    /* ignore */
  }
}

function readStore(): ProjectStore {
  if (memory) return memory
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) {
      const parsed = parseStore(raw)
      if (parsed) {
        memory = parsed
        return parsed
      }
    }
  } catch {
    /* seed below */
  }
  writeStore(structuredClone(EMPTY))
  return memory ?? EMPTY
}

function update(change: (store: ProjectStore) => void): ProjectStore {
  const next = structuredClone(readStore())
  change(next)
  writeStore(next)
  return next
}

export function readProjects(): StoredProject[] {
  return readStore().projects
}

export function readCurrentProjectId(): string {
  return readStore().currentProjectId
}

export function rememberProject(projectId: string) {
  if (!projectId) return
  if (readStore().currentProjectId === projectId) return
  update((next) => {
    next.currentProjectId = projectId
  })
}

/** Point the open project at the server id, and drop the local copy of that project. */
export function replaceProjectId(from: string, to: string) {
  update((next) => {
    if (next.currentProjectId === from) next.currentProjectId = to
    next.projects = next.projects.filter((project) => project.id !== from)
  })
}

export function removeProject(projectId: string) {
  update((next) => {
    next.projects = next.projects.filter((project) => project.id !== projectId)
    if (next.currentProjectId === projectId) next.currentProjectId = ''
  })
}
