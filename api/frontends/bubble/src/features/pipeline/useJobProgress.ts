import { progressFromJob, type JobProgress, type JobProgressInput } from './jobProgress'

/** The conversion for one job. The page folds the facts the workers appended. */
export function useJobProgress(job: JobProgressInput | undefined): JobProgress | null {
  if (!job) return null
  return progressFromJob(job)
}
