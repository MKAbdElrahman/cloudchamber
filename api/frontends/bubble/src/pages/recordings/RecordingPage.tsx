import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useLocation, useParams } from 'react-router-dom'

import { Banner } from '@astryxdesign/core/Banner'
import { Button } from '@astryxdesign/core/Button'
import { VStack } from '@astryxdesign/core/Layout'
import { createJob } from '@/features/pipeline'
import { addProjectJob, attachRecordingJob, useRecording } from '@/features/projects'
import { ApiError } from '@/shared/api'
import { PageHeader, PendingLines, WorkspacePage, WORKSPACE_PATHS } from '@/shared/layout'
import { InlineMediaPlayer, playbackUrlFromSourceRef, sourceMediaKind } from '@/shared/media'
import { WORKSPACE_GAP } from '@/shared/spacing'

import { ProjectTrail } from '../projects/ProjectTrail'

type RecordingPageProps = {
  token: string
}

type StartErrorState = {
  startError?: string
}

function kindLabel(kind: ReturnType<typeof sourceMediaKind>): string {
  switch (kind) {
    case 'video':
      return 'Video'
    case 'audio':
      return 'Audio'
    default:
      return 'Recording'
  }
}

/** The recording itself. The conversion starts from here, and only when asked. */
export function RecordingPage({ token }: RecordingPageProps) {
  const { projectId = '', recordingId = '' } = useParams()
  const location = useLocation()
  const startError = (location.state as StartErrorState | null)?.startError
  const queryClient = useQueryClient()
  const recordingQuery = useRecording(projectId, recordingId)
  const recording = recordingQuery.data

  const create = useMutation({
    mutationFn: () => {
      if (!recording) throw new Error('Missing recording.')
      return createJob(token, recording.sourceRef, `ui-${crypto.randomUUID()}`)
    },
    onSuccess: async (job) => {
      await addProjectJob(projectId, job.id)
      await attachRecordingJob(recordingId, job.id)
      queryClient.setQueryData(['pipeline-job', job.id], job)
      await queryClient.invalidateQueries({ queryKey: ['project-recording', projectId, recordingId] })
      await queryClient.invalidateQueries({ queryKey: ['project-recordings', projectId] })
      await queryClient.invalidateQueries({ queryKey: ['pipeline-jobs', projectId] })
    },
  })

  const pending = recordingQuery.isPending
  const title = recording?.name ?? (pending ? 'Recording' : 'Missing recording')
  const kind = recording ? sourceMediaKind(recording.sourceRef) : 'unknown'

  if (!recordingId) {
    return (
      <WorkspacePage header={<PageHeader title="Missing recording" />}>
        <VStack gap={WORKSPACE_GAP.group}>
          <ProjectTrail projectId={projectId} current="Missing recording" />
          <Banner status="error" title="Missing recording" description="No recording id in the URL." />
        </VStack>
      </WorkspacePage>
    )
  }

  return (
    <WorkspacePage
      header={
        <VStack gap={WORKSPACE_GAP.group} width="100%">
          <ProjectTrail projectId={projectId} current={title} />
          <PageHeader
            title={title}
            description={recording ? `${kindLabel(kind)} in this project.` : pending ? undefined : 'This project has no such recording.'}
            end={
              recording ? (
                recording.jobId ? (
                  <Button
                    label="Watch conversion"
                    variant="primary"
                    href={WORKSPACE_PATHS.historyJob(projectId, recording.jobId)}
                  />
                ) : (
                  <Button
                    label="Create document"
                    variant="primary"
                    onClick={() => create.mutate()}
                    isLoading={create.isPending}
                    isDisabled={create.isPending}
                  />
                )
              ) : undefined
            }
          />
        </VStack>
      }
    >
      <VStack gap={WORKSPACE_GAP.group}>
        {pending ? <PendingLines /> : null}
        {recordingQuery.isError ? (
          <Banner status="error" title="Could not open the recording" description={(recordingQuery.error as Error).message} />
        ) : null}
        {!pending && !recording ? (
          <Banner status="error" title="Missing recording" description="This recording is not in the project." />
        ) : null}
        {startError ? <Banner status="error" title="Could not start the conversion" description={startError} /> : null}
        {create.isError ? (
          <Banner
            status="error"
            title="Could not start the conversion"
            description={create.error instanceof ApiError ? create.error.message : (create.error as Error).message}
          />
        ) : null}
        {recording ? (
          <InlineMediaPlayer playbackUrl={playbackUrlFromSourceRef(recording.sourceRef)} kind={kind} />
        ) : null}
      </VStack>
    </WorkspacePage>
  )
}
