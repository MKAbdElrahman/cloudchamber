/** One step of a run, as the platform last recorded it. */
export type RunStep = {
  key: string
  state: 'working' | 'done' | 'skipped' | 'failed' | 'canceled' | 'paused'
  message?: string
  /** How far a working step has got, from 0 to 1. Only meaningful when `measured` is true. */
  fraction: number
  measured: boolean
  dateStarted: string
  dateUpdated: string
}

export type RunPhase = 'pending' | 'running' | 'paused' | 'succeeded' | 'failed' | 'canceled' | 'terminated'

/**
 * A run of any workflow. `spec` is what was asked for and `status` is what the
 * platform observed. `input` and `result` belong to the workflow, so their shape is
 * the type parameters.
 */
export type Run<Input = unknown, Result = unknown> = {
  id: string
  workflow: string
  ownerId: string
  projectId: string
  generation: number
  /** False while the platform still has work to do: starting, pausing, resuming or stopping. */
  settled: boolean
  spec: {
    input?: Input | null
    suspend: boolean
    cancel: boolean
    attempt: number
    resumeFrom: string
  }
  status: {
    phase: RunPhase
    observedGeneration: number
    attempt: number
    reason?: string
    message?: string
    steps: RunStep[]
    result?: Result | null
    dateClosed?: string
  }
  dateCreated: string
  dateUpdated: string
}

export type RunsPage<Input = unknown, Result = unknown> = {
  items?: Run<Input, Result>[]
  page?: number
  rowsPerPage?: number
  total?: number
}

/** The commands a person can give a run. Each one edits what the run is asked to do. */
export type RunCommand = 'suspend' | 'resume' | 'cancel' | 'retry' | 'restart'

/** One step a workflow may run, with the name a person reads. */
export type WorkflowStep = {
  key: string
  label: string
}

/** One workflow the service can start a run of. */
export type Workflow = {
  name: string
  title: string
  steps: WorkflowStep[]
  /** JSON Schema for the workflow's input. */
  inputSchema?: unknown
}
