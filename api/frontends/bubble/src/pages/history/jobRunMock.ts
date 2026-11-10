import { isJobSuccess, jobStepStatus, type PipelineJob } from '@/features/pipeline'

export type JobRunEvent = {
  id: string
  title: string
  detail?: string
  at: string
}

const PIPELINE_STEPS = ['Accepted', 'Queued', 'Running', 'Outputs ready'] as const

export function pipelineActiveStep(status: string | undefined): number {
  switch (status) {
    case 'submitted':
      return 1
    case 'running':
      return 2
    case 'completed':
      return 3
    default:
      return 0
  }
}

export function pipelineStepStatuses(status: string | undefined): Array<ReturnType<typeof jobStepStatus> | undefined> {
  const terminal = jobStepStatus(status)
  const active = pipelineActiveStep(status)
  return PIPELINE_STEPS.map((_, index) => {
    if (index < active) return 'success'
    if (index === active && terminal) return terminal
    return undefined
  })
}

/** Placeholder activity log until workflow events are exposed by the API. */
export function mockJobRunEvents(job: PipelineJob): JobRunEvent[] {
  const created = job.dateCreated ?? new Date().toISOString()
  const updated = job.dateUpdated ?? created
  const events: JobRunEvent[] = [
    {
      id: `${job.id}-created`,
      title: 'Job created',
      detail: job.sourceRef,
      at: created,
    },
  ]

  if (job.workflowId) {
    events.push({
      id: `${job.id}-workflow`,
      title: 'Workflow started',
      detail: job.workflowId,
      at: created,
    })
  }

  if (job.status === 'running' || isJobSuccess(job.status) || job.errorMessage) {
    events.push({
      id: `${job.id}-running`,
      title: 'Pipeline running',
      at: updated,
    })
  }

  if (isJobSuccess(job.status)) {
    events.push({
      id: `${job.id}-done`,
      title: 'Run completed',
      detail: 'Artifacts are ready (mock listing).',
      at: updated,
    })
  } else if (job.errorMessage) {
    events.push({
      id: `${job.id}-failed`,
      title: 'Run failed',
      detail: job.errorMessage,
      at: updated,
    })
  }

  return events
}

export { PIPELINE_STEPS }
