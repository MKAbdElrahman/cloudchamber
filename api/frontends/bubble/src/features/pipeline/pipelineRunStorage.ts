const RUN_KEY = 'bubble.pipeline.run.v1'
const LEGACY_RUN_KEY = 'cloudchamber.pipeline.run.v1'

export type PipelineRunSnapshot = {
  sourceRef: string | null
  currentJobId: string | null
}

export function readPipelineRun(): PipelineRunSnapshot {
  try {
    const current = localStorage.getItem(RUN_KEY)
    const raw = current ?? localStorage.getItem(LEGACY_RUN_KEY)
    if (!current && raw) {
      localStorage.setItem(RUN_KEY, raw)
      localStorage.removeItem(LEGACY_RUN_KEY)
    }
    if (!raw) return { sourceRef: null, currentJobId: null }
    const parsed = JSON.parse(raw) as Partial<PipelineRunSnapshot>
    return {
      sourceRef: typeof parsed.sourceRef === 'string' ? parsed.sourceRef : null,
      currentJobId: typeof parsed.currentJobId === 'string' ? parsed.currentJobId : null,
    }
  } catch {
    return { sourceRef: null, currentJobId: null }
  }
}

export function writePipelineRun(snapshot: PipelineRunSnapshot) {
  try {
    localStorage.setItem(RUN_KEY, JSON.stringify(snapshot))
  } catch {
    /* ignore */
  }
}
