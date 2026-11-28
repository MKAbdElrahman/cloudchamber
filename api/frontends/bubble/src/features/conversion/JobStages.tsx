import { Banner } from '@astryxdesign/core/Banner'
import { VStack } from '@astryxdesign/core/Layout'
import { RunSteps } from '@/shared/runs'
import { WORKSPACE_GAP } from '@/shared/spacing'

import { conversionNotice, type JobProgress } from './jobProgress'

/** A conversion's steps. A stopped job leads with the notice, and the steps are the sequence. */
export function JobStages({ progress }: { progress: JobProgress }) {
  const notice = conversionNotice(progress.phase, progress.stages, progress.reason)
  return (
    <VStack gap={WORKSPACE_GAP.group}>
      {notice ? <Banner status={notice.status} title={notice.title} description={notice.description} /> : null}
      <RunSteps steps={progress.stages} activeStep={progress.activeStep} label="Conversion" />
    </VStack>
  )
}
