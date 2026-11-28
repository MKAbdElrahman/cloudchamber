import { useQuery } from '@tanstack/react-query'

import { getRun, listWorkflows, listRuns, type RunsFilter } from './api'
import { isRunMoving } from './runState'
import type { Workflow } from './types'

/** How often a run that is moving is read again. */
export const LIVE_RUN_INTERVAL_MS = 1000
/** How often a list with a moving run in it is read again. */
export const LIVE_RUNS_INTERVAL_MS = 2000

/** One run of any workflow. While it is moving, it is read again every second. */
export function useRun<Input = unknown, Result = unknown>(token: string, runId: string | null) {
  return useQuery({
    queryKey: ['run', runId],
    queryFn: () => getRun<Input, Result>(token, runId!),
    enabled: Boolean(token) && Boolean(runId),
    staleTime: 0,
    refetchOnMount: 'always',
    placeholderData: (previous) => previous,
    refetchInterval: (query) => (query.state.data && !isRunMoving(query.state.data) ? false : LIVE_RUN_INTERVAL_MS),
  })
}

/** Runs the caller owns. While one of them is moving, the list is read again every two seconds. */
export function useRuns<Input = unknown, Result = unknown>(token: string, filter: RunsFilter = {}) {
  return useQuery({
    queryKey: ['runs', filter.projectId ?? null, filter.workflow ?? null],
    queryFn: () => listRuns<Input, Result>(token, filter),
    enabled: Boolean(token),
    staleTime: 0,
    refetchOnMount: 'always',
    refetchInterval: (query) => (query.state.data?.some(isRunMoving) ? LIVE_RUNS_INTERVAL_MS : false),
  })
}

/** The workflows the service can start. They change only when the service is deployed. */
export function useWorkflows(token: string) {
  return useQuery({
    queryKey: ['workflows'],
    queryFn: () => listWorkflows(token),
    enabled: Boolean(token),
    staleTime: 10 * 60_000,
  })
}

/** One workflow by name, once the workflows have loaded. */
export function useWorkflow(token: string, name: string): Workflow | undefined {
  return useWorkflows(token).data?.find((workflow) => workflow.name === name)
}
