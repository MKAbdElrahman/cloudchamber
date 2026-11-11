import { knownMediaName } from './fileName'

/** The files a job may have produced. The project lists the document only. */
export type JobArtifacts = {
  id: string
  fileName?: string
  dateUpdated?: string
  documentFileId?: string | null
  documentName?: string | null
}

/** The export a person reads. Transcript, audio, and frames stay on the job. */
export type ProjectFileKind = 'markdown' | 'pdf'

export type ProjectFile = {
  id: string
  jobId: string
  kind: ProjectFileKind
  /** Name in the project, as a file. */
  name: string
  /** Recording this document was composed from, when the job recorded one. */
  sourceName?: string
  fileId: string
  dateUpdated?: string
}

export function documentFormat(documentName: string | null | undefined): ProjectFileKind {
  return documentName?.trim().toLowerCase().endsWith('.pdf') ? 'pdf' : 'markdown'
}

/**
 * File name for the project. A recording named week-03.mp4 becomes week-03.md.
 * An opaque upload id keeps the export's own name.
 */
export function documentFileName(fileName: string | undefined, documentName: string | undefined, mediaName?: string): string {
  const extension = documentFormat(documentName) === 'pdf' ? 'pdf' : 'md'
  const media = knownMediaName(fileName, mediaName)
  if (media) return `${media.replace(/\.[^.]+$/, '')}.${extension}`
  const stored = documentName?.trim()
  if (stored) return stored
  return `document.${extension}`
}

/** The document a job has written. A job with no export contributes nothing. */
export function filesFromJob(job: JobArtifacts, mediaName?: string): ProjectFile[] {
  const fileId = job.documentFileId?.trim()
  if (!fileId) return []
  const named = knownMediaName(undefined, mediaName) ?? undefined
  const sourceName = named ?? knownMediaName(job.fileName) ?? undefined
  return [
    {
      id: `${job.id}:document`,
      jobId: job.id,
      kind: documentFormat(job.documentName),
      name: documentFileName(job.fileName, job.documentName ?? undefined, named),
      sourceName,
      fileId,
      dateUpdated: job.dateUpdated,
    },
  ]
}

/** Newest document first. */
export function filesFromJobs(jobs: JobArtifacts[]): ProjectFile[] {
  return jobs.flatMap((job) => filesFromJob(job)).sort((a, b) => (b.dateUpdated ?? '').localeCompare(a.dateUpdated ?? ''))
}
