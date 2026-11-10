export type OutputFileKind = 'pdf' | 'markdown' | 'video' | 'audio'

/** One job deliverable. Pages pass this list; the feature does not own fixtures. */
export type OutputFile = {
  id: string
  name: string
  kind: OutputFileKind
  dateUpdated: string
  jobId: string
  /** Inline text when the file is markdown. Omitted for binary files. */
  markdown?: string
}
