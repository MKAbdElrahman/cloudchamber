import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'

import { readPipelineRun, writePipelineRun } from './pipelineRunStorage'
import { PipelineRunContext } from './usePipelineRun'

export function PipelineRunProvider({
  projectId,
  adoptLegacy = false,
  children,
}: {
  projectId: string
  /** Move a pre-project run into this project once. Other projects do not see it. */
  adoptLegacy?: boolean
  children: ReactNode
}) {
  const initial = readPipelineRun(projectId, adoptLegacy)
  const [sourceRef, setSourceRefState] = useState<string | null>(initial.sourceRef)
  const [currentJobId, setCurrentJobIdState] = useState<string | null>(initial.currentJobId)

  useEffect(() => {
    writePipelineRun(projectId, { sourceRef, currentJobId })
  }, [projectId, sourceRef, currentJobId])

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
