import { BreadcrumbItem, Breadcrumbs } from '@astryxdesign/core/Breadcrumbs'
import { useProject } from '@/features/projects'
import { WORKSPACE_PATHS } from '@/shared/layout'

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
    <Breadcrumbs variant="supporting" aria-label="Project">
      <BreadcrumbItem href={WORKSPACE_PATHS.home}>Projects</BreadcrumbItem>
      {current ? (
        <BreadcrumbItem href={WORKSPACE_PATHS.project(projectId)}>{name}</BreadcrumbItem>
      ) : null}
      {parent ? <BreadcrumbItem href={parent.href}>{parent.label}</BreadcrumbItem> : null}
      <BreadcrumbItem isCurrent>{current ?? name}</BreadcrumbItem>
    </Breadcrumbs>
  )
}
