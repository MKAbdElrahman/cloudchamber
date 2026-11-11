import { useQuery } from '@tanstack/react-query'

import { useSession } from '@/shared/auth'

import { getRecording, listProjectRecordings, listProjects, projectIdForJob } from './api'

export function useProjects() {
  const { token } = useSession()
  return useQuery({
    queryKey: ['projects', token],
    queryFn: () => listProjects(token ?? ''),
    enabled: Boolean(token),
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

/** The project that owns a job, when one was chosen in this browser. */
export function useJobOwner(jobId: string) {
  return useQuery({
    queryKey: ['job-project', jobId],
    queryFn: () => projectIdForJob(jobId),
    enabled: Boolean(jobId),
  })
}
