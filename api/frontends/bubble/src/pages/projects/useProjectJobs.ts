import { useQuery, useQueryClient } from '@tanstack/react-query'

import { getJob, isTerminalJobStatus, listJobs, normalizeJobs, writeCachedJob, type PipelineJob } from '@/features/pipeline'
import { claimUnassignedJobs, DEFAULT_PROJECT_ID, listProjectJobIds } from '@/features/projects'

const POLL_MS = 3_000

function needsReconcile(job: PipelineJob): boolean {
  return !isTerminalJobStatus(job.status) && !job.documentRef?.trim()
}

/** Ask for each open job. The list endpoint does not reconcile a finished workflow. */
async function reconcileOpenJobs(token: string, jobs: PipelineJob[]): Promise<PipelineJob[]> {
  return Promise.all(
    jobs.map(async (job) => {
      if (!needsReconcile(job)) return job
      try {
        const current = await getJob(token, job.id)
        writeCachedJob(current)
        return current
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
      const jobs = normalizeJobs(await listJobs(token))
      await claimUnassignedJobs(
        jobs.map((job) => job.id),
        DEFAULT_PROJECT_ID,
      )
      const ids = new Set(await listProjectJobIds(projectId))
      const mine = await reconcileOpenJobs(token, jobs.filter((job) => ids.has(job.id)))
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
