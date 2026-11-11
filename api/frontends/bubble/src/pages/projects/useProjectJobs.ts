import { useQuery, useQueryClient } from '@tanstack/react-query'

import { isTerminalJobStatus, listJobs, loadJob, normalizeJobs, type PipelineJob } from '@/features/pipeline'

const POLL_MS = 3_000

function needsReconcile(job: PipelineJob): boolean {
  return !isTerminalJobStatus(job.status) && !job.documentFileId?.trim()
}

/** Ask for each open job. The list endpoint does not reconcile a finished workflow. */
async function reconcileOpenJobs(token: string, jobs: PipelineJob[]): Promise<PipelineJob[]> {
  return Promise.all(
    jobs.map(async (job) => {
      if (!needsReconcile(job)) return job
      try {
        return await loadJob(token, job.id)
      } catch {
        return job
      }
    }),
  )
}

/** Jobs that belong to one project. Open jobs are refreshed until they settle. */
export function useProjectJobs(token: string, projectId: string) {
  const queryClient = useQueryClient()
  return useQuery({
    queryKey: ['pipeline-jobs', projectId],
    queryFn: async (): Promise<PipelineJob[]> => {
      const jobs = normalizeJobs(await listJobs(token, projectId)).filter((job) => job.projectId === projectId)
      const mine = await reconcileOpenJobs(token, jobs)
      for (const job of mine) {
        if (!needsReconcile(job)) queryClient.setQueryData(['pipeline-job', job.id], job)
      }
      return mine
    },
    enabled: Boolean(token) && Boolean(projectId),
    staleTime: 30_000,
    refetchOnMount: 'always',
    refetchInterval: (query) => (query.state.data?.some(needsReconcile) ? POLL_MS : false),
  })
}
