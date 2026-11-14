import { useEffect, useRef } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'

import { followJobFacts } from './api'
import { maxFactPosition, pageHasUpdates, withFactPage } from './facts'
import { delay } from './wait'
import { loadJob } from './loadJob'
import { isTerminalJobStatus, visibleJobStatus } from './jobStatus'
import { readCachedJob } from './pipelineJobCache'
import type { PipelineJob } from './types'

/** One job. The first read loads its stream, then the page follows new facts. */
export function usePipelineJob(token: string, jobId: string | null) {
  const queryClient = useQueryClient()
  const query = useQuery({
    queryKey: ['pipeline-job', jobId],
    queryFn: () => loadJob(token, jobId!),
    enabled: Boolean(jobId),
    initialData: () => (jobId ? readCachedJob(jobId) : undefined),
    staleTime: 30_000,
    refetchOnMount: 'always',
    placeholderData: (previous) => previous,
  })
  const cursor = useRef(0)

  useEffect(() => {
    if (!token || !jobId || !query.isFetched) return
    const loaded = queryClient.getQueryData<PipelineJob>(['pipeline-job', jobId])
    if (!loaded || isTerminalJobStatus(visibleJobStatus(loaded))) return
    cursor.current = Math.max(maxFactPosition(loaded.facts), maxFactPosition(loaded.progress))
    let stopped = false

    const loop = async () => {
      while (!stopped) {
        const cached = queryClient.getQueryData<PipelineJob>(['pipeline-job', jobId])
        if (cached && isTerminalJobStatus(visibleJobStatus(cached))) return
        try {
          const page = await followJobFacts(token, jobId, cursor.current)
          if (stopped) return
          cursor.current = Math.max(cursor.current, page.cursor)
          if (!pageHasUpdates(page)) continue
          queryClient.setQueryData<PipelineJob>(['pipeline-job', jobId], (job) =>
            job ? withFactPage(job, page) : job,
          )
        } catch {
          await delay(1000)
        }
      }
    }
    void loop()
    return () => {
      stopped = true
    }
  }, [jobId, query.isFetched, queryClient, token])

  return query
}
