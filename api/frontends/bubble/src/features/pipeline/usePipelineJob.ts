import { useQuery } from '@tanstack/react-query'

import { loadJob } from './loadJob'
import { readCachedJob } from './pipelineJobCache'
import { isTerminalJobStatus } from './jobStatus'

const POLL_MS = 3000

/** Poll current pipeline job; session cache avoids loading chrome on refresh. */
export function usePipelineJob(token: string, jobId: string | null) {
  return useQuery({
    queryKey: ['pipeline-job', jobId],
    queryFn: () => loadJob(token, jobId!),
    enabled: Boolean(jobId),
    initialData: () => (jobId ? readCachedJob(jobId) : undefined),
    staleTime: 30_000,
    refetchOnMount: 'always',
    refetchInterval: (query) => (isTerminalJobStatus(query.state.data?.status) ? false : POLL_MS),
    placeholderData: (previous) => previous,
  })
}
