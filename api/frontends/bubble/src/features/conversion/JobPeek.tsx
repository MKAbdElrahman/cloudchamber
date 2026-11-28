import { useEffect, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'

import { Drawer, DrawerHeader } from '@astryxdesign/core/Drawer'
import { Layout, LayoutContent, VStack } from '@astryxdesign/core/Layout'
import { Banner } from '@astryxdesign/core/Banner'
import { Lightbox } from '@astryxdesign/core/Lightbox'
import { PendingLines, PIPELINE_DRAWER_WIDTH } from '@/shared/layout'
import { WORKSPACE_GAP } from '@/shared/spacing'

import { readFileURL } from './api'
import { JobStages } from './JobStages'
import { JobStatusLabel } from './JobStatusLabel'
import { conversionNotice } from './jobProgress'
import { visibleJobStatus } from './jobStatus'
import { useJobProgress } from './useJobProgress'
import { useConversion } from './useConversion'

export type PeekVideo = {
  name: string
  fileId: string
}

type JobPeekProps = {
  token: string
  video: PeekVideo | null
  videoOpen: boolean
  onVideoOpenChange: (isOpen: boolean) => void
  conversionId: string | null
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
  conversionId,
  pipelineName,
  pipelineOpen,
  onPipelineOpenChange,
}: JobPeekProps) {
  const jobQuery = useConversion(token, conversionId)
  const job = jobQuery.data
  const progress = useJobProgress(token, job)
  const notice = progress ? conversionNotice(progress.phase, progress.stages, progress.reason) : null
  const fileId = videoOpen ? video?.fileId : undefined
  const dialogRef = useRef<HTMLDialogElement>(null)
  const [held, setHeld] = useState<{ fileId: string; src: string } | null>(null)
  const renewing = useRef(false)
  const urlQuery = useQuery({
    queryKey: ['file-url', fileId],
    queryFn: () => readFileURL(token, fileId!),
    enabled: Boolean(fileId),
    staleTime: Infinity,
    refetchOnWindowFocus: false,
  })
  const playbackUrl = urlQuery.data?.url
  const expiresAt = urlQuery.data?.expiresAt
  const renewUrl = urlQuery.refetch
  const videoName = video?.name ?? 'Recording'
  const caption = playbackUrl ? videoName : urlQuery.isPending ? `Opening ${videoName}` : `${videoName} cannot be played from here.`
  const src = held && held.fileId === fileId ? held.src : ''

  if (!videoOpen && held) setHeld(null)
  if (videoOpen && fileId && playbackUrl && held?.fileId !== fileId) setHeld({ fileId, src: playbackUrl })

  useEffect(() => {
    if (!playbackUrl || !src || playbackUrl === src) return
    const videoEl = dialogRef.current?.querySelector('video')
    if (!videoEl) return
    const time = videoEl.currentTime
    const playing = !videoEl.paused
    videoEl.src = playbackUrl
    const restore = () => {
      if (Number.isFinite(time)) videoEl.currentTime = time
      if (playing) void videoEl.play()
    }
    videoEl.addEventListener('loadedmetadata', restore, { once: true })
  }, [playbackUrl, src])

  useEffect(() => {
    if (!expiresAt || !videoOpen) return
    const wait = new Date(expiresAt).getTime() - Date.now() - 60_000
    const timer = window.setTimeout(() => {
      void renewUrl()
    }, Math.max(wait, 0))
    return () => window.clearTimeout(timer)
  }, [expiresAt, videoOpen, renewUrl])

  useEffect(() => {
    const videoEl = dialogRef.current?.querySelector('video')
    if (!videoEl || !videoOpen) return
    const onError = () => {
      if (renewing.current) return
      renewing.current = true
      void renewUrl().finally(() => {
        renewing.current = false
      })
    }
    videoEl.addEventListener('error', onError)
    return () => videoEl.removeEventListener('error', onError)
  }, [playbackUrl, videoOpen, renewUrl])

  return (
    <>
      {video ? (
        <Lightbox
          ref={dialogRef}
          isOpen={videoOpen}
          onOpenChange={onVideoOpenChange}
          media={{
            src,
            alt: videoName,
            type: 'video',
            caption,
          }}
        />
      ) : null}
      {conversionId ? (
        <Drawer isOpen={pipelineOpen} onOpenChange={onPipelineOpenChange} label="Pipeline" width={PIPELINE_DRAWER_WIDTH}>
          <Layout
            header={
              <DrawerHeader
                title={pipelineName}
                subtitle={notice ? undefined : progress?.headline}
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
                  {job && progress ? <JobStages progress={progress} /> : null}
                </VStack>
              </LayoutContent>
            }
          />
        </Drawer>
      ) : null}
    </>
  )
}
