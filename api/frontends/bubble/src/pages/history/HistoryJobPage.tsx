import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { QueueListIcon } from '@heroicons/react/24/outline'

import { HStack, VStack } from '@astryxdesign/core/Layout'
import { Banner } from '@astryxdesign/core/Banner'
import { BreadcrumbItem, Breadcrumbs } from '@astryxdesign/core/Breadcrumbs'
import { Button } from '@astryxdesign/core/Button'
import { Icon } from '@astryxdesign/core/Icon'
import { List, ListItem } from '@astryxdesign/core/List'
import { MetadataList, MetadataListItem } from '@astryxdesign/core/MetadataList'
import { Step, Stepper } from '@astryxdesign/core/Stepper'
import { Tab, TabList } from '@astryxdesign/core/TabList'
import { Text } from '@astryxdesign/core/Text'
import { Timestamp } from '@astryxdesign/core/Timestamp'
import { isJobSuccess, JobStatusLabel, usePipelineJob } from '@/features/pipeline'
import { JobInputMedia } from '@/features/upload'
import { PageHeader, WorkspacePage, WORKSPACE_PATHS } from '@/shared/layout'
import { WORKSPACE_GAP } from '@/shared/spacing'
import { fileNameFromSourceRef } from '@/shared/media'
import { SHELL_ICON_COLOR } from '@/shared/theme'

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

/** Single job run — status, recording, and activity. Read opens the one document. */
export function HistoryJobPage({ token }: HistoryJobPageProps) {
  const navigate = useNavigate()
  const { jobId: jobIdParam } = useParams()
  const jobId = jobIdParam?.trim() ?? ''
  const [tab, setTab] = useState<HistoryJobTab>('status')

  const jobQuery = usePipelineJob(token, jobId || null)
  const job = jobQuery.data

  const events = useMemo(() => (job ? mockJobRunEvents(job) : []), [job])
  const stepStatuses = useMemo(() => pipelineStepStatuses(job?.status), [job?.status])
  const activeStep = pipelineActiveStep(job?.status)

  if (!jobId) {
    return (
      <WorkspacePage>
          <Breadcrumbs variant="supporting" aria-label="Job run navigation">
            <BreadcrumbItem href={WORKSPACE_PATHS.history}>History</BreadcrumbItem>
            <BreadcrumbItem isCurrent>Missing job</BreadcrumbItem>
          </Breadcrumbs>
          <Banner status="error" title="Missing job" description="No job id in the URL." />
      </WorkspacePage>
    )
  }

  return (
    <WorkspacePage
      header={
        <VStack gap={WORKSPACE_GAP.group} width="100%">
          <JobRunBreadcrumbs
            currentLabel={
              job ? fileNameFromSourceRef(job.sourceRef) : jobId
            }
          />
          <PageHeader
            title={job ? fileNameFromSourceRef(job.sourceRef) : jobId}
            description={
              job ? (
                <Text type="code">{job.id}</Text>
              ) : (
                'Loading job details…'
              )
            }
            end={
              job ? (
                <HStack gap={WORKSPACE_GAP.group} vAlign="center">
                  {isJobSuccess(job.status) ? (
                    <Button
                      label="Read"
                      variant="primary"
                      onClick={() => navigate(WORKSPACE_PATHS.historyJobDocument(job.id))}
                    />
                  ) : null}
                  <JobStatusLabel status={job.status} />
                </HStack>
              ) : undefined
            }
          />
          <TabList
            layout="fill"
            value={tab}
            onChange={(value) => setTab(value as HistoryJobTab)}
            aria-label="Job run sections"
          >
            <Tab value="status" label="Status" />
            <Tab value="input" label="Input" />
            <Tab value="history" label="History" />
          </TabList>
        </VStack>
      }
    >
        <VStack gap={WORKSPACE_GAP.group}>
          {jobQuery.isError ? (
            <Banner
              status="error"
              title="Could not load job"
              description={(jobQuery.error as Error).message}
            />
          ) : null}

          {jobQuery.isPending && !job ? (
            <Text type="supporting">Loading job…</Text>
          ) : null}

          {job && tab === 'status' ? (
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
                  <Text type="code">{job.id}</Text>
                </MetadataListItem>
                <MetadataListItem label="Status">
                  <JobStatusLabel status={job.status} />
                </MetadataListItem>
                {job.workflowId ? (
                  <MetadataListItem label="Workflow">{job.workflowId}</MetadataListItem>
                ) : null}
                {job.runId ? <MetadataListItem label="Run">{job.runId}</MetadataListItem> : null}
                {job.dateCreated ? (
                  <MetadataListItem label="Created">
                    <Timestamp value={job.dateCreated} format="date_time" />
                  </MetadataListItem>
                ) : null}
                {job.dateUpdated ? (
                  <MetadataListItem label="Updated">
                    <Timestamp value={job.dateUpdated} format="relative" />
                  </MetadataListItem>
                ) : null}
              </MetadataList>
              {job.errorMessage ? (
                <Banner status="error" title="Error" description={job.errorMessage} />
              ) : null}
            </VStack>
          ) : null}

          {job && tab === 'input' ? (
            <VStack gap={WORKSPACE_GAP.group}>
              <JobInputMedia sourceRef={job.sourceRef} />
              {job.dateCreated ? (
                <MetadataList>
                  <MetadataListItem label="Uploaded">
                    <Timestamp value={job.dateCreated} format="date_time" />
                  </MetadataListItem>
                </MetadataList>
              ) : null}
            </VStack>
          ) : null}

          {job && tab === 'history' ? (
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
