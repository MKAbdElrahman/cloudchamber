import { fileIdFromSourceRef } from '@/shared/media'

import type { ProjectRecording } from './types'

const STORAGE_KEY = 'bubble.projects.v1'

/** A project saved in the browser before the server owned the list. */
export type StoredProject = {
  id: string
  name: string
  dateCreated: string
}

type ProjectStore = {
  projects: StoredProject[]
  /** job id → project id */
  jobProject: Record<string, string>
  recordings: ProjectRecording[]
  currentProjectId: string
}

const EMPTY: ProjectStore = {
  projects: [],
  jobProject: {},
  recordings: [],
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
  const jobProject: Record<string, string> = {}
  if (parsed.jobProject && typeof parsed.jobProject === 'object') {
    for (const [jobId, projectId] of Object.entries(parsed.jobProject)) {
      if (typeof projectId === 'string' && projectId) jobProject[jobId] = projectId
    }
  }
  const current = typeof parsed.currentProjectId === 'string' ? parsed.currentProjectId : ''
  const recordings = Array.isArray(parsed.recordings) ? parsed.recordings.filter(isRecording).map(withFileId) : []
  return { projects, jobProject, recordings, currentProjectId: current }
}

function isRecording(value: unknown): value is ProjectRecording {
  if (!value || typeof value !== 'object') return false
  const row = value as Partial<ProjectRecording>
  return (
    typeof row.id === 'string' &&
    typeof row.projectId === 'string' &&
    (typeof row.fileId === 'string' || typeof row.sourceRef === 'string') &&
    typeof row.name === 'string' &&
    typeof row.dateAdded === 'string'
  )
}

function withFileId(recording: ProjectRecording): ProjectRecording {
  if (recording.fileId) return recording
  const fileId = fileIdFromSourceRef(recording.sourceRef)
  return fileId ? { ...recording, fileId } : recording
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

/** Point recordings and jobs that used a browser project id at the server id. */
export function replaceProjectId(from: string, to: string) {
  update((next) => {
    const jobProject: Record<string, string> = {}
    for (const [jobId, owner] of Object.entries(next.jobProject)) {
      jobProject[jobId] = owner === from ? to : owner
    }
    next.jobProject = jobProject
    next.recordings = next.recordings.map((recording) =>
      recording.projectId === from ? { ...recording, projectId: to } : recording,
    )
    if (next.currentProjectId === from) next.currentProjectId = to
    next.projects = next.projects.filter((project) => project.id !== from)
  })
}

export function removeProject(projectId: string) {
  update((next) => {
    next.projects = next.projects.filter((project) => project.id !== projectId)
    for (const [jobId, owner] of Object.entries(next.jobProject)) {
      if (owner === projectId) delete next.jobProject[jobId]
    }
    next.recordings = next.recordings.filter((recording) => recording.projectId !== projectId)
    if (next.currentProjectId === projectId) next.currentProjectId = ''
  })
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

export function insertRecording(projectId: string, input: { fileId: string; name: string }): ProjectRecording {
  const recording: ProjectRecording = {
    id: `rec-${crypto.randomUUID()}`,
    projectId,
    fileId: input.fileId,
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
