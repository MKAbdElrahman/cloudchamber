import type { Run, RunStep } from '@/shared/runs'

/** The input of the video-to-document workflow. */
export type VideoInput = { fileId?: string }

/** The result of the video-to-document workflow: the file each step produced. */
export type VideoResult = {
  sourceFileId?: string
  audioFileId?: string
  transcriptFileId?: string
  keyFramesFileId?: string
  documentFileId?: string
}

/** A run of the video-to-document workflow. */
export type VideoRun = Run<VideoInput, VideoResult>

/**
 * One conversion, as the pages read it. It is a run of the video-to-document
 * workflow with the file names filled in. `run` is the run itself, for the parts of
 * a page that work on a run of any workflow.
 */
export type Conversion = {
  id: string
  run: VideoRun
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
