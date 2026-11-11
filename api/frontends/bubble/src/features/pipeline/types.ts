export type PipelineJob = {
  id: string
  ownerId?: string
  workflowId?: string
  runId?: string | null
  projectId?: string
  fileId?: string
  fileName?: string
  status: string
  failureMessage?: string | null
  /** Activity Type the open run has reached. Empty once the run is closed. */
  progressActivity?: string | null
  /** Activity Type of the step where the run closed. */
  stoppedActivityType?: string | null
  /** Produced files. Absent until that stage of the job has finished. */
  audioFileId?: string | null
  transcriptFileId?: string | null
  keyFramesFileId?: string | null
  documentFileId?: string | null
  documentName?: string | null
  dateCreated?: string
  dateUpdated?: string
}

export type JobsQuery = {
  items?: PipelineJob[]
  page?: number
  rowsPerPage?: number
  total?: number
} | PipelineJob[]
