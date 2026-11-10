import { useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { ClockIcon, QueueListIcon } from '@heroicons/react/24/outline'

import { VStack } from '@astryxdesign/core/Layout'
import { Banner } from '@astryxdesign/core/Banner'
import { EmptyState } from '@astryxdesign/core/EmptyState'
import { Icon } from '@astryxdesign/core/Icon'
import { List, ListItem } from '@astryxdesign/core/List'
import { Text } from '@astryxdesign/core/Text'
import { Timestamp } from '@astryxdesign/core/Timestamp'

import { JobStatusLabel, listJobs, normalizeJobs } from '@/features/pipeline'
import { PageHeader, WorkspacePage, WORKSPACE_PATHS } from '@/shared/layout'
import { WORKSPACE_GAP } from '@/shared/spacing'
import { MUTED_ICON_COLOR, SHELL_ICON_COLOR } from '@/shared/theme'

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
    <WorkspacePage
      header={
        <PageHeader
          title="History"
          description="Pipeline jobs you have run, newest first."
        />
      }
    >
        <VStack gap={WORKSPACE_GAP.group}>
          {jobsQuery.isError ? (
            <Banner
              status="error"
              title="Could not load history"
              description={(jobsQuery.error as Error).message}
            />
          ) : null}

          {jobsQuery.isPending ? (
            <Text type="supporting">Loading job history…</Text>
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
            <List density="balanced" edge="flush" hasDividers aria-label="Job history">
              {jobs.map((job) => (
                <ListItem
                  key={job.id}
                  label={job.id}
                  description={
                    <Text type="supporting">
                      {job.sourceRef}
                      {job.dateCreated ? (
                        <>
                          {' '}
                          · <Timestamp value={job.dateCreated} format="relative" />
                        </>
                      ) : null}
                    </Text>
                  }
                  startContent={<Icon icon={ClockIcon} size="sm" color={SHELL_ICON_COLOR} />}
                  endContent={<JobStatusLabel status={job.status} />}
                  onClick={() => navigate(WORKSPACE_PATHS.historyJob(job.id))}
                />
              ))}
            </List>
          ) : null}
        </VStack>
    </WorkspacePage>
  )
}
