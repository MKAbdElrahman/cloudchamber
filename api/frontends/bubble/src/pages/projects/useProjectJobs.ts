import { useQuery, useQueryClient } from '@tanstack/react-query'

import { jobNeedsOpen, listJobs, normalizeJobs, openUnsettledJobs, type PipelineJob } from '@/features/pipeline'

const POLL_MS = 3_000

/** Jobs that belong to one project. Open jobs are refreshed until they settle. */
export function useProjectJobs(token: string, projectId: string) {
  const queryClient = useQueryClient()
  return useQuery({
    queryKey: ['pipeline-jobs', projectId],
    queryFn: async (): Promise<PipelineJob[]> => {
      const jobs = normalizeJobs(await listJobs(token, projectId)).filter((job) => job.projectId === projectId)
      const mine = await openUnsettledJobs(token, jobs)
      for (const job of mine) {
        if (!jobNeedsOpen(job)) queryClient.setQueryData(['pipeline-job', job.id], job)
      }
      return mine
    },
    enabled: Boolean(token) && Boolean(projectId),
    staleTime: 30_000,
    refetchOnMount: 'always',
    refetchInterval: (query) => (query.state.data?.some(jobNeedsOpen) ? POLL_MS : false),
  })
}
