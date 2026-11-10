import { useParams } from 'react-router-dom'

import { Banner } from '@astryxdesign/core/Banner'
import { Button } from '@astryxdesign/core/Button'
import { HStack, VStack } from '@astryxdesign/core/Layout'
import { filesFromJob } from '@/features/outputs'
import { JobStages, JobStatusLabel, jobOutcomeBanner, useJobProgress, usePipelineJob } from '@/features/pipeline'
import { ForeignProjectNotice, useJobOwner } from '@/features/projects'
import { PageHeader, PendingLines, WorkspacePage, WORKSPACE_PATHS } from '@/shared/layout'
import { fileNameFromSourceRef } from '@/shared/media'
import { WORKSPACE_GAP } from '@/shared/spacing'

import { ProjectTrail } from '../projects/ProjectTrail'

const MEDIA_FILE = /\.(mp4|webm|mov|m4v|mkv|avi|ogv|mp3|wav|ogg|m4a|aac|flac)$/i

type HistoryJobPageProps = {
  token: string
}

function recordingTitle(sourceRef: string | undefined): string {
  const name = sourceRef?.trim() ? fileNameFromSourceRef(sourceRef) : ''
  if (MEDIA_FILE.test(name)) return name
  return 'Conversion'
}

/** One recording becoming one document, as the workflow workers run it. */
export function HistoryJobPage({ token }: HistoryJobPageProps) {
  const { jobId: jobIdParam, projectId = '' } = useParams()
  const jobId = jobIdParam?.trim() ?? ''
  const owner = useJobOwner(jobId)
  const jobQuery = usePipelineJob(token, jobId || null)
  const job = jobQuery.data
  const inProject = owner.data === projectId
  const visibleJob = inProject ? job : undefined
  const progress = useJobProgress(visibleJob)
  const document = visibleJob ? filesFromJob(visibleJob)[0] : undefined
  const outcome = visibleJob ? jobOutcomeBanner(visibleJob.status) : null
  const pending = owner.isPending || (inProject && jobQuery.isPending && !job)
  const title = visibleJob ? recordingTitle(visibleJob.sourceRef) : 'Another project'

  if (!jobId) {
    return (
      <WorkspacePage header={<PageHeader title="Missing job" />}>
        <VStack gap={WORKSPACE_GAP.group}>
          <ProjectTrail projectId={projectId} current="Missing job" />
          <Banner status="error" title="Missing job" description="No job id in the URL." />
        </VStack>
      </WorkspacePage>
    )
  }

  return (
    <WorkspacePage
      header={
        <VStack gap={WORKSPACE_GAP.group} width="100%">
          <ProjectTrail projectId={projectId} current={visibleJob || pending ? title : 'Another project'} />
          <PageHeader
            title={visibleJob || pending ? title : 'Another project'}
            description={visibleJob ? progress?.headline : pending ? undefined : 'This project cannot show this job.'}
            end={
              visibleJob ? (
                <HStack gap={WORKSPACE_GAP.group} vAlign="center">
                  <JobStatusLabel status={visibleJob.status} />
                  {document ? (
                    <Button
                      label="Open document"
                      variant="primary"
                      href={WORKSPACE_PATHS.projectFile(projectId, visibleJob.id, document.kind)}
                    />
                  ) : null}
                </HStack>
              ) : undefined
            }
          />
        </VStack>
      }
    >
      <VStack gap={WORKSPACE_GAP.group}>
        {inProject && jobQuery.isError ? (
          <Banner status="error" title="Could not load job" description={(jobQuery.error as Error).message} />
        ) : null}

        {pending ? <PendingLines /> : null}

        {owner.data && owner.data !== projectId ? (
          <ForeignProjectNotice ownerId={owner.data} jobId={jobId} />
        ) : null}

        {visibleJob && progress ? (
          <VStack gap={WORKSPACE_GAP.group}>
            {visibleJob.errorMessage && outcome ? (
              <Banner status={outcome.status} title={outcome.title} description={visibleJob.errorMessage} />
            ) : null}
            <JobStages progress={progress} />
          </VStack>
        ) : null}
      </VStack>
    </WorkspacePage>
  )
}
