import { useState } from 'react'
import { VStack } from '@astryxdesign/core/Layout'
import { Tab, TabList } from '@astryxdesign/core/TabList'
import { Icon } from '@astryxdesign/core/Icon'
import {
  ArrowUpTrayIcon,
  CpuChipIcon,
  SignalIcon,
} from '@heroicons/react/24/outline'
import {
  ArrowUpTrayIcon as ArrowUpTraySolid,
  CpuChipIcon as CpuChipSolid,
  SignalIcon as SignalSolid,
} from '@heroicons/react/24/solid'
import { CheckCircleIcon as CheckCircleSolid } from '@heroicons/react/24/solid'

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

type PipelineTab = 'upload' | 'job' | 'monitor'

function StepCompleteMark({ done }: { done: boolean }) {
  if (!done) return null
  return <Icon icon={CheckCircleSolid} size="sm" color="success" aria-label="Complete" />
}

/** Pipeline — stage tabs (upload → job → monitor) on a single content surface. */
export function PipelineWorkspace({ token, projectId }: PipelineWorkspaceProps) {
  const project = useProject(projectId)
  const [tab, setTab] = useState<PipelineTab>('upload')
  const { sourceRef, setSourceRef, currentJobId, setCurrentJobId } = usePipelineRun()
  /** Stage checkmarks — session only; persisted sourceRef/job id must not imply “done” on a fresh visit. */
  const [uploadStageDone, setUploadStageDone] = useState(false)
  const [jobStageDone, setJobStageDone] = useState(false)

  const monitoring = tab === 'monitor'
  const jobQuery = usePipelineJob(token, monitoring ? currentJobId : null)
  const jobStatus = monitoring ? jobQuery.data?.status : undefined

  function handleSourceRef(ref: string) {
    setSourceRef(ref)
    setUploadStageDone(true)
    setTab('job')
  }

  function handleJobCreated(job: { id: string }) {
    void addProjectJob(projectId, job.id)
    setCurrentJobId(job.id)
    setJobStageDone(true)
    setTab('monitor')
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
            <TabList
              layout="fill"
              value={tab}
              onChange={(v) => setTab(v as PipelineTab)}
              aria-label="Pipeline stages"
            >
              <Tab
                value="upload"
                label="Upload"
                icon={tab === 'upload' ? <ArrowUpTraySolid /> : <ArrowUpTrayIcon />}
                endContent={<StepCompleteMark done={uploadStageDone} />}
              />
              <Tab
                value="job"
                label="Job"
                icon={tab === 'job' ? <CpuChipSolid /> : <CpuChipIcon />}
                endContent={<StepCompleteMark done={jobStageDone} />}
              />
              <Tab
                value="monitor"
                label="Status"
                icon={tab === 'monitor' ? <SignalSolid /> : <SignalIcon />}
              />
            </TabList>
          </VStack>
      }
    >
          {tab === 'upload' ? (
            <UploadPanel token={token} embedded onSourceRef={handleSourceRef} />
          ) : null}
          {tab === 'job' ? (
            <CreateJobForm
              token={token}
              sourceRef={sourceRef}
              embedded
              onJobCreated={handleJobCreated}
            />
          ) : null}
          {tab === 'monitor' ? (
            <JobsConsole
              token={token}
              jobId={currentJobId}
              jobQuery={jobQuery}
              embedded
              onGoToJob={() => setTab('job')}
            />
          ) : null}
    </WorkspacePage>
  )
}
