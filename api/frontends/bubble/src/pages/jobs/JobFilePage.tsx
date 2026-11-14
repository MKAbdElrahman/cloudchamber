import { useCallback, useEffect, useMemo, useState } from 'react'
import { useLocation, useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { useMediaQuery } from '@astryxdesign/core/hooks'
import { HStack, LayoutPanel, VStack } from '@astryxdesign/core/Layout'
import { Banner } from '@astryxdesign/core/Banner'
import { Button } from '@astryxdesign/core/Button'
import { Markdown } from '@astryxdesign/core/Markdown'
import { createMarkdownHeadingLinks } from '@astryxdesign/core/Markdown/plugins'
import { Outline, useOutlineFromMarkdown } from '@astryxdesign/core/Outline'
import { faceFileName, filesFromJob, knownMediaName, loadDocumentBody, mediaKindOf } from '@/features/outputs'
import { JobOpenIcons, JobPeek, JobStages, JobStatusLabel, cancelJob, createJob, isTerminalJobStatus, jobPhase, useJobProgress, usePipelineJob, visibleJobStatus } from '@/features/pipeline'
import { ForeignProjectNotice, ProjectTrail, type ProjectFile } from '@/features/projects'
import { ApiError } from '@/shared/api'
import { PageHeader, PendingLines, SHELL_NARROW_QUERY, SIDE_PANEL_WIDTH, TEXT_CONTENT_WIDTH, WORKSPACE_PATHS, WorkspacePage } from '@/shared/layout'
import { WORKSPACE_GAP } from '@/shared/spacing'

import { useProjectFiles } from '../projects/useProjectFiles'
import { useProjectJobs } from '../projects/useProjectJobs'

type JobFilePageProps = {
  token: string
}

type StartErrorState = {
  startError?: string
}

function sourceFile(files: ProjectFile[], id: string | undefined): ProjectFile | undefined {
  if (!id) return undefined
  return files.find((file) => file.id === id && file.kind === 'source')
}

function nearestScrollParent(node: HTMLElement | null): HTMLElement | null {
  let current = node?.parentElement ?? null
  while (current) {
    const overflow = getComputedStyle(current).overflowY
    if (overflow === 'auto' || overflow === 'scroll') return current
    current = current.parentElement
  }
  return null
}

/** One job file. The document is the page. Icons open the video and the pipeline. */
export function JobFilePage({ token }: JobFilePageProps) {
  const { projectId = '', jobId: jobIdParam, recordingId: recordingIdParam, fileId: fileIdParam } = useParams()
  const sourceId = fileIdParam || recordingIdParam || ''
  const location = useLocation()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const startError = (location.state as StartErrorState | null)?.startError
  const isNarrow = useMediaQuery(SHELL_NARROW_QUERY)
  const queryClient = useQueryClient()
  const filesQuery = useProjectFiles(token, projectId)
  const jobsQuery = useProjectJobs(token, projectId)
  const jobs = jobsQuery.data ?? []
  const matchedJob = sourceId ? jobs.find((item) => item.fileId === sourceId) : jobs.find((item) => item.id === jobIdParam)
  const resolvedJobId = jobIdParam || matchedJob?.id || null
  const jobQuery = usePipelineJob(token, resolvedJobId)
  const job = jobQuery.data ?? matchedJob
  const progress = useJobProgress(job)
  const foreign = Boolean(job?.projectId && job.projectId !== projectId)
  const source = sourceFile(filesQuery.data ?? [], job?.fileId || sourceId)
  const fileName = job?.fileName || source?.name
  const fileId = job?.fileId || source?.id
  const media = knownMediaName(fileName, source?.name)
  const kind = mediaKindOf(fileName, source?.name)
  const file = job ? filesFromJob(job, media ?? undefined)[0] : undefined
  const recordedAt = job?.dateUpdated ?? job?.dateCreated ?? source?.dateCreated
  const bodyQuery = useQuery({
    queryKey: ['document-body', file?.fileId],
    queryFn: () => {
      if (!file) throw new Error('missing document')
      return loadDocumentBody(token, file)
    },
    enabled: file != null,
  })
  const markdown = bodyQuery.data?.markdown ?? ''
  const documentId = file?.id ?? 'document'
  const plugins = useMemo(() => [createMarkdownHeadingLinks({ headingIdPrefix: documentId })], [documentId])
  const items = useOutlineFromMarkdown(markdown, { plugins })
  const [scrollRoot, setScrollRoot] = useState<HTMLElement | null>(null)
  const bindArticle = useCallback((node: HTMLElement | null) => {
    const next = nearestScrollParent(node)
    setScrollRoot((current) => (current === next ? current : next))
  }, [])
  const scrollContainerRef = useMemo(() => ({ current: scrollRoot }), [scrollRoot])
  const outline =
    items.length > 0 ? (
      <Outline items={items} density={isNarrow ? 'compact' : 'default'} label="Contents" scrollContainerRef={scrollContainerRef} />
    ) : null

  const create = useMutation({
    mutationFn: () => {
      if (!fileId) throw new Error('This file has no id.')
      return createJob(token, fileId, `ui-${crypto.randomUUID()}`)
    },
    onSuccess: async (created) => {
      queryClient.setQueryData(['pipeline-job', created.id], created)
      await queryClient.invalidateQueries({ queryKey: ['pipeline-jobs', projectId] })
      await queryClient.invalidateQueries({ queryKey: ['project-files', projectId] })
      navigate(WORKSPACE_PATHS.job(projectId, created.id))
    },
  })

  const canCancel = Boolean(job && !foreign && !isTerminalJobStatus(visibleJobStatus(job)))
  const cancel = useMutation({
    mutationFn: () => {
      if (!job) throw new Error('This run is not open.')
      return cancelJob(token, job.id)
    },
    onSuccess: async (updated) => {
      await queryClient.invalidateQueries({ queryKey: ['pipeline-job', updated.id] })
      await queryClient.invalidateQueries({ queryKey: ['pipeline-jobs'] })
    },
  })
  const isCanceling = cancel.isPending || cancel.isSuccess

  const asked = searchParams.get('open')
  const [videoOpen, setVideoOpen] = useState(asked === 'video')
  const [pipelineOpen, setPipelineOpen] = useState(asked === 'pipeline')
  const [pipelineJobId, setPipelineJobId] = useState<string | null>(asked === 'pipeline' ? resolvedJobId : null)

  useEffect(() => {
    if (asked !== 'video' && asked !== 'pipeline') return
    const next = new URLSearchParams(searchParams)
    next.delete('open')
    setSearchParams(next, { replace: true })
  }, [asked, searchParams, setSearchParams])

  const pending =
    filesQuery.isPending ||
    jobsQuery.isPending ||
    (Boolean(resolvedJobId) && jobQuery.isPending && !job)
  const ready = file != null && bodyQuery.data != null
  const title = faceFileName({
    face: file ? 'document' : 'video',
    fileName,
    recordingName: source?.name,
    documentName: file?.name,
    date: recordedAt,
  })
  const videoName = faceFileName({
    face: 'video',
    fileName,
    recordingName: source?.name,
    date: recordedAt,
  })
  const description = file && bodyQuery.data?.mocked
    ? bodyQuery.data.format === 'pdf'
      ? 'Shown as text.'
      : 'Not loaded from storage.'
    : file && media && media !== title
      ? media
      : undefined
  const badgeStatus = job ? visibleJobStatus(job) : undefined

  const outlinePanel = Boolean(file && !isNarrow && ready && outline)

  return (
    <WorkspacePage
      contentWidth={outlinePanel ? undefined : TEXT_CONTENT_WIDTH}
      start={
        outlinePanel ? (
          <LayoutPanel width={SIDE_PANEL_WIDTH} hasDivider>
            {outline}
          </LayoutPanel>
        ) : undefined
      }
      header={
        <VStack gap={WORKSPACE_GAP.group} width="100%">
          <ProjectTrail projectId={projectId} current={title} />
          <PageHeader
            title={title}
            description={description}
            end={
              foreign || pending ? undefined : (
                <HStack gap={WORKSPACE_GAP.local} vAlign="center">
                  {fileId || job ? (
                    <JobOpenIcons
                      kind={kind}
                      hasPipeline={Boolean(pipelineJobId ?? resolvedJobId)}
                      status={badgeStatus}
                      isCanceling={isCanceling}
                      size="md"
                      onOpen={(target) => {
                        if (target === 'video') {
                          setVideoOpen(true)
                          return
                        }
                        const jobId = pipelineJobId ?? resolvedJobId
                        if (!jobId) return
                        setPipelineJobId(jobId)
                        setPipelineOpen(true)
                      }}
                    />
                  ) : null}
                  {canCancel ? (
                    <Button
                      label={isCanceling ? 'Canceling' : 'Cancel'}
                      variant="secondary"
                      onClick={() => cancel.mutate()}
                      isDisabled={isCanceling}
                    />
                  ) : null}
                  {badgeStatus && jobPhase(badgeStatus) !== 'finished' ? <JobStatusLabel status={badgeStatus} /> : null}
                  {!job && source ? (
                    <Button
                      label="Create document"
                      variant="primary"
                      onClick={() => create.mutate()}
                      isLoading={create.isPending}
                      isDisabled={create.isPending}
                    />
                  ) : null}
                </HStack>
              )
            }
          />
        </VStack>
      }
    >
      <VStack ref={bindArticle} gap={WORKSPACE_GAP.section} width="100%" maxWidth={TEXT_CONTENT_WIDTH}>
        {pending ? <PendingLines /> : null}
        {foreign && job?.projectId && resolvedJobId ? (
          <ForeignProjectNotice ownerId={job.projectId} jobId={resolvedJobId} />
        ) : null}
        {startError ? <Banner status="error" title="Could not start the conversion" description={startError} /> : null}
        {create.isError ? (
          <Banner
            status="error"
            title="Could not start the conversion"
            description={create.error instanceof ApiError ? create.error.message : (create.error as Error).message}
          />
        ) : null}
        {cancel.isError ? (
          <Banner
            status="error"
            title="Could not cancel the run"
            description={cancel.error instanceof ApiError ? cancel.error.message : (cancel.error as Error).message}
          />
        ) : null}
        {jobQuery.isError ? (
          <Banner status="error" title="Could not load the job" description={(jobQuery.error as Error).message} />
        ) : null}
        {!pending && !foreign && !source && !job ? (
          <Banner status="error" title="Missing job" />
        ) : null}

        {!foreign && !file && progress && !pending ? <JobStages progress={progress} /> : null}

        {!foreign && file ? (
          <VStack gap={WORKSPACE_GAP.group}>
            {bodyQuery.isPending ? <PendingLines /> : null}
            {isNarrow && ready ? outline : null}
            {ready ? (
              <Markdown id={documentId} plugins={plugins} headingLevelStart={2}>
                {markdown}
              </Markdown>
            ) : null}
          </VStack>
        ) : null}
      </VStack>
      {!foreign && fileId ? (
        <JobPeek
          token={token}
          video={{ name: videoName, fileId }}
          videoOpen={videoOpen}
          onVideoOpenChange={setVideoOpen}
          pipelineJobId={pipelineJobId ?? resolvedJobId}
          pipelineName={title}
          pipelineOpen={pipelineOpen}
          onPipelineOpenChange={setPipelineOpen}
        />
      ) : null}
    </WorkspacePage>
  )
}
