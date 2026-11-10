const RUN_KEY = 'bubble.pipeline.run.v1'
const LEGACY_RUN_KEY = 'cloudchamber.pipeline.run.v1'

export type PipelineRunSnapshot = {
  sourceRef: string | null
  currentJobId: string | null
}

function runKey(projectId: string): string {
  return `${RUN_KEY}.${projectId}`
}

function parseSnapshot(raw: string | null): PipelineRunSnapshot | null {
  if (!raw) return null
  const parsed = JSON.parse(raw) as Partial<PipelineRunSnapshot>
  return {
    sourceRef: typeof parsed.sourceRef === 'string' ? parsed.sourceRef : null,
    currentJobId: typeof parsed.currentJobId === 'string' ? parsed.currentJobId : null,
  }
}

export function readPipelineRun(projectId: string, adoptLegacy: boolean): PipelineRunSnapshot {
  const empty = { sourceRef: null, currentJobId: null }
  try {
    const own = parseSnapshot(localStorage.getItem(runKey(projectId)))
    if (own) return own
    if (!adoptLegacy) return empty
    const legacy = localStorage.getItem(RUN_KEY) ?? localStorage.getItem(LEGACY_RUN_KEY)
    const snapshot = parseSnapshot(legacy) ?? empty
    localStorage.setItem(runKey(projectId), JSON.stringify(snapshot))
    localStorage.removeItem(RUN_KEY)
    localStorage.removeItem(LEGACY_RUN_KEY)
    return snapshot
  } catch {
    return empty
  }
}

export function writePipelineRun(projectId: string, snapshot: PipelineRunSnapshot) {
  try {
    localStorage.setItem(runKey(projectId), JSON.stringify(snapshot))
  } catch {
    /* ignore */
  }
}
