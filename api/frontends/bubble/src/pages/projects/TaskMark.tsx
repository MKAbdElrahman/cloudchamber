import type { MouseEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { QueueListIcon } from '@heroicons/react/24/outline'

import { Icon } from '@astryxdesign/core/Icon'
import { IconButton } from '@astryxdesign/core/IconButton'
import { jobPhase } from '@/features/pipeline'
import { WORKSPACE_PATHS } from '@/shared/layout'

type TaskMarkProps = {
  projectId: string
  jobId: string
  status?: string
  /** The document card is already a link, so the task cannot be one too. */
  inCard?: boolean
}

function taskDetail(status: string | undefined): string {
  switch (jobPhase(status)) {
    case 'running':
      return 'Task running'
    case 'finished':
      return 'Task finished'
    case 'failed':
      return 'Task failed'
    case 'paused':
      return 'Task paused'
    case 'canceled':
      return 'Task canceled'
    case 'waiting':
      return 'Task waiting'
  }
}

/** The pipeline behind a document: a queue mark, spinning while the task runs. */
export function TaskMark({ projectId, jobId, status, inCard = false }: TaskMarkProps) {
  const navigate = useNavigate()
  const detail = taskDetail(status)
  const running = jobPhase(status) === 'running'

  function openTask(event: MouseEvent) {
    event.preventDefault()
    event.stopPropagation()
    navigate(WORKSPACE_PATHS.historyJob(projectId, jobId))
  }

  return (
    <IconButton
      label="Task"
      tooltip={detail}
      variant="ghost"
      size="sm"
      isLoading={running}
      icon={<Icon icon={QueueListIcon} />}
      href={inCard ? undefined : WORKSPACE_PATHS.historyJob(projectId, jobId)}
      onClick={inCard ? openTask : undefined}
    />
  )
}
