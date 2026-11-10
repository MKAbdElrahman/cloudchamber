export type PipelineJob = {
  id: string
  ownerId?: string
  workflowId?: string
  runId?: string | null
  sourceRef: string
  status: string
  errorMessage?: string | null
  dateCreated?: string
  dateUpdated?: string
}

export type JobsQuery = {
  items?: PipelineJob[]
  page?: number
  rowsPerPage?: number
  total?: number
} | PipelineJob[]
