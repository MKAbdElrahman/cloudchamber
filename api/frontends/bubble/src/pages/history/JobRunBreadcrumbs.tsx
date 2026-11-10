import { ProjectTrail } from '../projects/ProjectTrail'

type JobRunBreadcrumbsProps = {
  projectId: string
  currentLabel: string
  /** The file this job belongs to, when the run already produced one. */
  parent?: { label: string; href: string }
}

/** Drill-in trail: Projects → this project → the file, then the job. */
export function JobRunBreadcrumbs({ projectId, currentLabel, parent }: JobRunBreadcrumbsProps) {
  return <ProjectTrail projectId={projectId} current={currentLabel} parent={parent} />
}
