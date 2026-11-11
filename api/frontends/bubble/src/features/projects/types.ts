/** One isolated workspace. Media, jobs, and documents stay inside it. */
export type Project = {
  id: string
  ownerID: string
  name: string
  dateCreated: string
  dateUpdated: string
}

/** One file stored in a project. */
export type ProjectFile = {
  id: string
  projectId: string
  name: string
  kind: string
  contentType: string
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
