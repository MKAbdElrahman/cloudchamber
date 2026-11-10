import { useState } from 'react'
import { VStack } from '@astryxdesign/core/Layout'
import { Step, Stepper } from '@astryxdesign/core/Stepper'

import {
  CreateJobForm,
  JobsConsole,
  JobStatusLabel,
  usePipelineJob,
  usePipelineRun,
} from '@/features/pipeline'
import { addProjectJob, useProject } from '@/features/projects'
import { UploadPanel } from '@/features/upload'
import { PageHeader, WorkspacePage } from '@/shared/layout'
import { ProjectTrail } from '../projects/ProjectTrail'
import { WORKSPACE_GAP } from '@/shared/spacing'

type PipelineWorkspaceProps = {
  token: string
  projectId: string
}

const STAGES = ['Upload', 'Job', 'Status'] as const

/** New job — a real three-step sequence. Each step is the panel under the stepper. */
export function PipelineWorkspace({ token, projectId }: PipelineWorkspaceProps) {
  const project = useProject(projectId)
  const [stage, setStage] = useState(0)
  const [reached, setReached] = useState(0)
  const { sourceRef, setSourceRef, currentJobId, setCurrentJobId } = usePipelineRun()

  const monitoring = stage === 2
  const jobQuery = usePipelineJob(token, monitoring ? currentJobId : null)
  const jobStatus = monitoring ? jobQuery.data?.status : undefined

  function openStage(index: number) {
    if (index > reached) return
    setStage(index)
  }

  function handleSourceRef(ref: string) {
    setSourceRef(ref)
    setReached((current) => Math.max(current, 1))
    setStage(1)
  }

  function handleJobCreated(job: { id: string }) {
    void addProjectJob(projectId, job.id)
    setCurrentJobId(job.id)
    setReached((current) => Math.max(current, 2))
    setStage(2)
  }

  return (
    <WorkspacePage
      header={
        <VStack gap={WORKSPACE_GAP.group} width="100%">
          <ProjectTrail projectId={projectId} current="New job" />
          <PageHeader
            title="New job"
            description={project ? `A new job stays in ${project.name}.` : 'A new job stays in this project.'}
            end={monitoring && currentJobId && jobStatus ? <JobStatusLabel status={jobStatus} /> : null}
          />
          <Stepper activeStep={stage} label="New job stages" onStepClick={openStage}>
            {STAGES.map((label, index) => (
              <Step key={label} step={index} label={label} isDisabled={index > reached} />
            ))}
          </Stepper>
        </VStack>
      }
    >
      <VStack gap={WORKSPACE_GAP.group} width="100%" maxWidth={640}>
        {stage === 0 ? (
          <UploadPanel token={token} embedded onSourceRef={handleSourceRef} />
        ) : null}
        {stage === 1 ? (
          <CreateJobForm
            token={token}
            sourceRef={sourceRef}
            embedded
            onJobCreated={handleJobCreated}
          />
        ) : null}
        {stage === 2 ? (
          <JobsConsole
            token={token}
            jobId={currentJobId}
            jobQuery={jobQuery}
            embedded
            onGoToJob={() => openStage(1)}
          />
        ) : null}
      </VStack>
    </WorkspacePage>
  )
}
