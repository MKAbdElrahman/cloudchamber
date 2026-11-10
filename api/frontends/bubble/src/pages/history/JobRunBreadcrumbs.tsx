import { ChevronRightIcon, ClockIcon } from '@heroicons/react/24/outline'

import { BreadcrumbItem, Breadcrumbs } from '@astryxdesign/core/Breadcrumbs'
import { Icon } from '@astryxdesign/core/Icon'
import { Text } from '@astryxdesign/core/Text'
import { WORKSPACE_PATHS } from '@/shared/layout'

type JobRunBreadcrumbsProps = {
  currentLabel: string
}

/** Drill-in trail: workspace History → current job run (router link via LinkProvider). */
export function JobRunBreadcrumbs({ currentLabel }: JobRunBreadcrumbsProps) {
  return (
    <Breadcrumbs
      variant="supporting"
      separator={<Icon icon={ChevronRightIcon} size="sm" color="secondary" />}
      aria-label="Job run navigation"
    >
      <BreadcrumbItem
        href={WORKSPACE_PATHS.history}
        startIcon={<Icon icon={ClockIcon} size="sm" color="secondary" />}
      >
        History
      </BreadcrumbItem>
      <BreadcrumbItem isCurrent>
        <Text maxLines={1}>{currentLabel}</Text>
      </BreadcrumbItem>
    </Breadcrumbs>
  )
}
