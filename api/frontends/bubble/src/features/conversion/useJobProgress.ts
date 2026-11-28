import { useWorkflow } from '@/shared/runs'

import { MEDIA2DOC_WORKFLOW } from './api'
import { progressFromJob, type JobProgress, type JobProgressInput } from './jobProgress'

/** The conversion for one job: the workflow's steps, in the state the run recorded for each. */
export function useJobProgress(token: string, job: JobProgressInput | undefined): JobProgress | null {
  const workflow = useWorkflow(token, MEDIA2DOC_WORKFLOW)
  if (!job) return null
  return progressFromJob(job, workflow)
}
