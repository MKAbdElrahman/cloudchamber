import type { Project, ProjectRecording } from './types'

const STORAGE_KEY = 'bubble.projects.v1'

/** Seed project that receives jobs created before projects existed. */
export const DEFAULT_PROJECT_ID = 'proj-lecture-notes'

type ProjectStore = {
  projects: Project[]
  /** job id → project id */
  jobProject: Record<string, string>
  recordings: ProjectRecording[]
  currentProjectId: string
}

const SEED: ProjectStore = {
  projects: [
    {
      id: DEFAULT_PROJECT_ID,
      name: 'Lecture notes',
      dateCreated: '2026-03-01T09:00:00.000Z',
    },
    {
      id: 'proj-seminar',
      name: 'Seminar recordings',
      dateCreated: '2026-03-04T09:00:00.000Z',
    },
  ],
  jobProject: {
    'job-demo-01': DEFAULT_PROJECT_ID,
    'job-demo-02': DEFAULT_PROJECT_ID,
  },
  recordings: [],
  currentProjectId: DEFAULT_PROJECT_ID,
}

let memory: ProjectStore | null = null

function isProject(value: unknown): value is Project {
  if (!value || typeof value !== 'object') return false
  const project = value as Partial<Project>
  return typeof project.id === 'string' && typeof project.name === 'string' && typeof project.dateCreated === 'string'
}

function parseStore(raw: string): ProjectStore | null {
  const parsed = JSON.parse(raw) as Partial<ProjectStore>
  if (!Array.isArray(parsed.projects)) return null
  const projects = parsed.projects.filter(isProject)
  const jobProject: Record<string, string> = {}
  if (parsed.jobProject && typeof parsed.jobProject === 'object') {
    for (const [jobId, projectId] of Object.entries(parsed.jobProject)) {
      if (typeof projectId === 'string' && projects.some((project) => project.id === projectId)) {
        jobProject[jobId] = projectId
      }
    }
  }
  const current = projects.some((project) => project.id === parsed.currentProjectId)
    ? parsed.currentProjectId!
    : (projects[0]?.id ?? '')
  const recordings = Array.isArray(parsed.recordings) ? parsed.recordings.filter(isRecording) : []
  return { projects, jobProject, recordings, currentProjectId: current }
}

function isRecording(value: unknown): value is ProjectRecording {
  if (!value || typeof value !== 'object') return false
  const row = value as Partial<ProjectRecording>
  return (
    typeof row.id === 'string' &&
    typeof row.projectId === 'string' &&
    typeof row.sourceRef === 'string' &&
    typeof row.name === 'string' &&
    typeof row.dateAdded === 'string'
  )
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
  if (memory) {
    memory.recordings = memory.recordings ?? []
    return memory
  }
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
  writeStore(structuredClone(SEED))
  return memory ?? SEED
}

function update(change: (store: ProjectStore) => void): ProjectStore {
  const next = structuredClone(readStore())
  change(next)
  writeStore(next)
  return next
}

export function readProjects(): Project[] {
  return readStore().projects
}

export function readCurrentProjectId(): string {
  return readStore().currentProjectId
}

export function rememberProject(projectId: string) {
  const store = readStore()
  if (!store.projects.some((project) => project.id === projectId)) return
  if (store.currentProjectId === projectId) return
  update((next) => {
    next.currentProjectId = projectId
  })
}

export function renameProject(projectId: string, name: string) {
  update((next) => {
    const project = next.projects.find((item) => item.id === projectId)
    if (project) project.name = name
  })
}

export function removeProject(projectId: string) {
  update((next) => {
    next.projects = next.projects.filter((project) => project.id !== projectId)
    for (const [jobId, owner] of Object.entries(next.jobProject)) {
      if (owner === projectId) delete next.jobProject[jobId]
    }
    next.recordings = next.recordings.filter((recording) => recording.projectId !== projectId)
    if (next.currentProjectId === projectId) {
      next.currentProjectId = next.projects[0]?.id ?? ''
    }
  })
}

export function insertProject(name: string): Project {
  const project: Project = {
    id: `proj-${crypto.randomUUID()}`,
    name,
    dateCreated: new Date().toISOString(),
  }
  update((next) => {
    next.projects.push(project)
    next.currentProjectId = project.id
  })
  return project
}

export function readProjectIdForJob(jobId: string): string | null {
  return readStore().jobProject[jobId] ?? null
}

export function readProjectJobIds(projectId: string): string[] {
  return Object.entries(readStore().jobProject)
    .filter(([, owner]) => owner === projectId)
    .map(([jobId]) => jobId)
}

export function assignJob(projectId: string, jobId: string) {
  update((next) => {
    next.jobProject[jobId] = projectId
  })
}

export function readProjectRecordings(projectId: string): ProjectRecording[] {
  return readStore()
    .recordings.filter((recording) => recording.projectId === projectId)
    .sort((a, b) => b.dateAdded.localeCompare(a.dateAdded))
}

export function readRecording(projectId: string, recordingId: string): ProjectRecording | null {
  return (
    readStore().recordings.find((recording) => recording.projectId === projectId && recording.id === recordingId) ??
    null
  )
}

export function insertRecording(projectId: string, input: { sourceRef: string; name: string }): ProjectRecording {
  const recording: ProjectRecording = {
    id: `rec-${crypto.randomUUID()}`,
    projectId,
    sourceRef: input.sourceRef,
    name: input.name.trim() || 'Recording',
    dateAdded: new Date().toISOString(),
  }
  update((next) => {
    next.recordings.push(recording)
  })
  return recording
}

export function linkRecordingJob(recordingId: string, jobId: string) {
  update((next) => {
    const recording = next.recordings.find((item) => item.id === recordingId)
    if (recording) recording.jobId = jobId
  })
}

/** Jobs with no project yet join the fallback project. Assigned jobs stay put. */
export function claimUnassigned(jobIds: readonly string[], fallbackProjectId: string) {
  update((next) => {
    if (!next.projects.some((project) => project.id === fallbackProjectId)) return
    for (const jobId of jobIds) {
      if (!jobId || next.jobProject[jobId]) continue
      next.jobProject[jobId] = fallbackProjectId
    }
  })
}
