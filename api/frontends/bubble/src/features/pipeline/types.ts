export type PipelineJob = {
  id: string
  ownerId?: string
  workflowId?: string
  runId?: string | null
  fileId?: string
  sourceRef: string
  status: string
  errorMessage?: string | null
  /** Produced artifacts. Absent until that stage of the job has finished. */
  audioRef?: string | null
  transcriptRef?: string | null
  keyFramesRef?: string | null
  documentRef?: string | null
  dateCreated?: string
  dateUpdated?: string
}

export type JobsQuery = {
  items?: PipelineJob[]
  page?: number
  rowsPerPage?: number
  total?: number
} | PipelineJob[]
