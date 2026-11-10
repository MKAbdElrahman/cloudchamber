import { useQuery } from '@tanstack/react-query'

import { claimUnassignedJobs, listProjects, projectIdForJob } from './api'
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
