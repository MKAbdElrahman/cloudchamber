import { stageIdOf, type JobStageId } from './jobProgress'
import type { PipelineFact, PipelineProgress } from './types'

/** One visible beat under a step. Repeated running facts with the same sentence update this row. */
export type FactLine = {
  key: string
  label: string
  time?: string
}

/** The activity under one step, in the order the workers wrote it. A finished step is the step mark, so it is not a row. */
export function factLines(
  stage: JobStageId,
  facts: PipelineFact[] | undefined,
  updates: PipelineProgress[] | undefined,
): FactLine[] {
  const lines: FactLine[] = []
  const ordered: { position: number; key: string; label: string; time?: string; kind: 'update' | 'event' }[] = []
  for (const update of updates ?? []) {
    if (stageIdOf(update.stage) !== stage) continue
    const label = update.headline?.trim()
    if (!label) continue
    ordered.push({
      position: update.position,
      key: String(update.globalPosition || update.position),
      label,
      time: update.time,
      kind: 'update',
    })
  }
  for (const fact of facts ?? []) {
    if (stageIdOf(fact.stage ?? undefined) !== stage) continue
    const outcome = fact.outcome ?? ''
    if (outcome === 'done' || outcome === 'skipped' || outcome === '') continue
    ordered.push({
      position: fact.position,
      key: String(fact.globalPosition || fact.position),
      label: fact.message?.trim() || (outcome === 'failed' ? 'Failed.' : 'Canceled.'),
      time: fact.time,
      kind: 'event',
    })
  }
  ordered.sort((a, b) => a.position - b.position)
  for (const row of ordered) {
    const last = lines.at(-1)
    if (last && row.kind === 'update' && last.label === row.label) {
      last.time = row.time
      continue
    }
    lines.push({ key: row.key, label: row.label, time: row.time })
  }
  return lines
}
