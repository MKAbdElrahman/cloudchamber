import { useQuery } from '@tanstack/react-query'

import { LIVE_RUN_INTERVAL_MS } from '@/shared/runs'

import { loadJob, jobNeedsOpen } from './loadJob'
import { readCachedJob } from './conversionCache'

/** One job. While its run is moving, the page reads it again every second. */
export function useConversion(token: string, jobId: string | null) {
  return useQuery({
    queryKey: ['conversion', jobId],
    queryFn: () => loadJob(token, jobId!),
    enabled: Boolean(jobId),
    initialData: () => (jobId ? readCachedJob(jobId) : undefined),
    staleTime: 0,
    refetchOnMount: 'always',
    placeholderData: (previous) => previous,
    refetchInterval: (query) => (query.state.data && !jobNeedsOpen(query.state.data) ? false : LIVE_RUN_INTERVAL_MS),
  })
}
