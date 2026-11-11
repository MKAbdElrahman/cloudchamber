import { progressFromJob, type JobProgress, type JobProgressInput } from './jobProgress'

/** The conversion steps for one job. The server names the live activity. */
export function useJobProgress(job: JobProgressInput | undefined): JobProgress | null {
  if (!job) return null
  return progressFromJob(job)
}
