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

