import { useQuery } from '@tanstack/react-query'
import { useNavigate, useParams } from 'react-router-dom'
import { DocumentIcon, DocumentTextIcon } from '@heroicons/react/24/outline'

import { Grid } from '@astryxdesign/core/Grid'
import { HStack, VStack } from '@astryxdesign/core/Layout'
import { Button } from '@astryxdesign/core/Button'
import { ClickableCard } from '@astryxdesign/core/ClickableCard'
import { EmptyState } from '@astryxdesign/core/EmptyState'
import { Heading, Text } from '@astryxdesign/core/Text'
import { Icon } from '@astryxdesign/core/Icon'
import { List, ListItem } from '@astryxdesign/core/List'
import { Timestamp } from '@astryxdesign/core/Timestamp'
import { Token } from '@astryxdesign/core/Token'
import { JobStatusLabel, listJobs, normalizeJobs, type PipelineJob } from '@/features/pipeline'
import { claimUnassignedJobs, DEFAULT_PROJECT_ID, listProjectJobIds, useProject } from '@/features/projects'
import { outputFileKindLabel, type OutputFile, type OutputFileKind } from '@/features/outputs'
import { PageHeader, WorkspacePage, WORKSPACE_PATHS } from '@/shared/layout'
import { fileNameFromSourceRef } from '@/shared/media'
import { WORKSPACE_GAP } from '@/shared/spacing'
import { META_TOKEN_COLOR, MUTED_ICON_COLOR } from '@/shared/theme'

import { readingDocument } from '../history/documentReading'
import { outputsForJob } from '../fixtures/mockOutputs'
import { LibraryViewControl } from './LibraryViewControl'
import { useLibraryView } from './useLibraryView'
import { ProjectTrail } from './ProjectTrail'

type ProjectHomePageProps = {
  token: string
}

type ProjectFile = {
  job: PipelineJob
  file: OutputFile
}

function fileIcon(kind: OutputFileKind) {
  return kind === 'markdown' ? DocumentTextIcon : DocumentIcon
}

function fileLabel(file: OutputFile, job: PipelineJob): string {
  if (file.name !== 'pipeline-output.md') return file.name
  const recording = fileNameFromSourceRef(job.sourceRef)
  const stem = recording.replace(/\.[^.]+$/, '')
  return `${stem}.md`
}

function fileTitle(file: OutputFile, job: PipelineJob): string {
  if (!file.markdown) return fileLabel(file, job)
  return readingDocument(file.markdown).title ?? fileLabel(file, job)
}

/** One project — the files it produced. A file opens the document. */
export function ProjectHomePage({ token }: ProjectHomePageProps) {
  const navigate = useNavigate()
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
  const files: ProjectFile[] = []
  const pending: PipelineJob[] = []
  for (const job of jobs) {
    const file = outputsForJob(job.id, job.status)[0]
    if (file) files.push({ job, file })
    else pending.push(job)
  }
  return (
    <WorkspacePage
      header={
        <VStack gap={WORKSPACE_GAP.group} width="100%">
          <ProjectTrail projectId={projectId} />
          <PageHeader
            title={project?.name ?? 'Project'}
            description="Open a file to read it."
            end={
              <Button
                label="New job"
                variant="primary"
                onClick={() => navigate(WORKSPACE_PATHS.pipeline(projectId))}
              />
            }
          />
        </VStack>
      }
    >
      <VStack gap={WORKSPACE_GAP.section}>
        <VStack gap={WORKSPACE_GAP.group}>
          <HStack justify="between" vAlign="center">
            <Heading level={2}>Files</Heading>
            {files.length > 0 ? <LibraryViewControl value={view} onChange={setView} /> : null}
          </HStack>
          {jobsQuery.isPending ? <Text type="supporting">Loading files…</Text> : null}
          {files.length === 0 && !jobsQuery.isPending ? (
            <EmptyState
              title="No files yet"
              description="A finished job leaves one file here."
              icon={<Icon icon={DocumentTextIcon} size="lg" color={MUTED_ICON_COLOR} />}
              isCompact
            />
          ) : null}
          {files.length > 0 && view === 'list' ? (
            <List density="spacious" edge="flush" hasDividers aria-label="Files">
              {files.map(({ job, file }) => (
                <ListItem
                  key={file.id}
                  label={fileTitle(file, job)}
                  description={
                    <Text type="supporting">
                      {fileLabel(file, job)}
                      {job.dateCreated ? (
                        <>
                          {' · '}
                          <Timestamp value={job.dateCreated} format="relative" />
                        </>
                      ) : null}
                    </Text>
                  }
                  startContent={<Icon icon={fileIcon(file.kind)} size="sm" color="secondary" />}
                  endContent={<Token label={outputFileKindLabel(file.kind)} size="sm" color={META_TOKEN_COLOR} />}
                  href={WORKSPACE_PATHS.historyJobDocument(projectId, job.id)}
                />
              ))}
            </List>
          ) : null}
          {files.length > 0 && view === 'grid' ? (
            <Grid columns={{ minWidth: 240 }} gap={4} width="100%" aria-label="Files">
              {files.map(({ job, file }) => (
                <ClickableCard
                  key={file.id}
                  label={`Open ${fileTitle(file, job)}`}
                  href={WORKSPACE_PATHS.historyJobDocument(projectId, job.id)}
                  padding={4}
                  elevation="low"
                  height="100%"
                >
                  <VStack gap={2}>
                    <HStack justify="between" vAlign="start">
                      <Icon icon={fileIcon(file.kind)} size="sm" color="secondary" />
                      <Token label={outputFileKindLabel(file.kind)} size="sm" color={META_TOKEN_COLOR} />
                    </HStack>
                    <Heading level={3} maxLines={2}>
                      {fileTitle(file, job)}
                    </Heading>
                    <Text type="supporting" color="secondary" maxLines={1}>
                      {fileLabel(file, job)}
                    </Text>
                    {job.dateCreated ? (
                      <Text type="supporting" color="secondary">
                        <Timestamp value={job.dateCreated} format="relative" />
                      </Text>
                    ) : null}
                  </VStack>
                </ClickableCard>
              ))}
            </Grid>
          ) : null}
        </VStack>

        {pending.length > 0 ? (
          <VStack gap={WORKSPACE_GAP.group}>
            <Heading level={2}>In progress</Heading>
            <List density="balanced" edge="flush" hasDividers aria-label="Jobs still running">
              {pending.map((job) => (
                <ListItem
                  key={job.id}
                  label={fileNameFromSourceRef(job.sourceRef)}
                  description={
                    <Text type="supporting">
                      {job.dateCreated ? <Timestamp value={job.dateCreated} format="relative" /> : 'Waiting for a file'}
                    </Text>
                  }
                  endContent={<JobStatusLabel status={job.status} />}
                  onClick={() => navigate(WORKSPACE_PATHS.historyJob(projectId, job.id))}
                />
              ))}
            </List>
          </VStack>
        ) : null}
      </VStack>
    </WorkspacePage>
  )
}
