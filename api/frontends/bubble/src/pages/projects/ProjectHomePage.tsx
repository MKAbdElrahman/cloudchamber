import { useEffect, useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { Square2StackIcon } from '@heroicons/react/24/outline'

import { Banner } from '@astryxdesign/core/Banner'
import { Card } from '@astryxdesign/core/Card'
import { Grid } from '@astryxdesign/core/Grid'
import { HStack, VStack } from '@astryxdesign/core/Layout'
import { Button } from '@astryxdesign/core/Button'
import { ClickableCard } from '@astryxdesign/core/ClickableCard'
import { EmptyState } from '@astryxdesign/core/EmptyState'
import { Heading, Text } from '@astryxdesign/core/Text'
import { Icon } from '@astryxdesign/core/Icon'
import { List, ListItem } from '@astryxdesign/core/List'
import { Section } from '@astryxdesign/core/Section'
import { Timestamp } from '@astryxdesign/core/Timestamp'
import { JobOpenIcons, JobPeek, JobStatusLabel, jobPhase, visibleJobStatus, type PeekVideo, type PipelineJob } from '@/features/pipeline'
import { ProjectTrail, useProject, useProjectRecordings, type ProjectRecording } from '@/features/projects'
import { faceFileName, filesFromJob, knownMediaName, mediaKindOf } from '@/features/outputs'
import { PageHeader, PendingLines, WorkspacePage, WORKSPACE_CONTENT_PADDING, WORKSPACE_PATHS } from '@/shared/layout'
import type { SourceMediaKind } from '@/shared/media'
import { WORKSPACE_CARD, WORKSPACE_GAP } from '@/shared/spacing'
import { MUTED_ICON_COLOR } from '@/shared/theme'

import { AddRecordingDialog } from './AddRecordingDialog'
import { LibraryViewControl } from './LibraryViewControl'
import { useLibraryView } from './useLibraryView'
import { useProjectJobs } from './useProjectJobs'

type ProjectHomePageProps = {
  token: string
}

function EntryWhen({ entry }: { entry: LibraryEntry }) {
  return (
    <Text type="supporting">
      {entry.detail ? entry.detail : null}
      {entry.detail && entry.date ? ' · ' : null}
      {entry.date ? <Timestamp value={entry.date} format="relative" /> : null}
    </Text>
  )
}

type LibraryEntry = {
  id: string
  jobId?: string
  fileId?: string
  fileName?: string
  recordingName?: string
  name: string
  href?: string
  date?: string
  detail: string
  kind: SourceMediaKind
  status?: string
  showStatus: boolean
}

function containersFor(projectId: string, jobs: PipelineJob[], recordings: ProjectRecording[]): LibraryEntry[] {
  const recordingByJob = new Map(
    recordings.flatMap((recording) => (recording.jobId ? [[recording.jobId, recording] as const] : [])),
  )
  const recordingByFile = new Map(
    recordings.flatMap((recording) => (recording.fileId ? [[recording.fileId, recording] as const] : [])),
  )
  const linked = new Set<string>()
  const fromJobs = jobs.map((job) => {
    const recording =
      recordingByJob.get(job.id) ?? (job.fileId ? recordingByFile.get(job.fileId) : undefined)
    if (recording) linked.add(recording.id)
    const fileName = job.fileName || recording?.name
    const media = knownMediaName(fileName, recording?.name)
    const document = filesFromJob(job, media ?? undefined)[0]
    const date = job.dateUpdated ?? recording?.dateAdded
    const status = visibleJobStatus(job)
    const name = faceFileName({
      face: 'library',
      fileName,
      recordingName: recording?.name,
      documentName: document?.name,
      date,
    })
    return {
      id: job.id,
      jobId: job.id,
      fileId: job.fileId ?? recording?.fileId,
      fileName,
      recordingName: recording?.name,
      name,
      href: WORKSPACE_PATHS.job(projectId, job.id),
      date,
      detail: media && media !== name ? media : '',
      kind: mediaKindOf(fileName, recording?.name),
      status,
      showStatus: jobPhase(status) !== 'finished',
    }
  })
  const drafts = recordings
    .filter((recording) => !recording.jobId && !linked.has(recording.id))
    .map((recording) => {
      const media = knownMediaName(recording.name, recording.name)
      const name = faceFileName({
        face: 'video',
        fileName: recording.name,
        recordingName: recording.name,
        date: recording.dateAdded,
      })
      return {
        id: recording.id,
        fileId: recording.fileId,
        fileName: recording.name,
        recordingName: recording.name,
        name,
        href: WORKSPACE_PATHS.recording(projectId, recording.id),
        date: recording.dateAdded,
        detail: media && media !== name ? media : '',
        kind: mediaKindOf(recording.name, recording.name),
        showStatus: false,
      }
    })
  return [...fromJobs, ...drafts].sort((a, b) => (b.date ?? '').localeCompare(a.date ?? ''))
}

function uploadingEntry(fileName: string): LibraryEntry {
  return {
    id: 'upload-in-progress',
    fileName,
    recordingName: fileName,
    name: fileName,
    detail: '',
    kind: mediaKindOf(undefined, fileName),
    status: 'uploading',
    showStatus: true,
  }
}

/** One project. Each job is one file that holds its recording and, once written, its document. */
export function ProjectHomePage({ token }: ProjectHomePageProps) {
  const { projectId = '' } = useParams()
  const project = useProject(projectId)
  const jobsQuery = useProjectJobs(token, projectId)
  const recordingsQuery = useProjectRecordings(projectId)
  const [view, setView] = useLibraryView()
  const [searchParams, setSearchParams] = useSearchParams()
  const askedToAdd = searchParams.get('add') === 'recording'
  const [addOpen, setAddOpen] = useState(askedToAdd)
  const [formKey, setFormKey] = useState(0)
  const [startError, setStartError] = useState<string | null>(null)
  const [video, setVideo] = useState<PeekVideo | null>(null)
  const [videoOpen, setVideoOpen] = useState(false)
  const [pipeline, setPipeline] = useState<{ id: string; name: string } | null>(null)
  const [pipelineOpen, setPipelineOpen] = useState(false)
  const [uploadingFile, setUploadingFile] = useState<string | null>(null)

  function openPeek(entry: LibraryEntry, target: 'video' | 'pipeline') {
    if (target === 'video') {
      if (!entry.fileId) return
      setVideo({
        name: faceFileName({
          face: 'video',
          fileName: entry.fileName,
          recordingName: entry.recordingName,
          date: entry.date,
        }),
        fileId: entry.fileId,
      })
      setVideoOpen(true)
      return
    }
    if (!entry.jobId) return
    setPipeline({ id: entry.jobId, name: entry.name })
    setPipelineOpen(true)
  }
  const listed = containersFor(projectId, jobsQuery.data ?? [], recordingsQuery.data ?? [])
  const entries = uploadingFile ? [uploadingEntry(uploadingFile), ...listed] : listed
  const pending = jobsQuery.isPending || recordingsQuery.isPending

  if (askedToAdd && !addOpen) {
    setFormKey((current) => current + 1)
    setStartError(null)
    setAddOpen(true)
  }

  useEffect(() => {
    if (!askedToAdd) return
    const next = new URLSearchParams(searchParams)
    next.delete('add')
    setSearchParams(next, { replace: true })
  }, [askedToAdd, searchParams, setSearchParams])

  function openAddRecording() {
    setFormKey((current) => current + 1)
    setStartError(null)
    setAddOpen(true)
  }

  return (
    <WorkspacePage
      header={
        <VStack gap={WORKSPACE_GAP.group} width="100%">
          <ProjectTrail projectId={projectId} />
          <PageHeader
            title={project?.name ?? 'Project'}
            description="One file for each recording."
            end={<Button label="Add recording" variant="primary" onClick={openAddRecording} />}
          />
        </VStack>
      }
    >
      <VStack gap={WORKSPACE_GAP.group}>
        {startError ? (
          <Banner
            status="error"
            title="The recording was saved"
            description={`${startError} You can start the conversion from the file.`}
          />
        ) : null}
        {entries.length > 0 ? (
          <HStack justify="end" vAlign="center">
            <LibraryViewControl value={view} onChange={setView} />
          </HStack>
        ) : null}
        {pending ? <PendingLines /> : null}
        {entries.length === 0 && !pending ? (
          <EmptyState
            title="No jobs yet"
            description="Add a recording. It shows up here as one file, and you can turn it into a document when you want."
            icon={<Icon icon={Square2StackIcon} size="lg" color={MUTED_ICON_COLOR} />}
            isCompact
          />
        ) : null}
        {entries.length > 0 && view === 'list' ? (
          <Section variant="transparent" padding={WORKSPACE_CONTENT_PADDING}>
            <List density="spacious" edge="flush" hasDividers aria-label="Jobs">
              {entries.map((entry) => (
                <ListItem
                  key={entry.id}
                  label={entry.name}
                  description={<EntryWhen entry={entry} />}
                  startContent={<Icon icon={Square2StackIcon} size="sm" color={MUTED_ICON_COLOR} />}
                  endContent={
                    <HStack gap={WORKSPACE_GAP.local} vAlign="center">
                      {entry.fileId || entry.jobId ? (
                        <JobOpenIcons
                          kind={entry.kind}
                          hasPipeline={Boolean(entry.jobId)}
                          status={entry.status}
                          onOpen={(target) => openPeek(entry, target)}
                        />
                      ) : null}
                      {entry.showStatus && entry.status ? <JobStatusLabel status={entry.status} /> : null}
                    </HStack>
                  }
                  href={entry.href}
                />
              ))}
            </List>
          </Section>
        ) : null}
        {entries.length > 0 && view === 'grid' ? (
          <Grid columns={{ minWidth: 240, max: 4 }} gap={WORKSPACE_GAP.group} width="100%" aria-label="Jobs">
            {entries.map((entry) => {
              const body = (
                <VStack gap={WORKSPACE_GAP.group}>
                  <HStack justify="between" vAlign="start">
                    <Icon icon={Square2StackIcon} size="sm" color={MUTED_ICON_COLOR} />
                    <HStack gap={WORKSPACE_GAP.local} vAlign="center">
                      {entry.fileId || entry.jobId ? (
                        <JobOpenIcons
                          kind={entry.kind}
                          hasPipeline={Boolean(entry.jobId)}
                          status={entry.status}
                          onOpen={(target) => openPeek(entry, target)}
                        />
                      ) : null}
                      {entry.showStatus && entry.status ? <JobStatusLabel status={entry.status} /> : null}
                    </HStack>
                  </HStack>
                  <Heading level={3} maxLines={2}>
                    {entry.name}
                  </Heading>
                  <EntryWhen entry={entry} />
                </VStack>
              )
              if (!entry.href) {
                return (
                  <Card key={entry.id} {...WORKSPACE_CARD} height="100%">
                    {body}
                  </Card>
                )
              }
              return (
                <ClickableCard key={entry.id} label={`Open ${entry.name}`} href={entry.href} {...WORKSPACE_CARD} height="100%">
                  {body}
                </ClickableCard>
              )
            })}
          </Grid>
        ) : null}
      </VStack>
      <JobPeek
        token={token}
        video={video}
        videoOpen={videoOpen}
        onVideoOpenChange={setVideoOpen}
        pipelineJobId={pipeline?.id ?? null}
        pipelineName={pipeline?.name ?? 'Pipeline'}
        pipelineOpen={pipelineOpen}
        onPipelineOpenChange={setPipelineOpen}
      />
      <AddRecordingDialog
        token={token}
        projectId={projectId}
        isOpen={addOpen}
        formKey={formKey}
        onOpenChange={setAddOpen}
        onUploadingFile={setUploadingFile}
        onFinished={(notice) => {
          setUploadingFile(null)
          setAddOpen(false)
          setStartError(notice ?? null)
        }}
      />
    </WorkspacePage>
  )
}
