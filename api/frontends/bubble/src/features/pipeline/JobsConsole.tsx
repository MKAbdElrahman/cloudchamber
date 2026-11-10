import type { ReactNode } from 'react'
import type { UseQueryResult } from '@tanstack/react-query'
import { useQueryClient } from '@tanstack/react-query'
import { HStack, VStack } from '@astryxdesign/core/Layout'
import { Heading, Text } from '@astryxdesign/core/Text'
import { Button } from '@astryxdesign/core/Button'
import { Card } from '@astryxdesign/core/Card'
import { EmptyState } from '@astryxdesign/core/EmptyState'
import { Icon } from '@astryxdesign/core/Icon'
import { MetadataList, MetadataListItem } from '@astryxdesign/core/MetadataList'
import { Banner } from '@astryxdesign/core/Banner'
import { Timestamp } from '@astryxdesign/core/Timestamp'
import { QueueListIcon, ArrowPathIcon, CpuChipIcon } from '@heroicons/react/24/outline'

import { PendingLines } from '@/shared/layout'
import { WORKSPACE_CARD, WORKSPACE_GAP } from '@/shared/spacing'
import { MUTED_ICON_COLOR } from '@/shared/theme'

import { JobStatusLabel } from './JobStatusLabel'
import { jobOutcomeBanner } from './jobStatus'
import type { PipelineJob } from './types'
import { usePipelineJob } from './usePipelineJob'

type JobsConsoleProps = {
  token: string
  jobId: string | null
  embedded?: boolean
  jobQuery?: UseQueryResult<PipelineJob, Error>
  onGoToJob?: () => void
}

function MonitorEmpty({ onGoToJob }: { onGoToJob?: () => void }) {
  return (
    <EmptyState
      title="No job yet"
      description="Create a job after upload to monitor status here."
      icon={<Icon icon={QueueListIcon} size="lg" color={MUTED_ICON_COLOR} />}
      isCompact
      actions={
        onGoToJob ? (
          <Button label="Create job" variant="secondary" size="lg" icon={<Icon icon={CpuChipIcon} />} onClick={onGoToJob} />
        ) : undefined
      }
    />
  )
}

/** Monitor — status for the current run. */
export function JobsConsole({
  token,
  jobId,
  embedded,
  jobQuery: jobQueryProp,
  onGoToJob,
}: JobsConsoleProps) {
  const qc = useQueryClient()
  const fallbackQuery = usePipelineJob(token, jobQueryProp ? null : jobId)
  const jobQuery = jobQueryProp ?? fallbackQuery
  const job = jobQuery.data
  const outcome = job ? jobOutcomeBanner(job.status) : null

  const shell = (body: ReactNode) => {
    if (embedded) return body
    return (
      <Card {...WORKSPACE_CARD}>
        {body}
      </Card>
    )
  }

  if (!jobId) {
    return shell(<MonitorEmpty onGoToJob={onGoToJob} />)
  }

  const body = (
    <VStack gap={WORKSPACE_GAP.section}>
      {!embedded ? (
        <HStack vAlign="center" justify="between">
          <Heading level={3}>Current job</Heading>
          <Button
            label="Refresh"
            variant="ghost"
            size="sm"
            icon={<Icon icon={ArrowPathIcon} />}
            isLoading={jobQuery.isFetching}
            onClick={() => {
              void qc.invalidateQueries({ queryKey: ['pipeline-job', jobId] })
              void jobQuery.refetch()
            }}
          />
        </HStack>
      ) : null}

      {jobQuery.isError ? (
        <Banner status="error" title="Could not load job" description={(jobQuery.error as Error).message} />
      ) : null}

      {jobQuery.isPending && !job ? <PendingLines /> : null}

      {job ? (
        <VStack gap={WORKSPACE_GAP.group}>
          <HStack vAlign="center" justify="between" gap={WORKSPACE_GAP.group}>
            <JobStatusLabel status={job.status} />
            {embedded ? (
              <Button
                label="Refresh"
                variant="ghost"
                size="sm"
                icon={<Icon icon={ArrowPathIcon} />}
                isLoading={jobQuery.isFetching}
                onClick={() => {
                  void qc.invalidateQueries({ queryKey: ['pipeline-job', jobId] })
                  void jobQuery.refetch()
                }}
              />
            ) : null}
          </HStack>
          <MetadataList>
            <MetadataListItem label="Job">
              <Text type="code">{job.id}</Text>
            </MetadataListItem>
            <MetadataListItem label="Source">
              <Text type="code">{job.sourceRef}</Text>
            </MetadataListItem>
            {job.workflowId ? <MetadataListItem label="Workflow">{job.workflowId}</MetadataListItem> : null}
            {job.dateUpdated ? (
              <MetadataListItem label="Updated">
                <Timestamp value={job.dateUpdated} format="relative" />
              </MetadataListItem>
            ) : null}
          </MetadataList>
          {job?.errorMessage && outcome ? (
            <Banner status={outcome.status} title={outcome.title} description={job.errorMessage} />
          ) : null}
        </VStack>
      ) : null}
    </VStack>
  )

  return shell(body)
}
