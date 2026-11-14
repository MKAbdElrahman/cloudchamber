import { Banner } from '@astryxdesign/core/Banner'
import { Button } from '@astryxdesign/core/Button'
import { WORKSPACE_PATHS } from '@/shared/layout'

import { useProject } from './useProjects'

type ForeignProjectNoticeProps = {
  ownerId: string
  jobId: string
}

/** Shown when a job URL is opened from a project that does not own it. */
export function ForeignProjectNotice({ ownerId, jobId }: ForeignProjectNoticeProps) {
  const project = useProject(ownerId)
  const name = project?.name ?? 'its project'

  return (
    <Banner
      status="info"
      title="This job is in another project"
      endContent={<Button label={`Open ${name}`} variant="secondary" href={WORKSPACE_PATHS.job(ownerId, jobId)} />}
    />
  )
}
