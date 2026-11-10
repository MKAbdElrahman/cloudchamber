import {
  CheckCircleIcon,
  ClockIcon,
  NoSymbolIcon,
  PauseCircleIcon,
  XCircleIcon,
} from '@heroicons/react/24/outline'

import { Icon } from '@astryxdesign/core/Icon'
import { VStack } from '@astryxdesign/core/Layout'
import { List, ListItem } from '@astryxdesign/core/List'
import { ProgressBar, type ProgressBarVariant } from '@astryxdesign/core/ProgressBar'
import { Spinner } from '@astryxdesign/core/Spinner'
import { WORKSPACE_GAP } from '@/shared/spacing'

import type { JobPhase } from './jobStatus'
import type { JobProgress, JobStageView } from './jobProgress'

function barVariant(phase: JobPhase): ProgressBarVariant {
  switch (phase) {
    case 'finished':
      return 'success'
    case 'failed':
      return 'error'
    case 'paused':
    case 'canceled':
      return 'warning'
    case 'waiting':
      return 'neutral'
    case 'running':
      return 'accent'
  }
}

function stageMark(stage: JobStageView, phase: JobPhase) {
  if (stage.state === 'current' && phase === 'waiting') {
    return <Icon icon={ClockIcon} size="sm" color="accent" />
  }
  switch (stage.state) {
    case 'done':
      return <Icon icon={CheckCircleIcon} size="sm" color="success" />
    case 'current':
      return <Spinner size="sm" shade="inherit" aria-label="Working" />
    case 'paused':
      return <Icon icon={PauseCircleIcon} size="sm" color="warning" />
    case 'failed':
      return <Icon icon={XCircleIcon} size="sm" color="error" />
    case 'canceled':
      return <Icon icon={NoSymbolIcon} size="sm" color="warning" />
    case 'upcoming':
      return <Icon icon={ClockIcon} size="sm" color="disabled" />
  }
}

/** The same six stages in every outcome. The marks and the bar carry the phase. */
export function JobStages({ progress }: { progress: JobProgress }) {
  return (
    <VStack gap={WORKSPACE_GAP.group}>
      <ProgressBar
        label="Stages"
        value={progress.completedCount}
        max={progress.stages.length}
        hasValueLabel
        formatValueLabel={(value, max) => `${value} of ${max}`}
        variant={barVariant(progress.phase)}
        isDisabled={progress.phase === 'canceled'}
      />
      <List hasDividers aria-label="Stages">
        {progress.stages.map((stage) => (
          <ListItem
            key={stage.id}
            label={stage.label}
            description={stage.detail}
            startContent={stageMark(stage, progress.phase)}
          />
        ))}
      </List>
    </VStack>
  )
}
