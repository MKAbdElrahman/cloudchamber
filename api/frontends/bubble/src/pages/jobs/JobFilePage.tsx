import { useCallback, useEffect, useMemo, useState } from 'react'
import { useLocation, useParams, useSearchParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { useMediaQuery } from '@astryxdesign/core/hooks'
import { HStack, LayoutPanel, VStack } from '@astryxdesign/core/Layout'
import { Banner } from '@astryxdesign/core/Banner'
import { Button } from '@astryxdesign/core/Button'
import { Markdown } from '@astryxdesign/core/Markdown'
import { Text } from '@astryxdesign/core/Text'
import { createMarkdownHeadingLinks } from '@astryxdesign/core/Markdown/plugins'
import { Outline, useOutlineFromMarkdown } from '@astryxdesign/core/Outline'
import { filesFromJob, loadDocumentBody } from '@/features/outputs'
import { JobStatusLabel, createJob, jobPhase, usePipelineJob, visibleJobStatus } from '@/features/pipeline'
import {
  ForeignProjectNotice,
  addProjectJob,
  attachRecordingJob,
  useJobOwner,
  useProjectRecordings,
  useRecording,
} from '@/features/projects'
import { ApiError } from '@/shared/api'
import { PageHeader, PendingLines, SHELL_NARROW_QUERY, WorkspacePage } from '@/shared/layout'
import { WORKSPACE_GAP } from '@/shared/spacing'

import { ProjectTrail } from '../projects/ProjectTrail'
import { faceFileName, knownMediaName, mediaKindOf, mediaLabel } from './jobContainer'
import { JobOpenIcons } from './JobOpenIcons'
import { JobPeek } from './JobPeek'

type JobFilePageProps = {
  token: string
}

type StartErrorState = {
  startError?: string
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
  const { projectId = '', jobId: jobIdParam, recordingId: recordingIdParam } = useParams()
  const location = useLocation()
  const [searchParams, setSearchParams] = useSearchParams()
  const startError = (location.state as StartErrorState | null)?.startError
  const isNarrow = useMediaQuery(SHELL_NARROW_QUERY)
  const queryClient = useQueryClient()
  const recordingsQuery = useProjectRecordings(projectId)
  const recordingQuery = useRecording(projectId, recordingIdParam ?? '')
  const recordings = recordingsQuery.data ?? []
  const resolvedJobId = jobIdParam || (recordingIdParam ? recordingQuery.data?.jobId : undefined) || null
  const owner = useJobOwner(resolvedJobId ?? '')
  const foreign = Boolean(owner.data && owner.data !== projectId)
  const jobQuery = usePipelineJob(token, foreign ? null : resolvedJobId)
  const job = jobQuery.data
  const recording = recordingIdParam
    ? recordingQuery.data ?? undefined
    : recordings.find((item) => item.jobId === jobIdParam) ??
      recordings.find((item) => job?.sourceRef && item.sourceRef === job.sourceRef)
  const sourceRef = job?.sourceRef ?? recording?.sourceRef
  const media = knownMediaName(sourceRef, recording?.name)
  const kind = mediaKindOf(sourceRef, recording?.name)
  const file = job ? filesFromJob(job, media ?? undefined)[0] : undefined
  const recordedAt = job?.dateUpdated ?? job?.dateCreated ?? recording?.dateAdded
  const bodyQuery = useQuery({
    queryKey: ['document-body', file?.ref],
    queryFn: () => {
      if (!file) throw new Error('missing document')
      return loadDocumentBody(file)
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
      if (!recording) throw new Error('Missing recording.')
      return createJob(token, recording.sourceRef, `ui-${crypto.randomUUID()}`)
    },
    onSuccess: async (created) => {
      await addProjectJob(projectId, created.id)
      if (recording) await attachRecordingJob(recording.id, created.id)
      queryClient.setQueryData(['pipeline-job', created.id], created)
      await queryClient.invalidateQueries({ queryKey: ['project-recordings', projectId] })
      if (recordingIdParam) {
        await queryClient.invalidateQueries({ queryKey: ['project-recording', projectId, recordingIdParam] })
      }
      await queryClient.invalidateQueries({ queryKey: ['pipeline-jobs', projectId] })
      await queryClient.invalidateQueries({ queryKey: ['job-project', created.id] })
      setPipelineJobId(created.id)
      setPipelineOpen(true)
    },
  })

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
    (Boolean(recordingIdParam) && recordingQuery.isPending) ||
    (Boolean(resolvedJobId) && !foreign && jobQuery.isPending && !job) ||
    (Boolean(resolvedJobId) && owner.isPending)
  const ready = file != null && bodyQuery.data != null
  const title = faceFileName({
    face: file ? 'document' : 'video',
    sourceRef,
    recordingName: recording?.name,
    documentName: file?.name,
    date: recordedAt,
  })
  const videoName = faceFileName({
    face: 'video',
    sourceRef,
    recordingName: recording?.name,
    date: recordedAt,
  })
  const description = foreign
    ? 'This project cannot show this job.'
    : file && bodyQuery.data?.mocked
      ? bodyQuery.data.format === 'pdf'
        ? 'The PDF is not loaded from storage yet, so this is the document as text.'
        : 'The export is not loaded from storage yet.'
      : file && media && media !== title
        ? media
        : !job && recording
          ? `${mediaLabel(kind)} in this project. Create the document when you are ready.`
          : undefined
  const badgeStatus = job ? visibleJobStatus(job) : undefined

  return (
    <WorkspacePage
      start={
        file && !isNarrow && ready && outline ? (
          <LayoutPanel width="17.5rem" hasDivider>
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
                  {sourceRef ? (
                    <JobOpenIcons
                      kind={kind}
                      hasPipeline={Boolean(pipelineJobId ?? resolvedJobId)}
                      status={badgeStatus}
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
                  {badgeStatus && jobPhase(badgeStatus) !== 'finished' ? <JobStatusLabel status={badgeStatus} /> : null}
                  {!job && recording ? (
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
      <VStack ref={bindArticle} gap={WORKSPACE_GAP.section}>
        {pending ? <PendingLines /> : null}
        {foreign && owner.data && resolvedJobId ? (
          <ForeignProjectNotice ownerId={owner.data} jobId={resolvedJobId} />
        ) : null}
        {startError ? <Banner status="error" title="Could not start the conversion" description={startError} /> : null}
        {create.isError ? (
          <Banner
            status="error"
            title="Could not start the conversion"
            description={create.error instanceof ApiError ? create.error.message : (create.error as Error).message}
          />
        ) : null}
        {jobQuery.isError ? (
          <Banner status="error" title="Could not load the job" description={(jobQuery.error as Error).message} />
        ) : null}
        {!pending && !foreign && !recording && !job ? (
          <Banner status="error" title="Missing job" description="This project does not have that job." />
        ) : null}

        {!foreign && !file && recording && !pending ? (
          <Text type="supporting">The document is not written yet.</Text>
        ) : null}

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
      {!foreign && sourceRef ? (
        <JobPeek
          token={token}
          video={{ name: videoName, sourceRef }}
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
