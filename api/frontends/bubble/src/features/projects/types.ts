/** One isolated workspace. Media, jobs, and documents stay inside it. */
export type Project = {
  id: string
  name: string
  dateCreated: string
}

/** A recording kept in a project. Stand-in until a media API exists. */
export type ProjectRecording = {
  id: string
  projectId: string
  /** Durable id the API uses. The storage location stays on the server. */
  fileId?: string
  /** Older rows only. New recordings do not store this. */
  sourceRef?: string
  name: string
  dateAdded: string
  jobId?: string
}
