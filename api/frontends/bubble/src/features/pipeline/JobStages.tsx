import { Banner } from '@astryxdesign/core/Banner'
import { VStack } from '@astryxdesign/core/Layout'
import { List, ListItem } from '@astryxdesign/core/List'
import { ProgressBar } from '@astryxdesign/core/ProgressBar'
import { Step, Stepper } from '@astryxdesign/core/Stepper'
import { Text } from '@astryxdesign/core/Text'
import { Timestamp } from '@astryxdesign/core/Timestamp'
import { VisuallyHidden } from '@astryxdesign/core/VisuallyHidden'
import { WORKSPACE_GAP } from '@/shared/spacing'

import { factLines, type FactLine } from './factLog'
import { conversionNotice, type JobProgress, type JobStageView } from './jobProgress'
import type { JobPhase } from './jobStatus'
import type { PipelineFact, PipelineProgress } from './types'

/** One bar for the step that is running. The label is the latest sentence; the value is the percent. */
function stepBar(stage: JobStageView) {
  if (stage.state !== 'current') return null
  const known = stage.progress != null
  return (
    <ProgressBar
      label={stage.detail || stage.label}
      value={known ? Math.round((stage.progress ?? 0) * 100) : 0}
      hasValueLabel={known}
      isIndeterminate={!known}
      variant="accent"
    />
  )
}

function stepStatus(stage: JobStageView, phase: JobPhase): 'error' | 'warning' | undefined {
  if (stage.state !== 'failed' && stage.state !== 'canceled') return undefined
  switch (phase) {
    case 'failed':
      return 'error'
    case 'canceled':
      return 'warning'
    case 'waiting':
    case 'running':
    case 'finished':
      return undefined
  }
}

/**
 * Sentences already written for this step. The live sentence is the progress bar's label,
 * so it is not repeated on the current step.
 */
function earlierLines(stage: JobStageView, facts: PipelineFact[] | undefined, updates: PipelineProgress[] | undefined): FactLine[] {
  const lines = factLines(stage.id, facts, updates)
  if (stage.state !== 'current') return lines
  const live = stage.detail
  const last = lines.at(-1)
  if (live && last?.label === live) return lines.slice(0, -1)
  return lines
}

function factLog(stage: JobStageView, lines: FactLine[]) {
  if (lines.length === 0) return null
  return (
    <List density="compact" edge="flush" header={<VisuallyHidden>{stage.label} activity</VisuallyHidden>}>
      {lines.map((line) => (
        <ListItem
          key={line.key}
          label={<Text maxLines={2}>{line.label}</Text>}
          description={line.time ? <Timestamp value={line.time} format="relative" /> : undefined}
        />
      ))}
    </List>
  )
}

/** The five activities. A stopped job leads with the notice; the stepper is the sequence. */
export function JobStages({ progress, facts, updates }: { progress: JobProgress; facts?: PipelineFact[]; updates?: PipelineProgress[] }) {
  const notice = conversionNotice(progress.phase, progress.stages, progress.reason)
  return (
    <VStack gap={WORKSPACE_GAP.group}>
      {notice ? <Banner status={notice.status} title={notice.title} description={notice.description} /> : null}
      <Stepper orientation="vertical" activeStep={progress.activeStep} label="Conversion">
        {progress.stages.map((stage, index) => {
          const earlier = earlierLines(stage, facts, updates)
          const bar = stepBar(stage)
          const log = factLog(stage, earlier)
          return (
            <Step
              key={stage.id}
              step={index}
              label={stage.label}
              description={stage.state === 'current' ? undefined : stage.detail || undefined}
              status={stepStatus(stage, progress.phase)}
            >
              {bar || log ? (
                <VStack gap={WORKSPACE_GAP.local}>
                  {log}
                  {bar}
                </VStack>
              ) : null}
            </Step>
          )
        })}
      </Stepper>
    </VStack>
  )
}
