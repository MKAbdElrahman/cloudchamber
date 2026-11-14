import { useEffect, useRef } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'

import { factsCursor, followProjectFacts, listJobs, normalizeJobs, type PipelineJob } from '@/features/pipeline'
import { applyProjectFacts, pageHasUpdates } from '@/features/pipeline/facts'
import { delay } from '@/features/pipeline/wait'

/** Jobs that belong to one project. New facts arrive on the project stream. */
export function useProjectJobs(token: string, projectId: string) {
  const queryClient = useQueryClient()
  const cursor = useRef(0)
  const query = useQuery({
    queryKey: ['pipeline-jobs', projectId],
    queryFn: async (): Promise<PipelineJob[]> => {
      const listed = await listJobs(token, projectId)
      const next = factsCursor(listed)
      cursor.current = Math.max(cursor.current, next)
      return normalizeJobs(listed).filter((job) => job.projectId === projectId)
    },
    enabled: Boolean(token) && Boolean(projectId),
    staleTime: 30_000,
    refetchOnMount: 'always',
  })

  useEffect(() => {
    if (!token || !projectId || !query.isFetched) return
    let stopped = false

    const loop = async () => {
      while (!stopped) {
        try {
          const page = await followProjectFacts(token, projectId, cursor.current)
          if (stopped) return
          cursor.current = Math.max(cursor.current, page.cursor)
          if (!pageHasUpdates(page)) continue
          const unknown = applyProjectFacts(queryClient, projectId, page)
          if (unknown) void queryClient.invalidateQueries({ queryKey: ['pipeline-jobs', projectId] })
        } catch {
          await delay(1000)
        }
      }
    }
    void loop()
    return () => {
      stopped = true
    }
  }, [query.isFetched, queryClient, projectId, token])

  return query
}
