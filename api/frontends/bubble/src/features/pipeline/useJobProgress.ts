import { useEffect, useState } from 'react'

import { jobPhase } from './jobStatus'
import { progressFromJob, type JobProgress, type JobProgressInput } from './jobProgress'

const TICK_MS = 1_000

/** Recompute the conversion while the job is still running. */
export function useJobProgress(job: JobProgressInput | undefined): JobProgress | null {
  const [now, setNow] = useState(() => Date.now())
  const live = job != null && jobPhase(job.status) === 'running' && !job.documentRef?.trim()

  useEffect(() => {
    if (!live) return undefined
    const id = window.setInterval(() => setNow(Date.now()), TICK_MS)
    return () => window.clearInterval(id)
  }, [live])

  if (!job) return null
  return progressFromJob(job, now)
}
