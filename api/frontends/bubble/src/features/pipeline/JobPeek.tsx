import { useEffect, useRef } from 'react'
import { useQuery } from '@tanstack/react-query'

import { Drawer, DrawerHeader } from '@astryxdesign/core/Drawer'
import { Layout, LayoutContent, VStack } from '@astryxdesign/core/Layout'
import { Banner } from '@astryxdesign/core/Banner'
import { Lightbox } from '@astryxdesign/core/Lightbox'
import { PendingLines } from '@/shared/layout'
import { WORKSPACE_GAP } from '@/shared/spacing'

import { readFileURL, reportPlayback } from './api'
import { JobStages } from './JobStages'
import { JobStatusLabel } from './JobStatusLabel'
import { jobOutcomeBanner, visibleJobStatus } from './jobStatus'
import { useJobProgress } from './useJobProgress'
import { usePipelineJob } from './usePipelineJob'

export type PeekVideo = {
  name: string
  fileId: string
}

type JobPeekProps = {
  token: string
  video: PeekVideo | null
  videoOpen: boolean
  onVideoOpenChange: (isOpen: boolean) => void
  pipelineJobId: string | null
  pipelineName: string
  pipelineOpen: boolean
  onPipelineOpenChange: (isOpen: boolean) => void
}

/** The recording opens in a lightbox. The pipeline opens in a drawer. The page underneath stays. */
export function JobPeek({
  token,
  video,
  videoOpen,
  onVideoOpenChange,
  pipelineJobId,
  pipelineName,
  pipelineOpen,
  onPipelineOpenChange,
}: JobPeekProps) {
  const jobQuery = usePipelineJob(token, pipelineJobId)
  const job = jobQuery.data
  const progress = useJobProgress(job)
  const outcome = job ? jobOutcomeBanner(job.status) : null
  const fileId = videoOpen ? video?.fileId : undefined
  const urlQuery = useQuery({
    queryKey: ['file-url', fileId],
    queryFn: () => readFileURL(token, fileId!),
    enabled: Boolean(fileId),
    staleTime: 60_000,
    refetchOnWindowFocus: false,
  })
  const playbackUrl = urlQuery.data?.url
  const traceparent = urlQuery.data?.traceparent
  const videoName = video?.name ?? 'Recording'
  const caption = playbackUrl ? videoName : urlQuery.isPending ? `Opening ${videoName}` : `${videoName} cannot be played from here.`
  const dialogRef = useRef<HTMLDialogElement>(null)

  useEffect(() => {
    const root = dialogRef.current
    if (!fileId || !playbackUrl || !traceparent || !root) return
    const media = root.querySelector('video')
    if (!media) return

    let sent = false
    const started = performance.now()
    const send = (outcome: 'loaded' | 'error') => {
      if (sent) return
      sent = true
      void reportPlayback(token, fileId, traceparent, {
        outcome,
        readyState: media.readyState,
        waitedMs: Math.round(performance.now() - started),
        mediaError: media.error?.code,
      }).catch(() => undefined)
    }
    const onLoaded = () => send('loaded')
    const onError = () => send('error')
    media.addEventListener('loadeddata', onLoaded)
    media.addEventListener('error', onError)
    if (media.readyState >= HTMLMediaElement.HAVE_CURRENT_DATA) send('loaded')
    return () => {
      media.removeEventListener('loadeddata', onLoaded)
      media.removeEventListener('error', onError)
    }
  }, [fileId, playbackUrl, token, traceparent])

  return (
    <>
      {video ? (
        <Lightbox
          ref={dialogRef}
          isOpen={videoOpen}
          onOpenChange={onVideoOpenChange}
          media={{
            src: playbackUrl ?? '',
            alt: videoName,
            type: 'video',
            caption,
          }}
        />
      ) : null}
      {pipelineJobId ? (
        <Drawer isOpen={pipelineOpen} onOpenChange={onPipelineOpenChange} label="Pipeline" width="28rem">
          <Layout
            header={
              <DrawerHeader
                title={pipelineName}
                subtitle={progress?.headline}
                endContent={job ? <JobStatusLabel status={visibleJobStatus(job)} /> : undefined}
                onOpenChange={onPipelineOpenChange}
              />
            }
            content={
              <LayoutContent>
                <VStack gap={WORKSPACE_GAP.group}>
                  {jobQuery.isPending && !job ? <PendingLines /> : null}
                  {jobQuery.isError ? (
                    <Banner status="error" title="Could not load the job" description={(jobQuery.error as Error).message} />
                  ) : null}
                  {job && progress ? (
                    <>
                      {job.errorMessage && outcome ? (
                        <Banner status={outcome.status} title={outcome.title} description={job.errorMessage} />
                      ) : null}
                      <JobStages progress={progress} />
                    </>
                  ) : null}
                </VStack>
              </LayoutContent>
            }
          />
        </Drawer>
      ) : null}
    </>
  )
}
