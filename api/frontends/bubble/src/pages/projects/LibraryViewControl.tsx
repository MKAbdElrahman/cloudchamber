import { Bars4Icon, Squares2X2Icon } from '@heroicons/react/24/outline'

import { Icon } from '@astryxdesign/core/Icon'
import { SegmentedControl, SegmentedControlItem } from '@astryxdesign/core/SegmentedControl'

import type { LibraryView } from './useLibraryView'

type LibraryViewControlProps = {
  value: LibraryView
  onChange: (next: LibraryView) => void
}

/** The file-browser switch: grid or list, icons only. */
export function LibraryViewControl({ value, onChange }: LibraryViewControlProps) {
  return (
    <SegmentedControl
      label="View"
      size="sm"
      value={value}
      onChange={(next) => {
        if (next === 'grid' || next === 'list') onChange(next)
      }}
    >
      <SegmentedControlItem value="grid" label="Grid" isLabelHidden icon={<Icon icon={Squares2X2Icon} size="sm" />} />
      <SegmentedControlItem value="list" label="List" isLabelHidden icon={<Icon icon={Bars4Icon} size="sm" />} />
    </SegmentedControl>
  )
}
