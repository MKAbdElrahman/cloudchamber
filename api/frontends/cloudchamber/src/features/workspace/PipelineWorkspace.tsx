import { useState } from 'react'
import {
  Layout,
  LayoutContent,
  LayoutHeader,
  VStack,
} from '@astryxdesign/core/Layout'
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

import { UploadPanel } from '@/features/upload/UploadPanel'
import { CreateJobForm } from '@/features/pipeline/PipelinePanel'
import { JobsConsole } from '@/features/pipeline/JobsConsole'
import { JobStatusLabel } from '@/features/pipeline/JobStatusLabel'
import { usePipelineRun } from '@/features/pipeline/PipelineRunContext'
import { usePipelineJob } from '@/features/pipeline/usePipelineJob'
import { PageHeader } from '@/shared/layout/PageHeader'
import { WORKSPACE_CONTENT_PADDING, WORKSPACE_CONTENT_WIDTH, WORKSPACE_GAP } from '@/shared/layout/contentSurface'

type PipelineWorkspaceProps = {
  token: string
}

type PipelineTab = 'upload' | 'job' | 'monitor'

function StepCompleteMark({ done }: { done: boolean }) {
  if (!done) return null
  return <Icon icon={CheckCircleSolid} size="sm" color="success" aria-label="Complete" />
}

/** Pipeline — stage tabs (upload → job → monitor) on a single content surface. */
export function PipelineWorkspace({ token }: PipelineWorkspaceProps) {
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
    setCurrentJobId(job.id)
    setJobStageDone(true)
    setTab('monitor')
  }

  return (
    <Layout
      height="fill"
      contentWidth={WORKSPACE_CONTENT_WIDTH}
      header={
        <LayoutHeader hasDivider>
          <VStack gap={WORKSPACE_GAP.group} width="100%">
            <PageHeader
              title="Pipeline"
              end={monitoring && currentJobId && jobStatus ? <JobStatusLabel status={jobStatus} /> : null}
            />
            <TabList
              layout="fill"
              hasDivider
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
        </LayoutHeader>
      }
      content={
        <LayoutContent padding={WORKSPACE_CONTENT_PADDING}>
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
        </LayoutContent>
      }
    />
  )
}
