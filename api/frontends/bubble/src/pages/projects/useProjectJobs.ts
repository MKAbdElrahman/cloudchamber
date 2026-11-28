import { useQuery } from '@tanstack/react-query'

import { jobNeedsOpen, listJobs } from '@/features/conversion'

/** How often a project with a moving job is read again. */
const LIVE_PROJECT_INTERVAL_MS = 2000

/** Jobs that belong to one project. While one is moving, the list is read again every two seconds. */
export function useProjectJobs(token: string, projectId: string) {
  return useQuery({
    queryKey: ['conversions', projectId],
    queryFn: () => listJobs(token, projectId),
    enabled: Boolean(token) && Boolean(projectId),
    staleTime: 0,
    refetchOnMount: 'always',
    refetchInterval: (query) => (query.state.data?.some(jobNeedsOpen) ? LIVE_PROJECT_INTERVAL_MS : false),
  })
}
