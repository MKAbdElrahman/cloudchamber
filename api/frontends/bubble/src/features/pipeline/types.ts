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
  facts?: PipelineFact[]
  /** Live reports while an activity runs. */
  progress?: PipelineProgress[]
  /** Produced files. Absent until that stage of the job has finished. */
  audioFileId?: string | null
  transcriptFileId?: string | null
  keyFramesFileId?: string | null
  documentFileId?: string | null
  documentName?: string | null
  dateCreated?: string
  dateUpdated?: string
}

export type PipelineFact = {
  jobId: string
  position: number
  globalPosition: number
  fact: string
  stage?: string
  outcome?: string
  message?: string | null
  fileId?: string | null
  time?: string
}

export type PipelineProgress = {
  jobId: string
  position: number
  globalPosition: number
  stage: string
  fraction?: number | null
  headline?: string | null
  time?: string
}

export type JobsQuery = {
  items?: PipelineJob[]
  page?: number
  rowsPerPage?: number
  total?: number
  factsCursor?: number
} | PipelineJob[]

/** The folded row for one job. Follow sends this with the facts so the screen does not fold them. */
export type JobProjection = {
  id: string
  status: string
  failureMessage?: string | null
  runId?: string
  audioFileId?: string
  transcriptFileId?: string
  keyFramesFileId?: string
  documentFileId?: string
  dateUpdated?: string
}

export type FactPage = {
  facts: PipelineFact[]
  progress?: PipelineProgress[]
  cursor: number
  projections?: JobProjection[]
}
