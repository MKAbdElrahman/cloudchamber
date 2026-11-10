import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'

import { readPipelineRun, writePipelineRun } from './pipelineRunStorage'

type PipelineRunContextValue = {
  sourceRef: string | null
  setSourceRef: (ref: string | null) => void
  currentJobId: string | null
  setCurrentJobId: (id: string | null) => void
}

const PipelineRunContext = createContext<PipelineRunContextValue | null>(null)

export function PipelineRunProvider({ children }: { children: ReactNode }) {
  const initial = readPipelineRun()
  const [sourceRef, setSourceRefState] = useState<string | null>(initial.sourceRef)
  const [currentJobId, setCurrentJobIdState] = useState<string | null>(initial.currentJobId)

  useEffect(() => {
    writePipelineRun({ sourceRef, currentJobId })
  }, [sourceRef, currentJobId])

  const setSourceRef = useCallback((ref: string | null) => setSourceRefState(ref), [])
  const setCurrentJobId = useCallback((id: string | null) => setCurrentJobIdState(id), [])

  const value = useMemo(
    () => ({
      sourceRef,
      setSourceRef,
      currentJobId,
      setCurrentJobId,
    }),
    [sourceRef, currentJobId, setSourceRef, setCurrentJobId],
  )

  return <PipelineRunContext.Provider value={value}>{children}</PipelineRunContext.Provider>
}

export function usePipelineRun() {
  const ctx = useContext(PipelineRunContext)
  if (!ctx) throw new Error('usePipelineRun must be used within PipelineRunProvider')
  return ctx
}
