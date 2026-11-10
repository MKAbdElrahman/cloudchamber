import { useQuery } from '@tanstack/react-query'

import { claimUnassignedJobs, getRecording, listProjectRecordings, listProjects, projectIdForJob } from './api'
import { DEFAULT_PROJECT_ID } from './store'

export function useProjects() {
  return useQuery({
    queryKey: ['projects'],
    queryFn: listProjects,
    staleTime: Infinity,
  })
}

export function useProject(projectId: string) {
  const projects = useProjects()
  return projects.data?.find((project) => project.id === projectId)
}

export function useProjectRecordings(projectId: string) {
  return useQuery({
    queryKey: ['project-recordings', projectId],
    queryFn: () => listProjectRecordings(projectId),
    enabled: Boolean(projectId),
  })
}

export function useRecording(projectId: string, recordingId: string) {
  return useQuery({
    queryKey: ['project-recording', projectId, recordingId],
    queryFn: () => getRecording(projectId, recordingId),
    enabled: Boolean(projectId && recordingId),
  })
}

/** The project that owns a job. An unassigned job joins the default project. */
export function useJobOwner(jobId: string) {
  return useQuery({
    queryKey: ['job-project', jobId],
    queryFn: async () => {
      const existing = await projectIdForJob(jobId)
      if (existing) return existing
      await claimUnassignedJobs([jobId], DEFAULT_PROJECT_ID)
      return projectIdForJob(jobId)
    },
    enabled: Boolean(jobId),
  })
}
