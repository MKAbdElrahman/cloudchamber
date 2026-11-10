import { useState } from 'react'
import { useParams } from 'react-router-dom'

import { VStack } from '@astryxdesign/core/Layout'
import { Banner } from '@astryxdesign/core/Banner'
import { MetadataList, MetadataListItem } from '@astryxdesign/core/MetadataList'
import { Tab, TabList } from '@astryxdesign/core/TabList'
import { Text } from '@astryxdesign/core/Text'
import { Timestamp } from '@astryxdesign/core/Timestamp'
import { JobStatusLabel, usePipelineJob } from '@/features/pipeline'
import { ForeignProjectNotice, useJobOwner } from '@/features/projects'
import { JobInputMedia } from '@/features/upload'
import { PageHeader, PendingLines, WorkspacePage } from '@/shared/layout'
import { WORKSPACE_GAP } from '@/shared/spacing'
import { fileNameFromSourceRef } from '@/shared/media'

import { ProjectTrail } from '../projects/ProjectTrail'

type HistoryJobTab = 'status' | 'input'

type HistoryJobPageProps = {
  token: string
}

/** One job: status and the uploaded source. Nothing here is invented beyond the job record. */
export function HistoryJobPage({ token }: HistoryJobPageProps) {
  const { jobId: jobIdParam, projectId = '' } = useParams()
  const jobId = jobIdParam?.trim() ?? ''
  const owner = useJobOwner(jobId)
  const [tab, setTab] = useState<HistoryJobTab>('status')

  const jobQuery = usePipelineJob(token, jobId || null)
  const job = jobQuery.data
  const inProject = owner.data === projectId
  const visibleJob = inProject ? job : undefined
  const pending = owner.isPending || (inProject && jobQuery.isPending && !job)

  if (!jobId) {
    return (
      <WorkspacePage header={<PageHeader title="Missing job" />}>
        <VStack gap={WORKSPACE_GAP.group}>
          <ProjectTrail projectId={projectId} current="Missing job" />
          <Banner status="error" title="Missing job" description="No job id in the URL." />
        </VStack>
      </WorkspacePage>
    )
  }

  return (
    <WorkspacePage
      header={
        <VStack gap={WORKSPACE_GAP.group} width="100%">
          <ProjectTrail
            projectId={projectId}
            current={visibleJob ? fileNameFromSourceRef(visibleJob.sourceRef) : 'Job'}
          />
          <PageHeader
            title={
              visibleJob
                ? fileNameFromSourceRef(visibleJob.sourceRef)
                : pending
                  ? 'Job'
                  : 'Another project'
            }
            description={
              visibleJob ? (
                <Text type="code">{visibleJob.id}</Text>
              ) : pending ? undefined : (
                'This project cannot show this job.'
              )
            }
            end={visibleJob ? <JobStatusLabel status={visibleJob.status} /> : undefined}
          />
          {inProject ? (
            <TabList
              layout="fill"
              value={tab}
              onChange={(value) => setTab(value as HistoryJobTab)}
              aria-label="Job sections"
            >
              <Tab value="status" label="Status" />
              <Tab value="input" label="Input" />
            </TabList>
          ) : null}
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

        {pending ? <PendingLines /> : null}

        {owner.data && owner.data !== projectId ? (
          <ForeignProjectNotice ownerId={owner.data} jobId={jobId} />
        ) : null}

        {visibleJob && tab === 'status' ? (
          <VStack gap={WORKSPACE_GAP.group}>
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
      </VStack>
    </WorkspacePage>
  )
}
