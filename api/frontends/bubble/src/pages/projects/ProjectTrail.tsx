import { ChevronRightIcon } from '@heroicons/react/24/outline'

import { BreadcrumbItem, Breadcrumbs } from '@astryxdesign/core/Breadcrumbs'
import { Icon } from '@astryxdesign/core/Icon'
import { useProject } from '@/features/projects'
import { WORKSPACE_PATHS } from '@/shared/layout'
import { MUTED_ICON_COLOR } from '@/shared/theme'

type ProjectTrailProps = {
  projectId: string
  /** The page you are on. Omit it when the project itself is the current page. */
  current?: string
  /** A step between the project and the current page, such as the job. */
  parent?: { label: string; href: string }
}

/** Projects → this project → anything opened from it. */
export function ProjectTrail({ projectId, current, parent }: ProjectTrailProps) {
  const project = useProject(projectId)
  const name = project?.name ?? 'Project'

  return (
    <Breadcrumbs
      variant="supporting"
      separator={<Icon icon={ChevronRightIcon} size="sm" color={MUTED_ICON_COLOR} />}
      aria-label="Project"
    >
      <BreadcrumbItem href={WORKSPACE_PATHS.home}>Projects</BreadcrumbItem>
      {current ? (
        <BreadcrumbItem href={WORKSPACE_PATHS.project(projectId)}>{name}</BreadcrumbItem>
      ) : null}
      {parent ? <BreadcrumbItem href={parent.href}>{parent.label}</BreadcrumbItem> : null}
      <BreadcrumbItem isCurrent>{current ?? name}</BreadcrumbItem>
    </Breadcrumbs>
  )
}
