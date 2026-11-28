import { Button } from '@astryxdesign/core/Button'

import { allowedCommands, pendingRequest } from './runState'
import type { Run, RunCommand } from './types'

const LABELS: Record<RunCommand, string> = {
  suspend: 'Pause',
  resume: 'Resume',
  cancel: 'Cancel',
  retry: 'Retry',
  restart: 'Run again',
}

type RunControlsProps = {
  run: Pick<Run, 'spec' | 'status' | 'settled'>
  onCommand: (command: RunCommand) => void
  /** True while a command is being sent. */
  isPending?: boolean
  /** Commands this page does not offer, such as running a finished conversion again. */
  hide?: RunCommand[]
}

/**
 * The buttons for a run of any workflow. It offers the commands the run's state
 * allows, and shows a request that is underway until the run catches up.
 * Place it inside a row: it renders buttons and no layout of its own.
 */
export function RunControls({ run, onCommand, isPending = false, hide = [] }: RunControlsProps) {
  const request = pendingRequest(run)
  const commands = allowedCommands(run).filter((command) => !hide.includes(command))

  return (
    <>
      {request === 'canceling' ? <Button label="Canceling" variant="secondary" isDisabled /> : null}
      {request === 'restarting' ? <Button label="Starting" variant="secondary" isDisabled /> : null}
      {commands.map((command) => {
        const pausing = command === 'resume' && request === 'pausing'
        return (
          <Button
            key={command}
            label={pausing ? 'Pausing' : LABELS[command]}
            variant="secondary"
            onClick={() => onCommand(command)}
            isDisabled={isPending || pausing}
          />
        )
      })}
    </>
  )
}
