import { useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { InboxArrowDownIcon, QueueListIcon } from '@heroicons/react/24/outline'

import { Layout, LayoutContent, LayoutHeader, VStack } from '@astryxdesign/core/Layout'
import { Banner } from '@astryxdesign/core/Banner'
import { BreadcrumbItem, Breadcrumbs } from '@astryxdesign/core/Breadcrumbs'
import { EmptyState } from '@astryxdesign/core/EmptyState'
import { Icon } from '@astryxdesign/core/Icon'
import { List, ListItem } from '@astryxdesign/core/List'
import { MetadataList, MetadataListItem } from '@astryxdesign/core/MetadataList'
import { Step, Stepper } from '@astryxdesign/core/Stepper'
import { Tab, TabList } from '@astryxdesign/core/TabList'
import { Text } from '@astryxdesign/core/Text'
import { Timestamp } from '@astryxdesign/core/Timestamp'
import { JobStatusLabel } from '@/features/pipeline/JobStatusLabel'
import { usePipelineJob } from '@/features/pipeline/usePipelineJob'
import { JobInputMedia } from '@/features/upload/JobInputMedia'
import { fileNameFromSourceRef } from '@/features/upload/sourceMedia'
import { PageHeader } from '@/shared/layout/PageHeader'
import {
  WORKSPACE_CONTENT_PADDING,
  WORKSPACE_CONTENT_WIDTH,
  WORKSPACE_GAP,
} from '@/shared/layout/contentSurface'
import { MUTED_ICON_COLOR } from '@/shared/theme/presentation'

import {
  mockJobRunEvents,
  pipelineActiveStep,
  pipelineStepStatuses,
  PIPELINE_STEPS,
} from './jobRunMock'
import { JobOutputsPanel } from './JobOutputsPanel'
import { WORKSPACE_PATHS } from '@/shared/layout/workspacePaths'

import { JobRunBreadcrumbs } from './JobRunBreadcrumbs'
import { outputsForJob } from './mockOutputs'

type HistoryJobTab = 'status' | 'input' | 'output' | 'history'

type HistoryJobPageProps = {
  token: string
}

/** Single job run — input, outputs (mock), status, and activity. */
export function HistoryJobPage({ token }: HistoryJobPageProps) {
  const { jobId: jobIdParam } = useParams()
  const jobId = jobIdParam?.trim() ?? ''
  const [tab, setTab] = useState<HistoryJobTab>('status')

  const jobQuery = usePipelineJob(token, jobId || null)
  const job = jobQuery.data

  const outputs = useMemo(
    () => (job ? outputsForJob(job.id, job.status) : []),
    [job],
  )
  const events = useMemo(() => (job ? mockJobRunEvents(job) : []), [job])
  const stepStatuses = useMemo(() => pipelineStepStatuses(job?.status), [job?.status])
  const activeStep = pipelineActiveStep(job?.status)

  if (!jobId) {
    return (
      <Layout height="fill" contentWidth={WORKSPACE_CONTENT_WIDTH}>
        <LayoutContent padding={WORKSPACE_CONTENT_PADDING}>
          <Breadcrumbs variant="supporting" aria-label="Job run navigation">
            <BreadcrumbItem href={WORKSPACE_PATHS.history}>History</BreadcrumbItem>
            <BreadcrumbItem isCurrent>Missing job</BreadcrumbItem>
          </Breadcrumbs>
          <Banner status="error" title="Missing job" description="No job id in the URL." />
        </LayoutContent>
      </Layout>
    )
  }

  return (
    <Layout height="fill" contentWidth={WORKSPACE_CONTENT_WIDTH}>
      <LayoutHeader hasDivider>
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
                <Text type="code" size="sm">
                  {job.id}
                </Text>
              ) : (
                'Loading job details…'
              )
            }
            end={job ? <JobStatusLabel status={job.status} /> : undefined}
          />
          <TabList
            value={tab}
            onChange={(value) => setTab(value as HistoryJobTab)}
            aria-label="Job run sections"
          >
            <Tab value="status" label="Status" />
            <Tab value="input" label="Input" />
            <Tab value="output" label="Output" />
            <Tab value="history" label="History" />
          </TabList>
        </VStack>
      </LayoutHeader>
      <LayoutContent padding={WORKSPACE_CONTENT_PADDING}>
        <VStack gap={WORKSPACE_GAP.group}>
          {jobQuery.isError ? (
            <Banner
              status="error"
              title="Could not load job"
              description={(jobQuery.error as Error).message}
            />
          ) : null}

          {jobQuery.isPending && !job ? (
            <Text type="supporting" color="secondary">
              Loading job…
            </Text>
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
                  <Text type="code" size="sm">
                    {job.id}
                  </Text>
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

          {job && tab === 'output' ? (
            outputs.length > 0 ? (
              <JobOutputsPanel files={outputs} />
            ) : (
              <EmptyState
                title="No outputs yet"
                description={
                  job.status === 'completed'
                    ? 'Outputs will appear here when artifact listing is wired to the API.'
                    : 'Outputs are generated when the pipeline run completes.'
                }
                icon={<Icon icon={InboxArrowDownIcon} size="lg" color={MUTED_ICON_COLOR} />}
                isCompact
              />
            )
          ) : null}

          {job && tab === 'history' ? (
            <List density="spacious" aria-label="Run history">
              {events.map((event) => (
                <ListItem
                  key={event.id}
                  label={event.title}
                  description={
                    event.detail ? (
                      <Text type="supporting" color="secondary" size="sm">
                        {event.detail}
                      </Text>
                    ) : undefined
                  }
                  startContent={<Icon icon={QueueListIcon} size="sm" color={MUTED_ICON_COLOR} />}
                  endContent={<Timestamp value={event.at} format="relative" />}
                />
              ))}
            </List>
          ) : null}
        </VStack>
      </LayoutContent>
    </Layout>
  )
}
