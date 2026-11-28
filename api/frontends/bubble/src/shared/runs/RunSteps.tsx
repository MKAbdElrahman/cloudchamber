import { ProgressBar } from '@astryxdesign/core/ProgressBar'
import { Step, Stepper } from '@astryxdesign/core/Stepper'

import type { StepView } from './runState'

/** One bar for the step that is running or paused. The label is its sentence and the value is the percent. */
function stepBar(step: StepView) {
  if (step.state !== 'current' && step.state !== 'paused') return null
  const known = step.progress != null
  if (step.state === 'paused' && !known) return null
  return (
    <ProgressBar
      label={step.detail || step.label}
      value={known ? Math.round((step.progress ?? 0) * 100) : 0}
      hasValueLabel={known}
      isIndeterminate={!known}
      variant={step.state === 'paused' ? 'neutral' : 'accent'}
    />
  )
}

function stepStatus(step: StepView): 'error' | 'warning' | undefined {
  switch (step.state) {
    case 'failed':
      return 'error'
    case 'canceled':
      return 'warning'
    case 'done':
    case 'current':
    case 'paused':
    case 'upcoming':
      return undefined
  }
}

/** The steps of a run of any workflow, in order, with the live one showing how far it has got. */
export function RunSteps({ steps, activeStep, label }: { steps: StepView[]; activeStep: number; label: string }) {
  return (
    <Stepper orientation="vertical" activeStep={activeStep} label={label}>
      {steps.map((step, index) => {
        const bar = stepBar(step)
        return (
          <Step
            key={step.key}
            step={index}
            label={step.label}
            description={bar ? undefined : step.detail || undefined}
            status={stepStatus(step)}
          >
            {bar}
          </Step>
        )
      })}
    </Stepper>
  )
}
