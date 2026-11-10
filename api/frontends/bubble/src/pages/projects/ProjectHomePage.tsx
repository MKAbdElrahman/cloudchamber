import { useQuery } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { QueueListIcon } from '@heroicons/react/24/outline'

import { Grid } from '@astryxdesign/core/Grid'
import { HStack, VStack } from '@astryxdesign/core/Layout'
import { Button } from '@astryxdesign/core/Button'
import { ClickableCard } from '@astryxdesign/core/ClickableCard'
import { EmptyState } from '@astryxdesign/core/EmptyState'
import { Heading, Text } from '@astryxdesign/core/Text'
import { Icon } from '@astryxdesign/core/Icon'
import { List, ListItem } from '@astryxdesign/core/List'
import { Timestamp } from '@astryxdesign/core/Timestamp'
import { JobStatusLabel, listJobs, normalizeJobs, type PipelineJob } from '@/features/pipeline'
import { claimUnassignedJobs, DEFAULT_PROJECT_ID, listProjectJobIds, useProject } from '@/features/projects'
import { PageHeader, PendingLines, WorkspacePage, WORKSPACE_PATHS } from '@/shared/layout'
import { fileNameFromSourceRef } from '@/shared/media'
import { WORKSPACE_CARD, WORKSPACE_GAP } from '@/shared/spacing'
import { MUTED_ICON_COLOR } from '@/shared/theme'

import { LibraryViewControl } from './LibraryViewControl'
import { useLibraryView } from './useLibraryView'
import { ProjectTrail } from './ProjectTrail'

type ProjectHomePageProps = {
  token: string
}

function jobLabel(job: PipelineJob): string {
  return fileNameFromSourceRef(job.sourceRef)
}

/** One project — the jobs it owns. A job opens its status page. */
export function ProjectHomePage({ token }: ProjectHomePageProps) {
  const { projectId = '' } = useParams()
  const project = useProject(projectId)
  const [view, setView] = useLibraryView()

  const jobsQuery = useQuery({
    queryKey: ['pipeline-jobs', projectId],
    queryFn: async () => {
      const jobs = normalizeJobs(await listJobs(token))
      await claimUnassignedJobs(
        jobs.map((job) => job.id),
        DEFAULT_PROJECT_ID,
      )
      const ids = new Set(await listProjectJobIds(projectId))
      return jobs.filter((job) => ids.has(job.id))
    },
    enabled: Boolean(projectId),
    staleTime: 30_000,
  })

  const jobs = jobsQuery.data ?? []

  return (
    <WorkspacePage
      header={
        <VStack gap={WORKSPACE_GAP.group} width="100%">
          <ProjectTrail projectId={projectId} />
          <PageHeader
            title={project?.name ?? 'Project'}
            description="Open a job to see its status and source."
            end={
              <Button
                label="New job"
                variant="primary"
                href={WORKSPACE_PATHS.pipeline(projectId)}
              />
            }
          />
        </VStack>
      }
    >
      <VStack gap={WORKSPACE_GAP.group}>
        <HStack justify="between" vAlign="center">
          <Heading level={2}>Jobs</Heading>
          {jobs.length > 0 ? <LibraryViewControl value={view} onChange={setView} /> : null}
        </HStack>
        {jobsQuery.isPending ? <PendingLines /> : null}
        {jobs.length === 0 && !jobsQuery.isPending ? (
          <EmptyState
            title="No jobs yet"
            description="Start a job to process one recording."
            icon={<Icon icon={QueueListIcon} size="lg" color={MUTED_ICON_COLOR} />}
            isCompact
          />
        ) : null}
        {jobs.length > 0 && view === 'list' ? (
          <List density="spacious" edge="flush" hasDividers aria-label="Jobs">
            {jobs.map((job) => (
              <ListItem
                key={job.id}
                label={jobLabel(job)}
                description={
                  job.dateCreated ? (
                    <Text type="supporting">
                      <Timestamp value={job.dateCreated} format="relative" />
                    </Text>
                  ) : undefined
                }
                startContent={<Icon icon={QueueListIcon} size="sm" color={MUTED_ICON_COLOR} />}
                endContent={<JobStatusLabel status={job.status} />}
                href={WORKSPACE_PATHS.historyJob(projectId, job.id)}
              />
            ))}
          </List>
        ) : null}
        {jobs.length > 0 && view === 'grid' ? (
          <Grid columns={{ minWidth: 240, max: 4 }} gap={WORKSPACE_GAP.group} width="100%" aria-label="Jobs">
            {jobs.map((job) => (
              <ClickableCard
                key={job.id}
                label={`Open ${jobLabel(job)}`}
                href={WORKSPACE_PATHS.historyJob(projectId, job.id)}
                {...WORKSPACE_CARD}
                height="100%"
              >
                <VStack gap={WORKSPACE_GAP.group}>
                  <HStack justify="between" vAlign="start">
                    <Icon icon={QueueListIcon} size="sm" color={MUTED_ICON_COLOR} />
                    <JobStatusLabel status={job.status} />
                  </HStack>
                  <Heading level={3} maxLines={2}>
                    {jobLabel(job)}
                  </Heading>
                  {job.dateCreated ? (
                    <Text type="supporting">
                      <Timestamp value={job.dateCreated} format="relative" />
                    </Text>
                  ) : null}
                </VStack>
              </ClickableCard>
            ))}
          </Grid>
        ) : null}
      </VStack>
    </WorkspacePage>
  )
}
