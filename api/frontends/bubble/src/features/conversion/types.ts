import type { Run, RunStep } from '@/shared/runs'

/** The input of the media2doc workflow. */
export type ConversionInput = { fileId?: string }

/** The result of the media2doc workflow: the file each step produced. */
export type ConversionResult = {
  sourceFileId?: string
  audioFileId?: string
  transcriptFileId?: string
  keyFramesFileId?: string
  documentFileId?: string
}

/** A run of the media2doc workflow. */
export type ConversionRun = Run<ConversionInput, ConversionResult>

/**
 * One conversion, as the pages read it. It is a run of the media2doc
 * workflow with the file names filled in. `run` is the run itself, for the parts of
 * a page that work on a run of any workflow.
 */
export type Conversion = {
  id: string
  run: ConversionRun
  ownerId?: string
  projectId?: string
  fileId?: string
  fileName?: string
  status: string
  failureMessage?: string | null
  /** The steps the workers reported, in the order they started. */
  steps?: RunStep[]
  /** Produced files. Absent until the run has finished. */
  audioFileId?: string | null
  transcriptFileId?: string | null
  keyFramesFileId?: string | null
  documentFileId?: string | null
  documentName?: string | null
  dateCreated?: string
  dateUpdated?: string
}
