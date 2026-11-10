import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { QueueListIcon } from '@heroicons/react/24/outline'

import { HStack, VStack } from '@astryxdesign/core/Layout'
import { Banner } from '@astryxdesign/core/Banner'
import { Button } from '@astryxdesign/core/Button'
import { Icon } from '@astryxdesign/core/Icon'
import { List, ListItem } from '@astryxdesign/core/List'
import { MetadataList, MetadataListItem } from '@astryxdesign/core/MetadataList'
import { Step, Stepper } from '@astryxdesign/core/Stepper'
import { Tab, TabList } from '@astryxdesign/core/TabList'
import { Text } from '@astryxdesign/core/Text'
import { Timestamp } from '@astryxdesign/core/Timestamp'
import { isJobSuccess, JobStatusLabel, usePipelineJob } from '@/features/pipeline'
import { ForeignProjectNotice, useJobOwner } from '@/features/projects'
import { JobInputMedia } from '@/features/upload'
import { PageHeader, WorkspacePage, WORKSPACE_PATHS } from '@/shared/layout'
import { WORKSPACE_GAP } from '@/shared/spacing'
import { fileNameFromSourceRef } from '@/shared/media'
import { SHELL_ICON_COLOR } from '@/shared/theme'

import { outputsForJob } from '../fixtures/mockOutputs'
import { ProjectTrail } from '../projects/ProjectTrail'
import { readingDocument } from './documentReading'
import { JobRunBreadcrumbs } from './JobRunBreadcrumbs'
import {
  mockJobRunEvents,
  pipelineActiveStep,
  pipelineStepStatuses,
  PIPELINE_STEPS,
} from './jobRunMock'

type HistoryJobTab = 'status' | 'input' | 'history'

type HistoryJobPageProps = {
  token: string
}

