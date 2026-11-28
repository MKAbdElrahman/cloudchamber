import { commandRun, createRun, getRun, listRuns, type RunCommand, type RunPhase } from '@/shared/runs'
import { apiFetch } from '@/shared/api'
import type { Conversion, VideoInput, VideoResult, VideoRun } from './types'

/** The workflow that turns one recording into one document. */
export const VIDEO_WORKFLOW = 'video-to-document'

type NamedFile = { id: string; name: string }

/** The job status a run phase shows as. The pages keep one vocabulary. */
function statusOf(phase: RunPhase): string {
  switch (phase) {
    case 'pending':
      return 'submitted'
    case 'succeeded':
      return 'completed'
    case 'terminated':
      return 'canceled'
    case 'running':
    case 'paused':
    case 'failed':
      return phase
    case 'canceled':
      return phase
  }
}

/** A run as a conversion. `names` maps a file id to its name in the project. */
export function jobFromRun(run: VideoRun, names: Map<string, string> = new Map()): Conversion {
  const result = run.status.result ?? undefined
  const fileId = run.spec.input?.fileId ?? result?.sourceFileId
  const documentFileId = result?.documentFileId
  return {
    id: run.id,
    run,
    ownerId: run.ownerId,
    projectId: run.projectId,
    fileId,
    fileName: fileId ? names.get(fileId) : undefined,
    status: statusOf(run.status.phase),
    failureMessage: run.status.phase === 'failed' ? (run.status.message ?? null) : null,
    steps: run.status.steps ?? [],
    audioFileId: result?.audioFileId,
    transcriptFileId: result?.transcriptFileId,
    keyFramesFileId: result?.keyFramesFileId,
    documentFileId,
    documentName: documentFileId ? names.get(documentFileId) : undefined,
    dateCreated: run.dateCreated,
    dateUpdated: run.dateUpdated,
  }
}

/** File names in one project, by file id. A project that cannot be read has none. */
async function fileNames(token: string, projectId: string): Promise<Map<string, string>> {
  try {
    const data = await apiFetch<{ items?: NamedFile[] }>(`/v1/projects/${encodeURIComponent(projectId)}/files`, {
      method: 'GET',
      token,
    })
    return new Map((data.items ?? []).map((file) => [file.id, file.name]))
  } catch {
    return new Map()
  }
}

export async function createJob(token: string, projectId: string, fileId: string, idempotencyKey?: string): Promise<Conversion> {
  const run = await createRun<VideoInput, VideoResult>(token, { workflow: VIDEO_WORKFLOW, projectId, input: { fileId }, idempotencyKey })
  return jobFromRun(run)
}

export function readFileURL(token: string, fileId: string) {
  return apiFetch<{ url: string; expiresAt: string }>(`/v1/files/${encodeURIComponent(fileId)}/url`, {
    method: 'GET',
    token,
  })
}

/** The conversions in one project, newest first. */
export async function listJobs(token: string, projectId: string): Promise<Conversion[]> {
  const [runs, names] = await Promise.all([
    listRuns<VideoInput, VideoResult>(token, { projectId, workflow: VIDEO_WORKFLOW }),
    fileNames(token, projectId),
  ])
  return runs.map((run) => jobFromRun(run, names))
}

export async function getJob(token: string, jobId: string): Promise<Conversion> {
  const run = await getRun<VideoInput, VideoResult>(token, jobId)
  return jobFromRun(run, await fileNames(token, run.projectId))
}

/** Give a conversion a command. It is a run, so it takes the commands every run takes. */
export async function commandJob(token: string, jobId: string, command: RunCommand): Promise<Conversion> {
  return jobFromRun(await commandRun<VideoInput, VideoResult>(token, jobId, command))
}
