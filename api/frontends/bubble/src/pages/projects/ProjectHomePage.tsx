import { useCallback, useEffect, useRef, useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'

import { Banner } from '@astryxdesign/core/Banner'
import { FileInput } from '@astryxdesign/core/FileInput'
import { HStack, VStack } from '@astryxdesign/core/Layout'
import { Button } from '@astryxdesign/core/Button'
import { EmptyState } from '@astryxdesign/core/EmptyState'
import { Text } from '@astryxdesign/core/Text'
import { List, ListItem } from '@astryxdesign/core/List'
import { Timestamp } from '@astryxdesign/core/Timestamp'
import { JobOpenIcons, JobPeek, JobStatusLabel, jobPhase, visibleJobStatus, type PeekVideo, type Conversion } from '@/features/conversion'
import { ProjectTrail, useProject, type ProjectFile } from '@/features/projects'
import { faceFileName, filesFromJob, knownMediaName, mediaKindOf } from '@/features/outputs'
import { PAGE_CONTENT_WIDTH, PageHeader, PendingLines, WorkspacePage, WORKSPACE_PATHS } from '@/shared/layout'
import type { SourceMediaKind } from '@/shared/media'
import { WORKSPACE_GAP } from '@/shared/spacing'
import { AddRecordingDialog } from './AddRecordingDialog'
import { useProjectFiles } from './useProjectFiles'
import { useProjectJobs } from './useProjectJobs'

type ProjectHomePageProps = {
  token: string
}

function droppedFile(files: File | File[] | null): File | null {
  if (files == null) return null
  return Array.isArray(files) ? files[0] ?? null : files
}

/** True while a file drag is over the window. Cleared on drop or when the pointer leaves. */
function useFileDrag(): { dragging: boolean; clearDrag: () => void } {
  const [dragging, setDragging] = useState(false)
  const depth = useRef(0)
  const clearDrag = useCallback(() => {
    depth.current = 0
    setDragging(false)
  }, [])

  useEffect(() => {
    const hasFiles = (event: DragEvent) => Array.from(event.dataTransfer?.types ?? []).includes('Files')
    const onEnter = (event: DragEvent) => {
      if (!hasFiles(event)) return
      depth.current += 1
      setDragging(true)
    }
    const onLeave = (event: DragEvent) => {
      if (!hasFiles(event)) return
      depth.current -= 1
      if (depth.current <= 0) clearDrag()
    }
    const onOver = (event: DragEvent) => {
      if (hasFiles(event)) event.preventDefault()
    }
    const onDrop = (event: DragEvent) => {
      if (hasFiles(event)) event.preventDefault()
      clearDrag()
    }
    window.addEventListener('dragenter', onEnter)
    window.addEventListener('dragleave', onLeave)
    window.addEventListener('dragover', onOver)
    window.addEventListener('drop', onDrop)
    return () => {
      window.removeEventListener('dragenter', onEnter)
      window.removeEventListener('dragleave', onLeave)
      window.removeEventListener('dragover', onOver)
      window.removeEventListener('drop', onDrop)
    }
  }, [clearDrag])

  return { dragging, clearDrag }
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

function containersFor(projectId: string, jobs: Conversion[], files: ProjectFile[]): LibraryEntry[] {
  const sources = files.filter((file) => file.kind === 'source')
  const sourceById = new Map(sources.map((file) => [file.id, file]))
  const used = new Set<string>()
  const fromJobs = jobs.map((job) => {
    const source = job.fileId ? sourceById.get(job.fileId) : undefined
    if (source) used.add(source.id)
    const fileName = job.fileName || source?.name
    const media = knownMediaName(fileName, source?.name)
    const document = filesFromJob(job, media ?? undefined)[0]
    const date = job.dateUpdated ?? source?.dateCreated
    const status = visibleJobStatus(job)
    const name = faceFileName({
      face: 'library',
      fileName,
      recordingName: source?.name,
      documentName: document?.name,
      date,
    })
    return {
      id: job.id,
      jobId: job.id,
      fileId: job.fileId ?? source?.id,
      fileName,
      recordingName: source?.name,
      name,
      href: WORKSPACE_PATHS.job(projectId, job.id),
      date,
      detail: media && media !== name ? media : '',
      kind: mediaKindOf(fileName, source?.name),
      status,
      showStatus: jobPhase(status) !== 'finished',
    }
  })
  const drafts = sources
    .filter((source) => !used.has(source.id))
    .map((source) => {
      const media = knownMediaName(source.name, source.name)
      const name = faceFileName({
        face: 'video',
        fileName: source.name,
        recordingName: source.name,
        date: source.dateCreated,
      })
      return {
        id: source.id,
        fileId: source.id,
        fileName: source.name,
        recordingName: source.name,
        name,
        href: WORKSPACE_PATHS.file(projectId, source.id),
        date: source.dateCreated,
        detail: media && media !== name ? media : '',
        kind: mediaKindOf(source.name, source.name),
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
  const filesQuery = useProjectFiles(token, projectId)
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
  const [seedFile, setSeedFile] = useState<File | null>(null)
  const [dropError, setDropError] = useState<string | null>(null)
  const { dragging, clearDrag } = useFileDrag()

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
  const listed = containersFor(projectId, jobsQuery.data ?? [], filesQuery.data ?? [])
  const entries = uploadingFile ? [uploadingEntry(uploadingFile), ...listed] : listed
  const pending = jobsQuery.isPending || filesQuery.isPending

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

  function openAddRecording(file?: File) {
    setSeedFile(file ?? null)
    setDropError(null)
    setFormKey((current) => current + 1)
    setStartError(null)
    setAddOpen(true)
  }

  const showDrop = dragging && !addOpen && uploadingFile == null
  const recordingsEmpty = !showDrop && entries.length === 0 && !pending

  return (
    <WorkspacePage
      contentWidth={PAGE_CONTENT_WIDTH}
      header={
        <VStack gap={WORKSPACE_GAP.group} width="100%">
          <ProjectTrail projectId={projectId} />
          <PageHeader
            title={project?.name ?? 'Project'}
            end={
              recordingsEmpty || showDrop ? undefined : (
                <Button label="Add recording" variant="primary" onClick={() => openAddRecording()} />
              )
            }
          />
        </VStack>
      }
    >
      <VStack gap={WORKSPACE_GAP.group}>
        {dropError ? <Banner status="error" title={dropError} /> : null}
        {startError ? (
          <Banner
            status="warning"
            title="The document did not start"
            description={startError}
          />
        ) : null}
        {showDrop ? (
          <FileInput
            label="Add recording"
            placeholder="Drop the recording"
            accept="video/*,audio/*"
            mode="dropzone"
            value={null}
            onChange={(files) => {
              clearDrag()
              const next = droppedFile(files)
              if (next) {
                openAddRecording(next)
                return
              }
              setDropError('Drop a video or audio recording.')
            }}
          />
        ) : null}
        {!showDrop && pending && entries.length === 0 ? <PendingLines variant="rows" /> : null}
        {!showDrop && entries.length === 0 && !pending ? (
          <EmptyState
            title="No recordings yet"
            isCompact
            actions={<Button label="Add recording" variant="primary" onClick={() => openAddRecording()} />}
          />
        ) : null}
        {!showDrop && entries.length > 0 ? (
          <List density="spacious" aria-label="Jobs">
            {entries.map((entry) => (
              <ListItem
                key={entry.id}
                label={entry.name}
                description={<EntryWhen entry={entry} />}
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
        ) : null}
      </VStack>
      <JobPeek
        token={token}
        video={video}
        videoOpen={videoOpen}
        onVideoOpenChange={setVideoOpen}
        conversionId={pipeline?.id ?? null}
        pipelineName={pipeline?.name ?? 'Pipeline'}
        pipelineOpen={pipelineOpen}
        onPipelineOpenChange={setPipelineOpen}
      />
      <AddRecordingDialog
        token={token}
        projectId={projectId}
        isOpen={addOpen}
        formKey={formKey}
        onOpenChange={(open) => {
          if (!open) setSeedFile(null)
          setAddOpen(open)
        }}
        onUploadingFile={setUploadingFile}
        initialFile={seedFile}
        onFinished={(notice) => {
          setUploadingFile(null)
          setAddOpen(false)
          setStartError(notice ?? null)
        }}
      />
    </WorkspacePage>
  )
}
