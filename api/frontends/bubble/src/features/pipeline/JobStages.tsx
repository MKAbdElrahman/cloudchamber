import { Banner } from '@astryxdesign/core/Banner'
import { VStack } from '@astryxdesign/core/Layout'
import { Step, Stepper } from '@astryxdesign/core/Stepper'
import { WORKSPACE_GAP } from '@/shared/spacing'

import { conversionNotice, type JobProgress, type JobStageView } from './jobProgress'
import type { JobPhase } from './jobStatus'

function stepStatus(stage: JobStageView, phase: JobPhase): 'error' | 'warning' | undefined {
  if (stage.state !== 'failed' && stage.state !== 'canceled' && stage.state !== 'paused') return undefined
  switch (phase) {
    case 'failed':
      return 'error'
    case 'canceled':
    case 'paused':
      return 'warning'
    case 'waiting':
    case 'running':
    case 'finished':
      return undefined
  }
}

/** The five activities. A stopped job leads with the notice; the stepper is the sequence. */
export function JobStages({ progress }: { progress: JobProgress }) {
  const notice = conversionNotice(progress.phase, progress.stages, progress.reason)
  return (
    <VStack gap={WORKSPACE_GAP.group}>
      {notice ? <Banner status={notice.status} title={notice.title} description={notice.description} /> : null}
      <Stepper orientation="vertical" activeStep={progress.activeStep} label="Conversion">
        {progress.stages.map((stage, index) => (
          <Step
            key={stage.id}
            step={index}
            label={stage.label}
            description={stage.state === 'upcoming' ? undefined : stage.detail}
            status={stepStatus(stage, progress.phase)}
          />
        ))}
      </Stepper>
    </VStack>
  )
}