/** The job behind a file. Opened only when someone asks to see it. */
export function HistoryJobPage({ token }: HistoryJobPageProps) {
  const navigate = useNavigate()
  const { jobId: jobIdParam, projectId = '' } = useParams()
  const jobId = jobIdParam?.trim() ?? ''
  const owner = useJobOwner(jobId)
  const [tab, setTab] = useState<HistoryJobTab>('status')

  const jobQuery = usePipelineJob(token, jobId || null)
  const job = jobQuery.data
  const inProject = owner.data === projectId
  const visibleJob = inProject ? job : undefined

  const events = useMemo(() => (job ? mockJobRunEvents(job) : []), [job])
  const stepStatuses = useMemo(() => pipelineStepStatuses(job?.status), [job?.status])
  const activeStep = pipelineActiveStep(job?.status)
  const file = visibleJob && isJobSuccess(visibleJob.status) ? outputsForJob(visibleJob.id, visibleJob.status)[0] : undefined
  const fileTitle = file?.markdown ? (readingDocument(file.markdown).title ?? file.name) : file?.name

  if (!jobId) {
    return (
      <WorkspacePage>
          <ProjectTrail projectId={projectId} current="Missing job" />
          <Banner status="error" title="Missing job" description="No job id in the URL." />
      </WorkspacePage>
    )
  }

  return (
    <WorkspacePage
      header={
        <VStack gap={WORKSPACE_GAP.group} width="100%">
          <JobRunBreadcrumbs
            projectId={projectId}
            currentLabel={
              fileTitle ? 'Job' : visibleJob ? fileNameFromSourceRef(visibleJob.sourceRef) : jobId
            }
            parent={
              fileTitle
                ? { label: fileTitle, href: WORKSPACE_PATHS.historyJobDocument(projectId, jobId) }
                : undefined
            }
          />
          <PageHeader
            title={
              visibleJob
                ? fileNameFromSourceRef(visibleJob.sourceRef)
                : owner.isPending || inProject
                  ? jobId
                  : 'Another project'
            }
            description={
              visibleJob ? (
                <Text type="code">{visibleJob.id}</Text>
              ) : owner.isPending || inProject ? (
                'Loading job details…'
              ) : (
                'This project cannot show this job.'
              )
            }
            end={
              visibleJob ? (
                <HStack gap={WORKSPACE_GAP.group} vAlign="center">
                  {fileTitle ? (
                    <Button
                      label="Open file"
                      variant="secondary"
                      onClick={() => navigate(WORKSPACE_PATHS.historyJobDocument(projectId, visibleJob.id))}
                    />
                  ) : null}
                  <JobStatusLabel status={visibleJob.status} />
                </HStack>
              ) : undefined
            }
          />
          {inProject ? <TabList
            layout="fill"
            value={tab}
            onChange={(value) => setTab(value as HistoryJobTab)}
            aria-label="Job run sections"
          >
            <Tab value="status" label="Status" />
            <Tab value="input" label="Input" />
            <Tab value="history" label="History" />
          </TabList> : null}
        </VStack>
      }
    >
        <VStack gap={WORKSPACE_GAP.group}>
          {inProject && jobQuery.isError ? (
            <Banner
              status="error"
              title="Could not load job"
              description={(jobQuery.error as Error).message}
            />
          ) : null}

          {owner.isPending || (inProject && jobQuery.isPending && !job) ? (
            <Text type="supporting">Loading job…</Text>
          ) : null}

          {owner.data && owner.data !== projectId ? (
            <ForeignProjectNotice ownerId={owner.data} jobId={jobId} />
          ) : null}

          {visibleJob && tab === 'status' ? (
            <VStack gap={WORKSPACE_GAP.group}>
              <Stepper activeStep={activeStep} orientation="vertical" density="spacious">
                {PIPELINE_STEPS.map((label, index) => (
                  <Step
                    key={label}
                    step={index}
                    label={label}
                    status={stepStatuses[index]}
                  />
                ))}
              </Stepper>
              <MetadataList>
                <MetadataListItem label="Job">
                  <Text type="code">{visibleJob.id}</Text>
                </MetadataListItem>
                <MetadataListItem label="Status">
                  <JobStatusLabel status={visibleJob.status} />
                </MetadataListItem>
                {visibleJob.workflowId ? (
                  <MetadataListItem label="Workflow">{visibleJob.workflowId}</MetadataListItem>
                ) : null}
                {visibleJob.runId ? <MetadataListItem label="Run">{visibleJob.runId}</MetadataListItem> : null}
                {visibleJob.dateCreated ? (
                  <MetadataListItem label="Created">
                    <Timestamp value={visibleJob.dateCreated} format="date_time" />
                  </MetadataListItem>
                ) : null}
                {visibleJob.dateUpdated ? (
                  <MetadataListItem label="Updated">
                    <Timestamp value={visibleJob.dateUpdated} format="relative" />
                  </MetadataListItem>
                ) : null}
              </MetadataList>
              {visibleJob.errorMessage ? (
                <Banner status="error" title="Error" description={visibleJob.errorMessage} />
              ) : null}
            </VStack>
          ) : null}

          {visibleJob && tab === 'input' ? (
            <VStack gap={WORKSPACE_GAP.group}>
              <JobInputMedia sourceRef={visibleJob.sourceRef} />
              {visibleJob.dateCreated ? (
                <MetadataList>
                  <MetadataListItem label="Uploaded">
                    <Timestamp value={visibleJob.dateCreated} format="date_time" />
                  </MetadataListItem>
                </MetadataList>
              ) : null}
            </VStack>
          ) : null}

          {visibleJob && tab === 'history' ? (
            <List density="compact" edge="flush" hasDividers aria-label="Run history">
              {events.map((event) => (
                <ListItem
                  key={event.id}
                  label={event.title}
                  description={
                    event.detail ? <Text type="supporting">{event.detail}</Text> : undefined
                  }
                  startContent={<Icon icon={QueueListIcon} size="sm" color={SHELL_ICON_COLOR} />}
                  endContent={<Timestamp value={event.at} format="relative" />}
                />
              ))}
            </List>
          ) : null}
        </VStack>
    </WorkspacePage>
  )
}
