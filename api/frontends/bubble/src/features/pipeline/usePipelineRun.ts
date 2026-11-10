import { createContext, useContext } from 'react'

export type PipelineRunContextValue = {
  sourceRef: string | null
  setSourceRef: (ref: string | null) => void
  currentJobId: string | null
  setCurrentJobId: (id: string | null) => void
}

export const PipelineRunContext = createContext<PipelineRunContextValue | null>(null)

export function usePipelineRun() {
  const ctx = useContext(PipelineRunContext)
  if (!ctx) throw new Error('usePipelineRun must be used within PipelineRunProvider')
  return ctx
}
