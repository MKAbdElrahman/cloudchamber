import { useNavigate } from 'react-router-dom'

import { VStack } from '@astryxdesign/core/Layout'
import { Banner } from '@astryxdesign/core/Banner'
import { Button } from '@astryxdesign/core/Button'
import { WORKSPACE_PATHS } from '@/shared/layout'
import { WORKSPACE_GAP } from '@/shared/spacing'

import { useProject } from './useProjects'

type ForeignProjectNoticeProps = {
  ownerId: string
  jobId: string
}

/** Shown when a job URL is opened from a project that does not own it. */
export function ForeignProjectNotice({ ownerId, jobId }: ForeignProjectNoticeProps) {
  const navigate = useNavigate()
  const project = useProject(ownerId)
  const name = project?.name ?? 'its project'

  return (
    <VStack gap={WORKSPACE_GAP.group}>
      <Banner
        status="info"
        title="This job is in another project"
        description={`Open it from ${name}. This project cannot show it.`}
      />
      <Button
        label={`Open ${name}`}
        variant="secondary"
        onClick={() => navigate(WORKSPACE_PATHS.historyJob(ownerId, jobId))}
      />
    </VStack>
  )
}
