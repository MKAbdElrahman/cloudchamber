import {
  assignJob,
  claimUnassigned,
  insertProject,
  removeProject,
  renameProject,
  readProjectIdForJob,
  readProjectJobIds,
  readProjects,
} from './store'
import type { Project } from './types'

/**
 * Stand-in for the project API. The shapes match the endpoints we will add:
 * GET/POST /v1/projects, PATCH/DELETE /v1/projects/:id, GET/POST /v1/projects/:id/jobs.
 */

export function listProjects(): Promise<Project[]> {
  return Promise.resolve(readProjects())
}

export function createProject(name: string): Promise<Project> {
  const trimmed = name.trim()
  if (!trimmed) return Promise.reject(new Error('Project name is required.'))
  return Promise.resolve(insertProject(trimmed))
}

export function updateProject(projectId: string, name: string): Promise<void> {
  const trimmed = name.trim()
  if (!trimmed) return Promise.reject(new Error('Project name is required.'))
  renameProject(projectId, trimmed)
  return Promise.resolve()
}

export function deleteProject(projectId: string): Promise<void> {
  removeProject(projectId)
  return Promise.resolve()
}

export function listProjectJobIds(projectId: string): Promise<string[]> {
  return Promise.resolve(readProjectJobIds(projectId))
}

export function addProjectJob(projectId: string, jobId: string): Promise<void> {
  assignJob(projectId, jobId)
  return Promise.resolve()
}

export function projectIdForJob(jobId: string): Promise<string | null> {
  return Promise.resolve(readProjectIdForJob(jobId))
}

export function claimUnassignedJobs(jobIds: readonly string[], fallbackProjectId: string): Promise<void> {
  claimUnassigned(jobIds, fallbackProjectId)
  return Promise.resolve()
}
