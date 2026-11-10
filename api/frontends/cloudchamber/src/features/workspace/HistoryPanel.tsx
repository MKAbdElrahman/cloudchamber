import { useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { ClockIcon, QueueListIcon } from '@heroicons/react/24/outline'

import { Layout, LayoutContent, LayoutHeader, VStack } from '@astryxdesign/core/Layout'
import { Banner } from '@astryxdesign/core/Banner'
import { EmptyState } from '@astryxdesign/core/EmptyState'
import { Icon } from '@astryxdesign/core/Icon'
import { List, ListItem } from '@astryxdesign/core/List'
import { Text } from '@astryxdesign/core/Text'
import { Timestamp } from '@astryxdesign/core/Timestamp'

import { listJobs, normalizeJobs } from '@/features/pipeline/api'
import { JobStatusLabel } from '@/features/pipeline/JobStatusLabel'
import { PageHeader } from '@/shared/layout/PageHeader'
import { WORKSPACE_CONTENT_PADDING, WORKSPACE_CONTENT_WIDTH, WORKSPACE_GAP } from '@/shared/layout/contentSurface'
import { WORKSPACE_PATHS } from '@/shared/layout/workspacePaths'
import { MUTED_ICON_COLOR } from '@/shared/theme/presentation'

type HistoryPanelProps = {
  token: string
}

/** Recent pipeline jobs — open a run detail page. */
export function HistoryPanel({ token }: HistoryPanelProps) {
  const navigate = useNavigate()

  const jobsQuery = useQuery({
    queryKey: ['pipeline-jobs', 'history'],
    queryFn: () => listJobs(token),
    staleTime: 30_000,
    select: normalizeJobs,
  })

  const jobs = jobsQuery.data ?? []

  return (
    <Layout height="fill" contentWidth={WORKSPACE_CONTENT_WIDTH}>
      <LayoutHeader hasDivider>
        <PageHeader
          title="History"
          description="Pipeline jobs you have run, newest first."
        />
      </LayoutHeader>
      <LayoutContent padding={WORKSPACE_CONTENT_PADDING}>
        <VStack gap={WORKSPACE_GAP.group}>
          {jobsQuery.isError ? (
            <Banner
              status="error"
              title="Could not load history"
              description={(jobsQuery.error as Error).message}
            />
          ) : null}

          {jobsQuery.isPending ? (
            <Text type="supporting" color="secondary">
              Loading job history…
            </Text>
          ) : null}

          {jobs.length === 0 && !jobsQuery.isPending ? (
            <EmptyState
              title="No jobs yet"
              description="Create a job from the pipeline after you upload a recording."
              icon={<Icon icon={QueueListIcon} size="lg" color={MUTED_ICON_COLOR} />}
              isCompact
            />
          ) : null}

          {jobs.length > 0 ? (
            <List density="spacious" aria-label="Job history">
              {jobs.map((job) => (
                <ListItem
                  key={job.id}
                  label={job.id}
                  description={
                    <Text type="supporting" color="secondary" size="sm">
                      {job.sourceRef}
                      {job.dateCreated ? (
                        <>
                          {' '}
                          · <Timestamp value={job.dateCreated} format="relative" />
                        </>
                      ) : null}
                    </Text>
                  }
                  startContent={<Icon icon={ClockIcon} size="sm" color={MUTED_ICON_COLOR} />}
                  endContent={<JobStatusLabel status={job.status} />}
                  onClick={() => navigate(WORKSPACE_PATHS.historyJob(job.id))}
                />
              ))}
            </List>
          ) : null}
        </VStack>
      </LayoutContent>
    </Layout>
  )
}
