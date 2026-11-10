import { fileNameFromSourceRef } from '@/shared/media'

const MEDIA_FILE = /\.(mp4|webm|mov|m4v|mkv|avi|ogv|mp3|wav|ogg|m4a|aac|flac)$/i

/** The refs a job may have. The project lists the document only. */
export type JobArtifacts = {
  id: string
  sourceRef?: string
  dateUpdated?: string
  documentRef?: string | null
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
  ref: string
  dateUpdated?: string
}

export function documentFormat(ref: string): ProjectFileKind {
  return fileNameFromSourceRef(ref).toLowerCase().endsWith('.pdf') ? 'pdf' : 'markdown'
}

function stemOfMediaName(name: string): string {
  if (!MEDIA_FILE.test(name)) return ''
  return name.trim().replace(/\.[^.]+$/, '')
}

/**
 * File name for the project. A recording named week-03.mp4 becomes week-03.md.
 * An opaque upload id keeps the export's own name, document.md.
 */
export function documentFileName(sourceRef: string | undefined, documentRef: string, mediaName?: string): string {
  const extension = documentFormat(documentRef) === 'pdf' ? 'pdf' : 'md'
  const fromMedia = stemOfMediaName(mediaName ?? '')
  if (fromMedia.length > 0) return `${fromMedia}.${extension}`
  const sourceName = sourceRef?.trim() ? fileNameFromSourceRef(sourceRef) : ''
  const fromRef = stemOfMediaName(sourceName)
  if (fromRef.length > 0) return `${fromRef}.${extension}`
  return fileNameFromSourceRef(documentRef)
}

/** The document a job has written. A job with no export contributes nothing. */
export function filesFromJob(job: JobArtifacts, mediaName?: string): ProjectFile[] {
  const ref = job.documentRef?.trim()
  if (!ref) return []
  const named = mediaName?.trim() && MEDIA_FILE.test(mediaName.trim()) ? mediaName.trim() : undefined
  const sourceName = named ?? (job.sourceRef?.trim() ? fileNameFromSourceRef(job.sourceRef) : undefined)
  return [
    {
      id: `${job.id}:document`,
      jobId: job.id,
      kind: documentFormat(ref),
      name: documentFileName(job.sourceRef, ref, named),
      sourceName,
      ref,
      dateUpdated: job.dateUpdated,
    },
  ]
}

/** Newest document first. */
export function filesFromJobs(jobs: JobArtifacts[]): ProjectFile[] {
  return jobs.flatMap((job) => filesFromJob(job)).sort((a, b) => (b.dateUpdated ?? '').localeCompare(a.dateUpdated ?? ''))
}
